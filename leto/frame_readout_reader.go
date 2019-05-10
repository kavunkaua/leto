package main

import (
	"io"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/proto"
)

func FrameReadoutReadAll(stream io.Reader, C chan<- *hermes.FrameReadout, E chan<- error) {
	defer func() {
		//Do not close C, it is shared by many connections, its a global
		close(E)
	}()

	dataSize := make([]byte, 10)
	for {
		idx := 0
		for ; idx < len(dataSize); idx++ {
			n, err := stream.Read(dataSize[idx:(idx + 1)])
			if err != nil {
				if err != io.EOF {
					E <- err
				}
				return
			}

			if n == 0 {
				idx--
				continue
			}
			if dataSize[idx]&0x80 == 0x00 {
				idx++
				break
			}
		}

		size, n := proto.DecodeVarint(dataSize[0:idx])
		if n == 0 {
			continue
		}
		data := make([]byte, size)
		_, err := io.ReadFull(stream, data)
		if err != nil {
			if err != io.EOF {
				E <- err
			}
			return
		}
		m := &hermes.FrameReadout{}

		err = proto.Unmarshal(data, m)
		if err != nil {
			E <- err
		} else {
			C <- m
		}
	}
}
