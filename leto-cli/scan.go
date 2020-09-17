package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/formicidae-tracker/leto"
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
	errors := make(chan error, 20)
	wg := sync.WaitGroup{}
	for _, n := range nodes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := leto.Status{}
			err := n.RunMethod("Leto.Status", &leto.NoArgs{}, &status)

			if err != nil {
				errors <- err
				return
			}
			statuses <- Result{Instance: n.Name, Status: status}
		}()
	}
	go func() {
		wg.Wait()
		close(errors)
		close(statuses)
	}()

	for err := range errors {
		log.Printf("Could not fetch status: %s", err)
	}

	formatStr := "%20s | %7s | %20s | %20s | %s\n"
	fmt.Fprintf(os.Stdout, formatStr, "Instance", "Status", "Experiment", "Since", "Links")
	fmt.Fprintf(os.Stdout, "--------------------------------------------------------------------------------\n")
	for r := range statuses {
		s := "Idle"
		exp := "N.A."
		since := "N.A."
		links := ""
		if len(r.Status.Master) != 0 {
			links = "↦ " + strings.TrimPrefix(r.Status.Master, "leto.")
		} else if len(r.Status.Slaves) != 0 {
			sep := "↤ "
			for _, s := range r.Status.Slaves {
				links += sep + strings.TrimPrefix(s, "leto.")
				sep = ",↤ "
			}
		}
		if r.Status.Experiment != nil {
			s = "Running"
			exp = r.Status.Experiment.Configuration.ExperimentName
			since = r.Status.Experiment.Since.Format("Mon Jan 2 15:04:05")
		}
		fmt.Fprintf(os.Stdout, formatStr, r.Instance, s, exp, since, links)
	}
	return nil
}

func init() {
	parser.AddCommand("scan", "scans local network for leto instances", "Uses zeroconf to discover available leto instances and their status over the network", scanCommand)

}
