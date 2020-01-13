package main

import (
	"github.com/formicidae-tracker/leto"
)

type LinkingOptions struct {
	Master  string `short:"M" long:"master" decsription:"instance to be the master" required:"true"`
	Slave   string `short:"S" long:"slave" decsription:"instance to be the slave" required:"true"`
	command string
}

var linkCommand = &LinkingOptions{command: "Leto.Link"}
var unlinkCommand = &LinkingOptions{command: "leto.Unlink"}

func (c *LinkingOptions) Execute(args []string) error {
	argsL := leto.Link{
		Master: c.Master,
		Slave:  c.Slave,
	}
	resp := &leto.Response{}
	if _, _, err := leto.RunForHost(c.Master, c.command, argsL, resp); err != nil {
		return err
	}
	return resp.ToError()
}

func init() {
	_, err := parser.AddCommand("link", "link two nodes together", "link a master to a slave", linkCommand)
	if err != nil {
		panic(err.Error())
	}
	_, err = parser.AddCommand("unlink", "unlink two linked nodes", "unlink a master and one of its slaves", unlinkCommand)
	if err != nil {
		panic(err.Error())
	}

}
