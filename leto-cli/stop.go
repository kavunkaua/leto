package main

import (
	"fmt"

	"github.com/formicidae-tracker/leto"
)

type StopCommand struct {
	Instance string `short:"I" long:"instance" decsription:"host to start the tracking" required:"true"`
}

var stopCommand = &StopCommand{}

func (c *StopCommand) Execute([]string) error {
	n, ok := nodes[c.Instance]
	if ok == false {
		return fmt.Errorf("Could not find node '%s'", c.Instance)
	}

	resp := &leto.Response{}
	if err := n.RunMethod("Leto.StopTracking", &leto.NoArgs{}, resp); err != nil {
		return err
	}
	return resp.ToError()
}

func init() {
	parser.AddCommand("stop", "stops tracking on a specified node", "Stops the tracking on a specified node", stopCommand)
}
