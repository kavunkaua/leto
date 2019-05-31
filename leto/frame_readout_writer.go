package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/proto"
)

type FrameReadoutFileWriter struct {
	period   time.Duration
	basename string
	lastname string
	file     *os.File
	gzip     *gzip.Writer
	logger   *log.Logger
	quit     chan struct{}
}

func NewFrameReadoutWriter(filepath string) (*FrameReadoutFileWriter, error) {

	return &FrameReadoutFileWriter{
		period:   2 * time.Hour,
		basename: filepath,
		quit:     make(chan struct{}),
		logger:   log.New(os.Stderr, fmt.Sprintf("[file/%s] ", filepath), log.LstdFlags),
	}, nil

}

func (w *FrameReadoutFileWriter) openFile(filepath string) error {
	var err error
	w.file, err = os.Create(filepath)
	if err != nil {
		return err
	}
	w.gzip = gzip.NewWriter(w.file)

	header := &hermes.Header{
		Type: hermes.Header_File,
		Version: &hermes.Version{
			Major: 0,
			Minor: 5,
		},
		Previous: w.lastname,
	}
	w.lastname = filepath

	b := proto.NewBuffer(nil)
	err = b.EncodeMessage(header)
	if err != nil {
		return err
	}

	return nil
}

func (w *FrameReadoutFileWriter) closeFiles(nextFile string) {
	footer := &hermes.Footer{}
	if len(nextFile) > 0 {
		footer.Next = nextFile
	} else {
		footer.EOS = true
	}
	if w.gzip != nil && w.file != nil {
		b := proto.NewBuffer(nil)
		if err := b.EncodeMessage(footer); err != nil {
			w.logger.Printf("Could not encode footer: %s", err)
		} else {
			if _, err := w.gzip.Write(b.Bytes()); err != nil {
				w.logger.Printf("Could not write footer: %s", err)
			}
		}
	}

	if w.gzip != nil {
		if err := w.gzip.Close(); err != nil {
			w.logger.Printf("could not close gzipper: %s", err)
		}
		w.gzip = nil
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			w.logger.Printf("could not close '%s': %s", w.lastname, err)
		}
		w.file = nil
	}

}

func (w *FrameReadoutFileWriter) Close() error {
	close(w.quit)
	w.closeFiles("")
	return nil
}

func (w *FrameReadoutFileWriter) WriteAll(readout <-chan *hermes.FrameReadout) {
	ticker := time.NewTicker(w.period)
	defer func() {
		ticker.Stop()
		w.closeFiles("")
	}()

	closeNext := false
	nextName, _, err := FilenameWithoutOverwrite(w.basename)
	if err != nil {
		w.logger.Printf("Could not find unique name: %s", err)
		return
	}

	for {
		select {
		case <-ticker.C:
			closeNext = true
		case <-w.quit:
			return
		case r, ok := <-readout:
			if ok == false {
				return
			}
			if w.file == nil {
				err := w.openFile(nextName)
				if err != nil {
					w.logger.Printf("Could not create file '%s': %s", nextName, err)
				}
				return
			}

			r.ProducerUuid = ""
			if closeNext == true {
				r.EOS = true
			}
			b := proto.NewBuffer(nil)
			if err := b.EncodeMessage(r); err != nil {
				w.logger.Printf("Could not encode message: %s", err)
			}
			_, err := w.gzip.Write(b.Bytes())
			if err != nil {
				w.logger.Printf("Could not write message: %s", err)
				return
			}
			if closeNext == true {
				nextName, _, err = FilenameWithoutOverwrite(w.basename)
				if err != nil {
					w.logger.Printf("Could not find unique name: %s", err)
				}
				w.closeFiles(nextName)
			}

		}
	}
}
