package main

import (
	"fmt"
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

func (m *RemoteManager) Close() error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if m.quit != nil {
		close(m.quit)
	}
	var res error
	if m.listener != nil {
		if err := m.listener.Close(); err != nil {
			if res != nil {
				res = fmt.Errorf("%s;%s", res, err)
			} else {
				res = err
			}
		}
	}
	for i, c := range m.connections {
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil {
			if res != nil {
				res = fmt.Errorf("%s;%s", res, err)
			} else {
				res = err
			}
		}
		m.connections[i] = nil
	}
	return res
}

func (m *RemoteManager) Listen(address string, readouts chan<- *hermes.FrameReadout) error {

	m.mx.Lock()
	if m.listener != nil {
		m.mx.Unlock()
		return fmt.Errorf("Already started, wait for former to close properly the output channel")
	}
	var err error
	m.listener, err = net.Listen("tcp", address)
	if err != nil {
		m.mx.Unlock()
		m.listener = nil
		return err
	}
	m.quit = make(chan struct{})
	m.mx.Unlock()

	wg := sync.WaitGroup{}

	defer func() {
		wg.Wait()
		m.mx.Lock()
		defer func() {
			m.mx.Unlock()
			close(readouts)
		}()

		m.connections = nil
		m.listener = nil
		m.quit = nil
	}()

	for {
		conn, err := m.listener.Accept()
		if err != nil {
			select {
			case <-m.quit:
				return nil
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
