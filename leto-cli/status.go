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
	status := leto.Status{}
	if _, _, err := leto.RunForHost(c.Instance, "Leto.Status", &leto.NoArgs{}, &status); err != nil {
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
	fmt.Printf("State: Running Experiment '%s' since %s\n", status.Experiment.Configuration.ExperimentName, status.Experiment.Since)
	fmt.Printf("Experiment Local Output Directory: %s\n", status.Experiment.ExperimentDir)
	fmt.Printf("=== Experiment YAML Configuration START ===\n")
	yamlConfig, err := yaml.Marshal(&status.Experiment.Configuration)
	if err != nil {
		return fmt.Errorf("Could not generate YAML configuration: %s", err)
	}
	fmt.Printf("%s\n", yamlConfig)
	fmt.Printf("=== Experiment YAML Configuration END ===\n")
	return nil
}

func init() {
	_, err := parser.AddCommand("status", "queries the full status on a speciied node", "Queries the complete status on a specified node", statusCommand)
	if err != nil {
		panic(err.Error())
	}
}
