package interfaces

import (
	"errors"
	"strings"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/types"
)

// FakeFeatures contains composable test feature APIs
type FakeFeatures interface {
	SortedIDs() []string
	SpecifyError(errorKey string, err error)
	SpecifyKeyPrefix(KeyPrefix string)
	MarkInputForError(errorKey string, input interface{}, ops ...string)
	constructErrorMark(operation string) string
}

// error definitions to reuse
var (
	FakeNotFound      = errdefs.NotFound(errors.New("service not found"))
	FakeInvalidArg    = errdefs.InvalidParameter(errors.New("not valid"))
	FakeUnavailable   = errdefs.Unavailable(errors.New("not available"))
	FakeUnimplemented = errdefs.Unavailable(errors.New("UNIMPLEMENTED"))
)

// temporary constant arguments in order to track their uses
const (
	DefaultGetServiceArg2    = false
	DefaultCreateServiceArg2 = ""
	DefaultCreateServiceArg3 = false
)

// FakeGetStackIDFromLabelFilter takes a filters.Args and determines if it includes
// a filter for StackLabel. If so, it returns the Stack ID specified by the
// label and true. If not, it returns emptystring and false.
func FakeGetStackIDFromLabelFilter(args filters.Args) (string, bool) {
	labelfilters := args.Get("label")
	// there should only be 1 string here, anything else is not supported
	if len(labelfilters) != 1 {
		return "", false
	}

	// we now have a filter that is in one of two forms:
	// SomeKey or SomeKey=SomeValue
	// We split on the =. If we get 1 string back, it means there is no =, and
	// therefore no value specified for the label.
	kvPair := strings.SplitN(labelfilters[0], "=", 2)
	if len(kvPair) != 2 {
		return "", false
	}

	// make sure the key is StackLabel
	if kvPair[0] != types.StackLabel {
		return "", false
	}

	// don't return true if the value is emptystring. there's no reason
	// emptystring wouldn't be a valid, except that i'm pretty sure allowing it
	// to be a valid ID in this context would invite bugs.
	if kvPair[1] == "" {
		return "", false
	}

	return kvPair[1], true
}
