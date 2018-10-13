package frame

import (
	"encoding/binary"
	"io"
)

type Writer struct {
	io.Writer
	version *Version
}

func (w *Writer) writeFrameLength(size int) (wrote int64, err error) {
	if w.version.LessThanOrEquals(V1) {
		err = binary.Write(w.Writer, binary.BigEndian, uint32(size))

		if err != nil {
			return
		}

		wrote = uint32Size

		return
	}

	return writeUInt24(w.Writer, binary.BigEndian, uint32(size))
}

func (w *Writer) WriteFrame(frame Frame) (wrote int64, err error) {
	wrote, err = w.writeFrameLength(frame.Size())

	if err != nil {
		return
	}

	var n int64

	n, err = frame.WriteTo(w.Writer)

	if err != nil {
		return
	}

	wrote += n

	return
}
