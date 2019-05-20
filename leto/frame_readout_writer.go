package main

import (
	"io"
	"os"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/proto"
)

type FrameReadoutFileWriter struct {
	f    io.WriteCloser
	quit chan struct{}
}

func NewFrameReadoutWriter(filepath string) (*FrameReadoutFileWriter, error) {
	f, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}

	header := &hermes.Version{
		Major: 0,
		Minor: 1,
	}
	b := proto.NewBuffer(nil)
	err = b.EncodeMessage(header)
	if err != nil {
		return nil, err
	}

	return &FrameReadoutFileWriter{
		f:    f,
		quit: make(chan struct{}),
	}, nil

}

func (w *FrameReadoutFileWriter) Close() error {
	close(w.quit)
	return w.f.Close()
}

func (w *FrameReadoutFileWriter) WriteAll(readout <-chan *hermes.FrameReadout) {
	defer func() {
		w.f.Close()
	}()
	for {
		select {
		case <-w.quit:
			return
		case r, ok := <-readout:
			if ok == false {
				return
			}
			b := proto.NewBuffer(nil)
			b.EncodeMessage(r)
		}
	}
}
