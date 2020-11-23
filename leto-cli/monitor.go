package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/formicidae-tracker/leto"
	"gopkg.in/yaml.v2"
)

var REFRESH_DURATION = 1 * time.Minute

type MonitorCommand struct {
	SlackURL   string `long:"slack-url" description:"URL for the slack service" required:"true"`
	SlackToken string `long:"slack-token" description:"Token for the slack service" required:"true"`

	lister *leto.NodeLister

	currentStatuses map[string]*leto.TrackingConfiguration
}

func (c *MonitorCommand) getStatuses() (map[string]*leto.TrackingConfiguration, error) {
	nodes, err := c.lister.ListNodes()
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	type Result struct {
		Instance string
		Config   *leto.TrackingConfiguration
		Error    error
	}
	results := make(chan Result, 20)
	for _, n := range nodes {
		wg.Add(1)
		go func() {
			defer wg.Done()

			status := leto.Status{}
			err := n.RunMethod("Leto.Status", &leto.NoArgs{}, &status)
			if err != nil {
				results <- Result{Instance: n.Name, Config: nil, Error: err}
				return
			}

			if status.Experiment == nil {
				results <- Result{Instance: n.Name, Config: nil, Error: nil}
			}

			config := &leto.TrackingConfiguration{}
			err = yaml.Unmarshal([]byte(status.Experiment.YamlConfiguration), &config)

			if err != nil {
				results <- Result{Instance: n.Name, Config: nil, Error: err}
				return
			}

			results <- Result{Instance: n.Name, Config: config, Error: nil}

		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	res := make(map[string]*leto.TrackingConfiguration)
	for r := range results {
		if r.Error != nil {
			log.Printf("Could not fetch status for '%s': %s", r.Instance, r.Error)
		} else {
			res[r.Instance] = r.Config
		}
	}

	return res, nil
}

func (c *MonitorCommand) buildEvents() ([]string, error) {
	newStatuses, err := c.getStatuses()
	if err != nil {
		return nil, err
	}
	defer func() {
		c.currentStatuses = newStatuses
	}()

	events := []string{}
	for nodeName, config := range c.currentStatuses {
		newConfig, ok := newStatuses[nodeName]
		if ok == false {
			if config != nil {
				events = append(events, fmt.Sprintf("[ERROR] Experiment '%s' on '%s' apparently ended unexpectedly", config.ExperimentName, nodeName))
			}
			continue
		}

		if newConfig == nil && config != nil {
			events = append(events, fmt.Sprintf("[INFO] Experiment '%s' on '%s' ended hopefully gracefully", config.ExperimentName, nodeName))
		}
	}

	return events, nil
}

func (c *MonitorCommand) Execute(args []string) error {
	c.currentStatuses = nil
	for {
		events, err := c.buildEvents()
		if err != nil {
			//TODO report an error
			log.Printf("Could not build events: %s", err)
		}
		for _, e := range events {
			//TODO publish events
			log.Printf(e)
		}

		time.Sleep(REFRESH_DURATION)
	}

}
