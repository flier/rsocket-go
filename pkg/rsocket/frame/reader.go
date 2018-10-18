package frame

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"

	"go.uber.org/zap"
)

const frameLengthSize = uint24Size

// ReadFrameDumper dumps read frame
var ReadFrameDumper io.Writer

// Reader implements convenience methods for reading frames from a RSocket connection.
type Reader struct {
	*zap.Logger
	io.Reader
	buf *bytes.Buffer
}

// NewReader returns a new Reader reading from r.
func NewReader(logger *zap.Logger, r io.Reader) *Reader {
	return &Reader{logger.Named("r"), r, nil}
}

// ReadFrame reads a frame from a RSocket connection.
func (r *Reader) ReadFrame() (frame Frame, err error) {
	for {
		if r.buf == nil {
			r.buf = bytes.NewBuffer(make([]byte, 0, frameLengthSize))
		}

		if r.buf.Len() < r.buf.Cap() {
			buf := make([]byte, r.buf.Cap()-r.buf.Len())

			var n int

			n, err = r.Reader.Read(buf)

			if err != nil {
				return
			}

			n, err = r.buf.Write(buf[:n])

			if err != nil {
				return
			}

			if n < len(buf) {
				continue
			}
		}

		if r.buf.Cap() == frameLengthSize {
			var len uint32

			len, err = readUInt24(r.buf, binary.BigEndian)

			if err != nil {
				return
			}

			r.buf.Grow(int(len))
		} else {
			var header *Header

			header, err = readHeader(r.Reader)

			if err != nil {
				return
			}

			r.Debug("read frame",
				zap.Stringer("type", header.Type()),
				zap.Binary("data", r.buf.Bytes()))

			if ReadFrameDumper != nil {
				hex.Dumper(ReadFrameDumper).Write(r.buf.Bytes())

				WriteFrameDumper.Write([]byte("\n"))
			}

			frame, err = readFrame(r.buf, header)

			r.buf = nil

			if err == ErrUnknownFrameType && header.CanIgnore() {
				continue
			}

			return
		}
	}
}
