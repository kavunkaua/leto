package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/formicidae-tracker/hermes"
	"github.com/formicidae-tracker/leto"
)

func Execute() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("Neead a file to read")
	}
	basedir := filepath.Dir(os.Args[1])
	prev := filepath.Base(os.Args[1])
	file, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer file.Close()
	uncomp, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer uncomp.Close()
	header := hermes.Header{}

	ok, err := leto.ReadDelimitedMessage(uncomp, &header)
	if ok == false {
		return fmt.Errorf("Could not read version of file")
	}
	if err != nil {
		return err
	}
	if len(header.Previous) != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: this file seems to have previous data '%s'", header.Previous)
	}

	for {
		line := hermes.FileLine{}
		ok, err = leto.ReadDelimitedMessage(uncomp, &line)
		if ok == false {
			return fmt.Errorf("Could not read file line")
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if line.Readout != nil {
			fmt.Fprintf(os.Stdout, "frame: %d, time: %s, ants: %d, error: %d\n", line.Readout.FrameID, line.Readout.Time.String(), len(line.Readout.Ants), line.Readout.Error)
		}
		if line.Footer == nil {
			continue
		}

		if len(line.Footer.Next) == 0 {
			//endof stream
			return nil
		}
		uncomp.Close()
		file.Close()
		next := filepath.Join(basedir, line.Footer.Next)
		fmt.Fprintf(os.Stderr, "opening next file '%s'\n", next)
		file, err = os.Open(next)
		if err != nil {
			return err
		}
		uncomp, err = gzip.NewReader(file)
		if err != nil {
			return err
		}
		ok, err = leto.ReadDelimitedMessage(uncomp, &header)
		if ok == false {
			return fmt.Errorf("Could not read header")
		}
		if err != nil {
			return err
		}
		if header.Previous != prev {
			return fmt.Errorf("previous file for '%s' is '%s' but '%s' expected", next, header.Previous, prev)
		}
		prev = line.Footer.Next
	}

	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Fatalf("Unhandled error: %s", err)
	}
}