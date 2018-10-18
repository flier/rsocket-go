package frame

import (
	"encoding/binary"
	"encoding/hex"
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

func (w *Writer) writeFrameLength(size int) (wrote int64, err error) {
	return writeUInt24(w.Writer, binary.BigEndian, uint32(size))
}

// WriteFrame write a frame to w.
func (w *Writer) WriteFrame(frame Frame) (wrote int64, err error) {
	frameSize := frame.Size()

	wrote, err = w.writeFrameLength(frameSize)

	if err != nil {
		return
	}

	var n int64

	n, err = frame.WriteTo(w)

	if err != nil {
		return
	}

	wrote += n

	w.Debug("write frame",
		zap.Stringer("type", frame.Type()),
		zap.Int64("size", n))

	if WriteFrameDumper != nil {
		dumper := hex.Dumper(WriteFrameDumper)

		writeUInt24(dumper, binary.BigEndian, uint32(frameSize))

		frame.WriteTo(dumper)

		WriteFrameDumper.Write([]byte("\n"))
	}

	return
}
