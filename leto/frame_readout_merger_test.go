package main

import (
	"sync"
	"time"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/ptypes"
	. "gopkg.in/check.v1"
)

type FrameReadoutMergerSuite struct{}

var _ = Suite(&FrameReadoutMergerSuite{})

func (s *FrameReadoutMergerSuite) TestEnd2End(c *C) {
	testdata := []struct {
		T time.Duration
		R *hermes.FrameReadout
	}{
		{
			T: 0 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      0,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 9 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      1,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 21 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      2,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 29 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      3,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 47 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      5,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 49 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      4,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 69 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      7,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 90 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      9,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 100 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      10,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 110 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      11,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 120 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      12,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 130 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      13,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 140 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      14,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 150 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      15,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 160 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      16,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 170 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      17,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 180 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      18,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
		{
			T: 190 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      19,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "bar",
			},
		},
		{
			T: 191 * time.Microsecond,
			R: &hermes.FrameReadout{
				FrameID:      6,
				Error:        hermes.FrameReadout_NO_ERROR,
				ProducerUuid: "foo",
			},
		},
	}

	expected := []*hermes.FrameReadout{
		&hermes.FrameReadout{
			FrameID:      0,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      1,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      2,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      3,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      4,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      5,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      6,
			Error:        hermes.FrameReadout_PROCESS_TIMEOUT,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      7,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      8,
			Error:        hermes.FrameReadout_PROCESS_TIMEOUT,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      9,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      10,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      11,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      12,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      13,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      14,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      15,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      16,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      17,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      18,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
		&hermes.FrameReadout{
			FrameID:      19,
			Error:        hermes.FrameReadout_NO_ERROR,
			ProducerUuid: "",
		},
	}

	inbound := make(chan *hermes.FrameReadout)
	outbound := make(chan *hermes.FrameReadout)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		start := time.Now()
		for _, d := range testdata {
			time.Sleep(start.Add(d.T).Sub(start))
			d.R.Time, _ = ptypes.TimestampProto(start.Add(d.T))
			inbound <- d.R
		}
		close(inbound)
		wg.Done()
	}()
	wb := &WorkloadBalance{
		FPS:    10000.0,
		Stride: 2,
		IDsByUUID: map[string][]bool{
			"foo": []bool{true, false},
			"bar": []bool{false, true},
		},
	}

	go func() {
		MergeFrameReadout(wb, inbound, outbound)
		wg.Done()
	}()

	i := 0
	for r := range outbound {
		c.Check(r.FrameID, Equals, expected[i].FrameID)
		c.Check(r.Error, Equals, expected[i].Error)
		c.Check(r.ProducerUuid, Equals, expected[i].ProducerUuid)
		i += 1
	}
	c.Check(i, Equals, len(expected))
	wg.Wait()

}
