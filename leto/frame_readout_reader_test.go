package main

import (
	"bytes"
	"testing"

	"github.com/formicidae-tracker/hermes"
	. "gopkg.in/check.v1"

	"github.com/golang/protobuf/proto"
	google_protobuf "github.com/golang/protobuf/ptypes/timestamp"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type FrameReadoutReaderSuite struct{}

var _ = Suite(&FrameReadoutReaderSuite{})

func (s *FrameReadoutReaderSuite) TestHelloWorld(c *C) {
	testdata := []*hermes.FrameReadout{
		&hermes.FrameReadout{
			Timestamp:    0,
			FrameID:      0,
			Ants:         nil,
			Time:         &google_protobuf.Timestamp{},
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "foo",
		},
		&hermes.FrameReadout{
			Timestamp:    10000,
			FrameID:      1,
			Ants:         nil,
			Time:         &google_protobuf.Timestamp{},
			Error:        hermes.FrameReadout_PROCESS_OVERFLOW,
			ProducerUuid: "foo",
		},
	}

	b := proto.NewBuffer(nil)

	for _, m := range testdata {
		b.EncodeMessage(m)
	}

	C := make(chan *hermes.FrameReadout)
	E := make(chan error)

	data := bytes.NewBuffer(b.Bytes())

	go FrameReadoutReadAll(data, C, E)

	i := 0
	for {
		select {
		case m, ok := <-C:
			if ok == false {
				C = nil
				break
			}
			c.Assert(i < len(testdata), Equals, true)
			c.Check(m, DeepEquals, testdata[i])
			i += 1
		case err, ok := <-E:
			if ok == false {
				E = nil
				break
			}
			c.Check(err, IsNil)

		}
		if E == nil && C == nil {
			break
		}
	}
	c.Check(i, Equals, len(testdata))

}
