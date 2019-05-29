package leto

import (
	"io"

	"github.com/golang/protobuf/proto"
)

func ReadDelimitedMessage(stream io.Reader, pb proto.Message) (bool, error) {
	dataSize := make([]byte, 10)
	idx := 0
	for ; idx < len(dataSize); idx++ {
		n, err := stream.Read(dataSize[idx:(idx + 1)])
		if err != nil {
			return false, err
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
	if n == 0 || size == 0 {
		return false, nil
	}
	data := make([]byte, size)
	_, err := io.ReadFull(stream, data)
	if err != nil {
		return false, err
	}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return false, err
	}
	return true, nil
}
