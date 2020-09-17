package main

import (
	"github.com/formicidae-tracker/leto"
)

type StopCommand struct {
	Args struct {
		Node Nodename
	} `positional-args:"yes" required:"yes"`
}

var stopCommand = &StopCommand{}

func (c *StopCommand) Execute([]string) error {
	n, err := c.Args.Node.GetNode()
	if err != nil {
		return err
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
