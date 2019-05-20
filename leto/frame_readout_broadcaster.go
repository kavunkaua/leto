package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/proto"
)

func BroadcastFrameReadout(address string, readouts <-chan *hermes.FrameReadout) {
	m := NewRemoteManager()

	mx := sync.RWMutex{}
	outgoing := map[int]chan []byte{}

	go func() {
		for r := range readouts {
			b := proto.NewBuffer(nil)
			b.EncodeMessage(r)
			mx.RLock()
			for _, o := range outgoing {
				o <- b.Bytes()
			}
			mx.RUnlock()
		}
		m.Close()
		mx.Lock()
		defer mx.Unlock()
		for _, o := range outgoing {
			close(o)
		}
	}()
	i := 0
	log.Printf("Broadcasting on %s", address)
	m.Listen(address, func(c net.Conn) {
		defer c.Close()
		logger := log.New(os.Stderr, fmt.Sprintf("[broadcast/%s]", c.RemoteAddr().String()), log.LstdFlags)

		b := proto.NewBuffer(nil)
		header := &hermes.Version{Major: 0, Minor: 5}
		b.EncodeMessage(header)

		_, err := c.Write(b.Bytes())
		if err != nil {
			logger.Printf("could not write header: %s", err)
			return
		}
		o := make(chan []byte, 10)
		mx.Lock()
		idx := i
		outgoing[idx] = o
		i += 1
		mx.Unlock()
		for buf := range o {
			_, err := c.Write(buf)
			if err != nil {
				logger.Printf("Could not write frame: %s", err)
				mx.Lock()
				close(o)
				delete(outgoing, idx)
				mx.Unlock()
			}
		}
	}, func() {
		log.Printf("Stopped broadcasting on %s", address)
	})
}
