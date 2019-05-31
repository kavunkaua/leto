package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/formicidae-tracker/hermes"
	"github.com/formicidae-tracker/leto"
)

type DisplayFrameReadoutCommand struct {
	Instance string `short:"I" long:"instance" decsription:"instance to read frame from" required:"true"`
}

var displayFrameReadoutCommand = &DisplayFrameReadoutCommand{}

type FrameReadoutDisplayer struct {
	Errors  int
	Frames  int
	AntsIDs map[uint32]bool
}

func (d *FrameReadoutDisplayer) DisplayFrameReadout(h *hermes.FrameReadout) string {
	d.Frames += 1
	currentAnts := len(h.Ants)
	if h.Error != hermes.FrameReadout_NO_ERROR {
		d.Errors += 1
	} else {
		for _, a := range h.Ants {
			d.AntsIDs[a.ID] = true
		}
	}

	return fmt.Sprintf("%s %06d %06d %04d/%04d   ", time.Now().Format("15:04:05"), d.Frames, d.Errors, currentAnts, len(d.AntsIDs))
}

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

	version := &hermes.Header{}
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

	d := FrameReadoutDisplayer{
		AntsIDs: make(map[uint32]bool),
	}

	fmt.Fprintf(os.Stdout, "Time    Frames  Errors Cur/Tot Ants\n\n")
	for {
		m := &hermes.FrameReadout{}
		ok, err := leto.ReadDelimitedMessage(conn, m)
		if err != nil {
			return err
		}
		if ok == true {
			fmt.Fprintf(os.Stdout, "\033[A%s\n", d.DisplayFrameReadout(m))
		}
	}

	return nil
}

func init() {
	parser.AddCommand("display-frame-readout", "display current frame readout", "Displays frame readout information on the standard output", displayFrameReadoutCommand)

}
