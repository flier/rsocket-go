package frame

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

type Position uint64

const lastReceivedSize = uint64Size

type KeepaliveFrame struct {
	*Header
	LastReceived Position
	Data         []byte
}

func ReadKeepaliveFrame(r io.Reader, header *Header) (frame *KeepaliveFrame, err error) {
	var lastReceived uint64
	var data []byte

	if err = binary.Read(r, binary.BigEndian, &lastReceived); err != nil {
		return
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &KeepaliveFrame{
		header,
		Position(lastReceived),
		data,
	}

	return
}

func (frame *KeepaliveFrame) Size() int {
	return frame.Header.Size() + keepaliveSize + len(frame.Data)
}

func (frame *KeepaliveFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	var n int64

	if n, err = frame.Header.WriteTo(w); err != nil {
		return
	}

	wrote = n

	if err = binary.Write(w, binary.BigEndian, uint64(frame.LastReceived)); err != nil {
		return
	}

	wrote += errorCodeSize

	if n, err = writeExact(w, []byte(frame.Data)); err != nil {
		return
	}

	wrote += n

	return
}
