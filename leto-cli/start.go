package main

import (
	"fmt"

	"github.com/formicidae-tracker/leto"
)

type StartCommand struct {
	Instance string `short:"I" long:"instance" decsription:"instance to start the tracking" required:"true"`
	Config   leto.TrackingConfiguration
}

var startCommand = &StartCommand{}

func (c *StartCommand) Execute(args []string) error {
	config := &(c.Config)
	if len(args) >= 1 {
		fileConfig, err := leto.ReadConfiguration(args[0])
		if err != nil {
			return err
		}
		if err := leto.MergeConfiguration(config, fileConfig); err != nil {
			return fmt.Errorf("Could not merge file and commandline configuration: %s", err)
		}
	}

	resp := &leto.Response{}
	if _, _, err := RunForHost(c.Instance, "Leto.StartTracking", config, resp); err != nil {
		return err
	}
	return resp.ToError()
}

func init() {
	_, err := parser.AddCommand("start", "starts tracking on a speciied node", "Starts the tracking on a specified node", startCommand)
	if err != nil {
		panic(err.Error())
	}
}
