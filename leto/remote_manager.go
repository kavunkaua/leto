package main

import (
	"log"
	"net"
	"sync"

	"github.com/formicidae-tracker/hermes"
)

type RemoteManager struct {
	mx          sync.Mutex
	quit        chan struct{}
	connections []net.Conn
	listener    net.Listener
}

func NewRemoteManager() *RemoteManager {
	return &RemoteManager{}
}

func (m *RemoteManager) Stop() {
	m.mx.Lock()
	defer m.mx.Unlock()
	if m.quit != nil {
		close(m.quit)
	}
	if m.listener != nil {
		m.listener.Close()
	}
	for i, c := range m.connections {
		if c == nil {
			continue
		}
		c.Close()
		m.connections[i] = nil
	}
}

func (m *RemoteManager) Listen(address string, readouts chan<- *hermes.FrameReadout) {
	wg := sync.WaitGroup{}

	defer func() {
		wg.Wait()
		close(readouts)
	}()

	m.mx.Lock()
	var err error
	m.listener, err = net.Listen("tcp", address)
	if err != nil {
		m.mx.Unlock()
		m.listener = nil
		return
	}
	m.quit = make(chan struct{})
	m.mx.Unlock()

	for {
		conn, err := m.listener.Accept()
		if err != nil {
			select {
			case <-m.quit:
				return
			default:
				log.Printf("accept: %s", err)
				continue
			}
		}

		m.mx.Lock()
		errors := make(chan error)
		go func(remote string) {
			for e := range errors {
				log.Printf("[remote/%s]: %s", remote, e)
			}
		}(conn.RemoteAddr().String())
		wg.Add(1)

		m.connections = append(m.connections, conn)
		go func(idx int) {
			FrameReadoutReadAll(conn, readouts, errors)
			m.mx.Lock()
			defer m.mx.Unlock()
			if m.connections[idx] != nil {
				m.connections[idx] = nil
			}
			wg.Done()
		}(len(m.connections) - 1)
		m.mx.Unlock()
	}

}
