package main

import (
	"context"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"
	"time"

	"github.com/formicidae-tracker/leto"
	"github.com/grandcat/zeroconf"
)

type ScanCommand struct {
}

var scanCommand = &ScanCommand{}

type Result struct {
	Instance string
	Status   leto.Status
}

func (c *ScanCommand) Execute(args []string) error {
	statuses := make(chan Result, 20)

	resolver, err := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)
	errors := make(chan error)
	go func(results <-chan *zeroconf.ServiceEntry) {
		defer func() { close(errors); close(statuses) }()

		for e := range results {

			client, err := rpc.DialHTTP("tcp",
				fmt.Sprintf("%s:%d", strings.TrimSuffix(e.HostName, "."), e.Port))
			if err != nil {
				errors <- err
				continue
			}
			status := &leto.Status{}
			err = client.Call("Leto.Status", status, status)
			if err != nil {
				errors <- err
			}
			statuses <- Result{
				Instance: e.Instance,
				Status:   *status,
			}
		}
	}(entries)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err = resolver.Browse(ctx, "_leto._tcp", "local.", entries)
		if err != nil {
			log.Printf("Could not browse for leto instances: %s", err)
		}
		err, ok := <-errors
		if ok == true {
			log.Printf("Could not browse for leto instances: %s", err)
		}
	}()

	formatStr := "%20s | %7s | %20s | %s\n"
	fmt.Fprintf(os.Stdout, formatStr, "Instance", "Status", "Experiment", "Since")
	fmt.Fprintf(os.Stdout, "--------------------------------------------------------------------------------\n")
	for r := range statuses {
		s := "Idle"
		exp := "N.A."
		since := "N.A."
		if r.Status.Running == true {
			s = "Running"
			exp = r.Status.ExperimentName
			since = r.Status.Since.Format("Mon Jan 2 15:04:05")
		}
		fmt.Fprintf(os.Stdout, formatStr, r.Instance, s, exp, since)
	}
	return nil
}

func init() {
	parser.AddCommand("scan", "scans local network for leto instances", "Uses zeroconf to discover available leto instances and their status over the network", scanCommand)

}