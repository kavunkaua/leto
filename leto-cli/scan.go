package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/atuleu/go-tablifier"
	"github.com/formicidae-tracker/leto"
	"gopkg.in/yaml.v2"
)

type ScanCommand struct {
}

var scanCommand = &ScanCommand{}

type Result struct {
	Instance string
	Status   leto.Status
}

func (r Result) running() int {
	if r.Status.Experiment == nil {
		return 0
	}
	return 1
}

type ResultTableLine struct {
	Node       string
	Status     string
	Experiment string
	Since      string
	Links      string
}

func (c *ScanCommand) Execute(args []string) error {

	statuses := make(chan Result, 20)
	errors := make(chan error, 20)
	wg := sync.WaitGroup{}
	for _, nlocal := range nodes {
		n := nlocal
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

	lines := make([]ResultTableLine, 0, len(nodes))

	now := time.Now()

	for r := range statuses {
		line := ResultTableLine{
			Node:       strings.TrimPrefix(r.Instance, "leto."),
			Status:     "Idle",
			Experiment: "N.A.",
			Since:      "N.A.",
		}
		if len(r.Status.Master) != 0 {
			line.Links = "↦ " + strings.TrimPrefix(r.Status.Master, "leto.")
		} else if len(r.Status.Slaves) != 0 {
			sep := "↤ "
			for _, s := range r.Status.Slaves {
				line.Links += sep + strings.TrimPrefix(s, "leto.")
				sep = ",↤ "
			}
		}
		if r.Status.Experiment != nil {
			line.Status = "Running"
			config := leto.TrackingConfiguration{}
			yaml.Unmarshal([]byte(r.Status.Experiment.YamlConfiguration), &config)
			line.Experiment = config.ExperimentName
			ellapsed := now.Sub(r.Status.Experiment.Since).Round(time.Second)
			line.Since = fmt.Sprintf("%s", ellapsed)
		}
		lines = append(lines, line)
	}

	sort.Slice(lines, func(i, j int) bool {
		if lines[i].Status == lines[j].Status {
			return lines[i].Node < lines[j].Node
		}
		return lines[i].Status == "Running"
	})

	tablifier.Tablify(lines)

	return nil
}

func init() {
	parser.AddCommand("scan", "scans local network for leto instances", "Uses zeroconf to discover available leto instances and their status over the network", scanCommand)

}
