package main

import (
	"fmt"

	"github.com/formicidae-tracker/leto"
	"gopkg.in/yaml.v2"
)

type LastExperimentLogCommand struct {
	Instance string `short:"I" long:"instance" decsription:"instance to start the tracking" required:"true"`
}

var lastExperimentCommand = &LastExperimentLogCommand{}

func (c *LastExperimentLogCommand) Execute(args []string) error {
	n, ok := nodes[c.Instance]
	if ok == false {
		return fmt.Errorf("Could not find node '%s'", c.Instance)
	}

	log := leto.ExperimentLog{}
	if err := n.RunMethod("Leto.LastExperimentLog", &leto.NoArgs{}, &log); err != nil {
		return err
	}

	config := leto.TrackingConfiguration{}
	err := yaml.Unmarshal([]byte(log.YamlConfiguration), &config)
	if err != nil {
		return fmt.Errorf("Could not parse YAML configuration: %s", err)
	}

	fmt.Printf("Experiment Name: %s\n", config.ExperimentName)
	fmt.Printf("Experiment Local Output Dir: %s\n", log.ExperimentDir)
	fmt.Printf("Experiment Start Date: %s\n", log.Start)
	fmt.Printf("Experiment End Date: %s\n", log.End)
	fmt.Printf("Artemis returned an error: %t\n", log.HasError)

	fmt.Printf("=== Experiment YAML Configuration START ===\n")
	fmt.Println(log.YamlConfiguration)
	fmt.Printf("=== Experiment YAML Configuration END ===\n")
	fmt.Printf("=== Artemis INFO LOG ===\n")
	fmt.Println(string(log.Log))
	fmt.Printf("=== Artemis INFO LOG END ===\n")

	return nil
}

func init() {
	_, err := parser.AddCommand("last-experiment-log", "queries the last experiment log on the node", "Queries the last experiment log on the node", lastExperimentCommand)
	if err != nil {
		panic(err.Error())
	}
}
