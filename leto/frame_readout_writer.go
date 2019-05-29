package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/proto"
)

type FrameReadoutFileWriter struct {
	f      io.WriteCloser
	logger *log.Logger
	quit   chan struct{}
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
		f:      f,
		quit:   make(chan struct{}),
		logger: log.New(os.Stderr, fmt.Sprintf("[file:%s] ", filepath), log.LstdFlags),
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
			r.ProducerUuid = ""
			b := proto.NewBuffer(nil)
			if err := b.EncodeMessage(r); err != nil {
				w.logger.Printf("Could not encode message: %s", err)
			}
			_, err := w.f.Write(b.Bytes())
			if err != nil {
				w.logger.Printf("Could not write message: %s", err)
				return
			}
		}
	}
}
