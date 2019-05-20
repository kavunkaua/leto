package main

import (
	"github.com/formicidae-tracker/leto"
)

type StartCommand struct {
	Instance string `short:"I" long:"instance" decsription:"instance to start the tracking" required:"true"`
	Config   leto.TrackingStart
}

var startCommand = &StartCommand{}

func (c *StartCommand) Execute(args []string) error {
	resp := &leto.Response{}
	if _, _, err := RunForHost(c.Instance, "Leto.StartTracking", &(c.Config), resp); err != nil {
		return err
	}
	return resp.ToError()
}

func init() {
	parser.AddCommand("start", "starts tracking on a speciied node", "Starts the tracking on a specified node", startCommand)

}
