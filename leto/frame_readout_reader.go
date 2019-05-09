package main

import (
	"io"

	"github.com/formicidae-tracker/hermes"
	"github.com/golang/protobuf/proto"
)

type FrameReadoutReader struct {
	stream io.ReadCloser
}

func NewFrameReadoutReader(s io.ReadCloser) *FrameReadoutReader {
	return &FrameReadoutReader{
		stream: s,
	}
}

func (r *FrameReadoutReader) Close() error {
	return r.stream.Close()
}

func (r *FrameReadoutReader) ReadAll(C chan<- *hermes.FrameReadout, E chan<- error) {
	defer func() {
		close(C)
		close(E)
	}()

	dataSize := make([]byte, 10)
	for {
		idx := 0
		for ; idx < len(dataSize); idx++ {
			n, err := r.stream.Read(dataSize[idx:(idx + 1)])
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
		_, err := io.ReadFull(r.stream, data)
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
