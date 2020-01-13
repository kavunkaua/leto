package main

import (
	"github.com/formicidae-tracker/leto"
)

type StopCommand struct {
	Instance string `short:"I" long:"instance" decsription:"host to start the tracking" required:"true"`
}

var stopCommand = &StopCommand{}

func (c *StopCommand) Execute([]string) error {
	resp := &leto.Response{}
	args := &leto.TrackingStop{}
	if _, _, err := leto.RunForHost(c.Instance, "Leto.StopTracking", args, resp); err != nil {
		return err
	}
	return resp.ToError()
}

func init() {
	parser.AddCommand("stop", "stops tracking on a specified node", "Stops the tracking on a specified node", stopCommand)

}
