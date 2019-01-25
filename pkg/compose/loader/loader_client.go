package loader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/stacks/pkg/compose/schema"
	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/pkg/errors"
)

// TODO - this file needs some significant refactoring
// * Refactor to just a simple/thin helper to read in one or more compose files
// * Refactor the env var parsing logic so it tracks unsubstituted variables, but doesn't try to replace them
//   during the parsing
// * Move the brunt of the parsing logic to something that can be wired up via an API route, and create
//   a type that's a list of compose files, and the return would be the spec plug env vars that need to be filled in

// LoadComposefile parse the composefile specified in the cli and returns its Config and version.
func LoadComposefile(composefiles []string) (*composetypes.Config, error) {
	configDetails, err := getConfigDetails(composefiles)
	if err != nil {
		return nil, err
	}

	dicts := getDictsFrom(configDetails.ConfigFiles)
	config, err := Load(configDetails)
	if err != nil {
		if fpe, ok := err.(*ForbiddenPropertiesError); ok {
			return nil, errors.Errorf("Compose file contains unsupported options:\n\n%s\n",
				propertyWarnings(fpe.Properties))
		}

		return nil, err
	}

	unsupportedProperties := GetUnsupportedProperties(dicts...)
	if len(unsupportedProperties) > 0 {
		fmt.Printf("Ignoring unsupported options: %s\n\n",
			strings.Join(unsupportedProperties, ", "))
	}

	deprecatedProperties := GetDeprecatedProperties(dicts...)
	if len(deprecatedProperties) > 0 {
		fmt.Printf("Ignoring deprecated options:\n\n%s\n\n",
			propertyWarnings(deprecatedProperties))
	}
	return config, nil
}

func getDictsFrom(configFiles []composetypes.ConfigFile) []map[string]interface{} {
	dicts := []map[string]interface{}{}

	for _, configFile := range configFiles {
		dicts = append(dicts, configFile.Config)
	}

	return dicts
}

func propertyWarnings(properties map[string]string) string {
	var msgs []string
	for name, description := range properties {
		msgs = append(msgs, fmt.Sprintf("%s: %s", name, description))
	}
	sort.Strings(msgs)
	return strings.Join(msgs, "\n\n")
}

func getConfigDetails(composefiles []string) (composetypes.ConfigDetails, error) {
	var details composetypes.ConfigDetails

	if len(composefiles) == 0 {
		return details, errors.New("no composefile(s)")
	}

	if composefiles[0] == "-" && len(composefiles) == 1 {
		workingDir, err := os.Getwd()
		if err != nil {
			return details, err
		}
		details.WorkingDir = workingDir
	} else {
		absPath, err := filepath.Abs(composefiles[0])
		if err != nil {
			return details, err
		}
		details.WorkingDir = filepath.Dir(absPath)
	}

	var err error
	details.ConfigFiles, err = loadConfigFiles(composefiles)
	if err != nil {
		return details, err
	}
	// Take the first file version (2 files can't have different version)
	details.Version = schema.Version(details.ConfigFiles[0].Config)
	details.Environment, err = buildEnvironment(os.Environ())
	return details, err
}

func buildEnvironment(env []string) (map[string]string, error) {
	result := make(map[string]string, len(env))
	for _, s := range env {
		// if value is empty, s is like "K=", not "K".
		if !strings.Contains(s, "=") {
			return result, errors.Errorf("unexpected environment %q", s)
		}
		kv := strings.SplitN(s, "=", 2)
		result[kv[0]] = kv[1]
	}
	return result, nil
}

func loadConfigFiles(filenames []string) ([]composetypes.ConfigFile, error) {
	var configFiles []composetypes.ConfigFile

	for _, filename := range filenames {
		configFile, err := loadConfigFile(filename)
		if err != nil {
			return configFiles, err
		}
		configFiles = append(configFiles, *configFile)
	}

	return configFiles, nil
}

func loadConfigFile(filename string) (*composetypes.ConfigFile, error) {
	var bytes []byte
	var err error

	bytes, err = ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config, err := ParseYAML(bytes)
	if err != nil {
		return nil, err
	}

	return &composetypes.ConfigFile{
		Filename: filename,
		Config:   config,
	}, nil
}
