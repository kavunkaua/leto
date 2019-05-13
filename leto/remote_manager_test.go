package main

import (
	"net"
	"sync"
	"time"

	"github.com/formicidae-tracker/hermes"
	. "gopkg.in/check.v1"
)

type RemoteManagerSuite struct{}

var _ = Suite(&RemoteManagerSuite{})

func (s *RemoteManagerSuite) TestManager(c *C) {
	readouts := make(chan *hermes.FrameReadout)
	closed := make(chan struct{})
	quit := make(chan struct{})

	m := NewRemoteManager()

	go m.Listen(":12345", readouts)
	go func() {
		conn, err := net.Dial("tcp", "localhost:12345")
		c.Assert(err, IsNil)
		time.Sleep(10 * time.Millisecond)
		conn.Close()
		close(closed)
	}()
	<-closed
	wg := sync.WaitGroup{}
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			conn, err := net.Dial("tcp", "localhost:12345")
			c.Assert(err, IsNil)
			<-quit
			conn.Close()
			wg.Done()
		}()
	}

	time.Sleep(70 * time.Millisecond)
	m.mx.Lock()
	c.Check(len(m.connections), Equals, 3)
	m.mx.Unlock()
	close(quit)
	wg.Wait()
	c.Check(m.Close(), IsNil)
	for _, conn := range m.connections {
		c.Check(conn, IsNil)
	}
	_, ok := <-readouts
	c.Check(ok, Equals, false)

}
