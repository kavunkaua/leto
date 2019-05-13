package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/ptypes"
)

type WorkloadBalance struct {
	FPS       float64
	Stride    int
	IDsByUUID map[string][]bool
}

func (wb WorkloadBalance) Check() error {
	fids := map[int]string{}

	if len(wb.IDsByUUID) > wb.Stride {
		return fmt.Errorf("WorkloadBalance: more Producer than Stride (%d): %v", wb.Stride, wb.IDsByUUID)
	}

	for puuid, ids := range wb.IDsByUUID {
		if len(ids) != wb.Stride {
			return fmt.Errorf("WorkloadBalance: invalid id definition for producer %s, require %d but got %v", puuid, wb.Stride, ids)
		}
		for i, set := range ids {
			if set == false {
				continue
			}
			if other, ok := fids[i]; ok == true {
				return fmt.Errorf("WorkloadBalance: Producer %s: Frame %d mod[%d] are already produced by %s", puuid, i, wb.Stride, other)
			}
			fids[i] = puuid
		}
	}

	for i := 0; i < wb.Stride; i++ {
		if _, ok := fids[i]; ok == false {
			return fmt.Errorf("WorkloadBalance: No producer set for frame %d mod[%d]", i, wb.Stride)
		}
	}

	return nil
}

func (wb *WorkloadBalance) FrameID(ID int64) int {
	return int(ID % int64(wb.Stride))
}

func (wb *WorkloadBalance) CheckFrame(f *hermes.FrameReadout) (int, error) {
	if len(f.ProducerUuid) == 0 {
		return -1, fmt.Errorf("Received frame has no ProducerUUID")
	}
	ids, ok := wb.IDsByUUID[f.ProducerUuid]
	if ok == false {
		return -1, fmt.Errorf("Invalid ProducerUUID %s", f.ProducerUuid)
	}
	if wb.Stride == 1 {
		return -1, nil
	}

	fid := wb.FrameID(f.FrameID)
	if ok := ids[fid]; ok == false {
		return -1, fmt.Errorf("Producer %s is not meant to produce frame %d mod [%d]", f.ProducerUuid, fid, wb.Stride)
	}
	return fid, nil
}

type ReadoutBuffer []*hermes.FrameReadout

func (r ReadoutBuffer) Len() int {
	return len(r)
}

func (r ReadoutBuffer) Swap(i, j int) {
	r[j], r[i] = r[i], r[j]
}

func (r ReadoutBuffer) Less(i, j int) bool {
	return r[i].FrameID < r[j].FrameID
}

func MergeFrameReadout(wb *WorkloadBalance, inbound <-chan *hermes.FrameReadout, outbound chan<- *hermes.FrameReadout) error {
	defer close(outbound)

	if err := wb.Check(); err != nil {
		return err
	}

	nextFrameToSend := int64(0)
	deadlines := map[int]time.Time{}
	//we reserve a large value, but with tiemout we should have no relocation
	buffer := make(ReadoutBuffer, 10*wb.Stride, 0)
	betweenFrame := time.Duration(1.0e9/wb.FPS) * time.Microsecond
	timeout := time.Duration(wb.Stride+2) * betweenFrame

	logger := log.New(os.Stderr, "[FrameReadoutMerger]:", log.LstdFlags)
	for {
		var timer *time.Timer = nil
		var timeoutC <-chan time.Time = nil
		if len(deadlines) > 0 {
			timer = time.NewTimer(timeout)
			timeoutC = timer.C
		}
		var now time.Time
		select {
		case i, ok := <-inbound:
			if ok == false {
				return nil
			}
			fid, err := wb.CheckFrame(i)
			if err != nil {
				logger.Printf("%s", err)
				continue
			}
			now = time.Now()
			if len(deadlines) == 0 {
				nextFrameToSend = i.FrameID
				for i := fid; i < fid+wb.Stride; i++ {
					ii := i % wb.Stride
					deadlines[ii] = now.Add(time.Duration(i-fid) * betweenFrame).Add(timeout)
				}
			}
			if i.FrameID < nextFrameToSend {
				//we already timeouted the frame
				logger.Printf("Received frame %d, but already sent a timeout", i.FrameID)
				continue
			}

			deadlines[fid] = now.Add(timeout)
			i.ProducerUuid = ""
			buffer = append(buffer, i)

		case t := <-timeoutC:
			now = t
		}
		if timer != nil {
			timer.Stop()
		}
		// we complete the buffer with timeouted values
		for i := nextFrameToSend; i < nextFrameToSend+int64(wb.Stride); i++ {
			mI := wb.FrameID(i)
			if now.After(deadlines[mI]) == true {
				nowPb, _ := ptypes.TimestampProto(now)
				logger.Printf("marking frame %d as timeouted", i)
				buffer = append(buffer, &hermes.FrameReadout{
					Error:   hermes.FrameReadout_PROCESS_TIMEOUT,
					FrameID: i,
					Time:    nowPb,
				})
			}
		}
		//we sort them all
		sort.Sort(buffer)

		//send all frames that we have received or timeouted
		for {
			if len(buffer) == 0 {
				//we are done !!!
				break
			}
			if buffer[0].FrameID < nextFrameToSend {
				logger.Printf("Inconsistent state, next frame is %d, and has %d buffered", nextFrameToSend, buffer[0].FrameID)
				buffer = buffer[1:]
				continue
			}
			if buffer[0].FrameID > nextFrameToSend {
				//we wait for it to arrive or to timeout
				break
			}

			buffer[0].ProducerUuid = ""
			outbound <- buffer[0]
			buffer = buffer[1:]
			nextFrameToSend++
		}

	}

}
