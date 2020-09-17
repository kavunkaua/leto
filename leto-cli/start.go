package main

import (
	"fmt"

	"github.com/formicidae-tracker/leto"
	"github.com/jessevdk/go-flags"
)

type StartCommand struct {
	Config leto.TrackingConfiguration

	Args struct {
		Node       Nodename
		ConfigFile *flags.Filename
	} `positional-args:"yes"`
}

var startCommand = &StartCommand{}

func (c *StartCommand) Execute(args []string) error {
	n, err := c.Args.Node.GetNode()
	if err != nil {
		return err
	}

	config := &(c.Config)
	if len(args) >= 1 {
		fileConfig, err := leto.ReadConfiguration(args[0])
		if err != nil {
			return err
		}
		if err := fileConfig.Merge(config); err != nil {
			return fmt.Errorf("Could not merge file and commandline configuration: %s", err)
		}
		config = fileConfig
	}
	config.Loads = nil
	resp := &leto.Response{}
	if err := n.RunMethod("Leto.StartTracking", config, resp); err != nil {
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
