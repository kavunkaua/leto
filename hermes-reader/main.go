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
		m := hermes.FrameReadout{}
		ok, err = leto.ReadDelimitedMessage(uncomp, &m)
		if ok == false {
			return fmt.Errorf("Could not read frame readout")
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		fmt.Fprintf(os.Stdout, "frame: %d, time: %s, ants: %d, error: %d\n", m.FrameID, m.Time.String(), len(m.Ants), m.Error)
		if m.EOS == false {
			continue
		}
		footer := hermes.Footer{}
		ok, err := leto.ReadDelimitedMessage(uncomp, &footer)
		if err != nil {
			return err
		}
		if ok == false {
			return fmt.Errorf("Could not read footer")
		}
		if footer.EOS == true {
			//endof stream
			return nil
		}
		uncomp.Close()
		file.Close()
		next := filepath.Join(basedir, footer.Next)
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
		prev = footer.Next
	}

	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Fatalf("Unhandled error: %s", err)
	}
}
