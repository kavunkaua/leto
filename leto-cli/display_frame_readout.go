package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/formicidae-tracker/hermes"
	"github.com/formicidae-tracker/leto"
)

type DisplayFrameReadoutCommand struct {
	Instance string `short:"I" long:"instance" decsription:"instance to read frame from" required:"true"`
}

var displayFrameReadoutCommand = &DisplayFrameReadoutCommand{}

func (c *DisplayFrameReadoutCommand) Execute(args []string) error {
	resp := &leto.Status{}
	hostname, _, err := RunForHost(c.Instance, "Leto.Status", resp, resp)
	if err != nil {
		return fmt.Errorf("Could not find  instance '%s'", c.Instance)
	}
	if resp.Running == false {
		return fmt.Errorf("'%s' is not running", c.Instance)
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", hostname, leto.ARTEMIS_OUT_PORT))
	if err != nil {
		return fmt.Errorf("Could not connect to '%s': %s", c.Instance, err)
	}

	version := &hermes.Version{}
	ok, err := leto.ReadDelimitedMessage(conn, version)
	if err != nil {
		conn.Close()
		return err
	}
	if ok == false {
		conn.Close()
		return fmt.Errorf("Did not receive an expected version header")
	}

	go func() {
		sigint := make(chan os.Signal)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		conn.Close()
	}()

	for {
		m := &hermes.FrameReadout{}
		ok, err := leto.ReadDelimitedMessage(conn, m)
		if err != nil {
			return err
		}
		if ok == true {
			log.Printf("%+v", m)
		}
	}

	return nil
}

func init() {
	parser.AddCommand("display-frame-readout", "display current frame readout", "Displays frame readout information on the command line ", displayFrameReadoutCommand)

}
