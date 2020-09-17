package main

import (
	"fmt"

	"github.com/formicidae-tracker/leto"
	"gopkg.in/yaml.v2"
)

type StatusCommand struct {
	Instance string `short:"I" long:"instance" decsription:"instance to start the tracking" required:"true"`
}

var statusCommand = &StatusCommand{}

func (c *StatusCommand) Execute(args []string) error {
	n, ok := nodes[c.Instance]
	if ok == false {
		return fmt.Errorf("Could not find node '%s'", c.Instance)
	}

	status := leto.Status{}
	if err := n.RunMethod("Leto.Status", &leto.NoArgs{}, &status); err != nil {
		return err
	}

	fmt.Printf("Node: %s\n", c.Instance)
	if len(status.Master) == 0 {
		fmt.Printf("Type: Master\nSlaves : %s\n", status.Slaves)
	} else {
		fmt.Printf("Type: Slave\nMaster : %s\n", status.Master)
	}

	if status.Experiment == nil {
		fmt.Printf("State: Idle\n")
		return nil
	}
	config := leto.TrackingConfiguration{}
	err := yaml.Unmarshal([]byte(status.Experiment.YamlConfiguration), &config)
	if err != nil {
		return err
	}

	fmt.Printf("State: Running Experiment '%s' since %s\n", config.ExperimentName, status.Experiment.Since)
	fmt.Printf("Experiment Local Output Directory: %s\n", status.Experiment.ExperimentDir)
	fmt.Printf("=== Experiment YAML Configuration START ===\n")
	fmt.Println(status.Experiment.YamlConfiguration)
	fmt.Printf("=== Experiment YAML Configuration END ===\n")
	return nil
}

func init() {
	_, err := parser.AddCommand("status", "queries the full status on a speciied node", "Queries the complete status on a specified node", statusCommand)
	if err != nil {
		panic(err.Error())
	}
}
