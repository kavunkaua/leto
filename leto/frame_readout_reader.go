package main

import (
	"io"

	"github.com/formicidae-tracker/hermes"
	"github.com/formicidae-tracker/leto"
)

func FrameReadoutReadAll(stream io.Reader, C chan<- *hermes.FrameReadout, E chan<- error) {
	defer func() {
		//Do not close C, it is shared by many connections, its a global
		close(E)
	}()

	for {
		m := &hermes.FrameReadout{}
		ok, err := leto.ReadDelimitedMessage(stream, m)
		if err != nil {
			if err == io.EOF {
				return
			}
			E <- err
		}
		if ok == true {
			C <- m
		}
	}
}
