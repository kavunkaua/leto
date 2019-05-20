package main

import (
	"log"

	"github.com/jessevdk/go-flags"
)

type Options struct {
}

var opts = &Options{}

var parser = flags.NewParser(opts, flags.Default)

func Execute() error {
	_, err := parser.Parse()
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
