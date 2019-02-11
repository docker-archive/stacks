package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/sirupsen/logrus"

	"github.com/docker/stacks/pkg/controller/standalone"
)

var cmdServer = cli.Command{
	Name:   "server",
	Usage:  "Starts the Standalone Stacks API server and reconciler",
	Action: RunStandaloneServer,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
		cli.StringFlag{
			Name:  "docker-socket",
			Usage: "Path to the Docker socket (default: /var/run/docker.sock)",
			Value: "/var/run/docker.sock",
		},
		cli.IntFlag{
			Name:  "port",
			Usage: "Port on which to expose the stacks API (default: 2375)",
			Value: 2375,
		},
	},
}

// RunStandaloneServer parses CLI arguments and runs the StandaloneServer
// method from the standalone package.
func RunStandaloneServer(c *cli.Context) error {
	return standalone.Server(standalone.ServerOptions{
		Debug:            c.Bool("debug"),
		DockerSocketPath: c.String("docker-socket"),
		ServerPort:       c.Int("port"),
	})
}

func main() {
	app := cli.NewApp()
	app.Name = "Stacks Standalone Controller"
	app.Usage = "Docker Stacks Standalone Controller"
	app.Commands = []cli.Command{
		cmdServer,
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
		os.Exit(1)
	}
}
