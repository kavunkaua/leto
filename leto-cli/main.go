package main

import (
	"fmt"
	"log"

	"github.com/formicidae-tracker/leto"
	"github.com/jessevdk/go-flags"
)

type Options struct {
}

var nodes map[string]leto.Node

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func Execute() error {
	var err error
	nodes, err = leto.NewNodeLister().ListNodes()
	if err != nil {
		return fmt.Errorf("Could not list nodes on local network: %s", err)
	}

	_, err = parser.Parse()
	if ferr, ok := err.(*flags.Error); ok == true && ferr.Type == flags.ErrHelp {
		err = nil
	}
	return err
}

func main() {
	if err := Execute(); err != nil {
		log.Fatalf("Unhandled error")
	}
}
