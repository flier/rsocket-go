package frame

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"go.uber.org/zap"
)

// WriteFrameDumper dumps wrote frame
var WriteFrameDumper io.Writer

// A Writer implements convenience methods for writing frames to a RSocket connection.
type Writer struct {
	*zap.Logger
	io.Writer
}

// NewWriter returns a new Writer wrting to w.
func NewWriter(logger *zap.Logger, w io.Writer) *Writer {
	return &Writer{logger.Named("w"), w}
}

// WriteFrame write a frame to w.
func (w *Writer) WriteFrame(frame Frame) (wrote int64, err error) {
	frameSize := frame.Size()
	buf := bytes.NewBuffer(make([]byte, 0, frameLengthSize+frameSize))

	wrote, err = writeUInt24(buf, binary.BigEndian, uint32(frameSize))

	if err != nil {
		return
	}

	var n int64

	n, err = frame.WriteTo(buf)

	if err != nil {
		return
	}

	wrote += n

	_, err = w.Write(buf.Bytes())

	if err != nil {
		return
	}

	w.Debug("write frame",
		zap.Stringer("type", frame.Type()),
		zap.Int64("size", n))

	if WriteFrameDumper != nil {
		fmt.Println(hex.Dump(buf.Bytes()))
	}

	return
}
