package frame

import (
	"bytes"
	"encoding/binary"
	"io"
)

type Reader struct {
	io.Reader
	version *Version
	buf     *bytes.Buffer
}

func NewReader(r io.Reader, version *Version) *Reader {
	return &Reader{r, version, nil}
}

func (r *Reader) frameLengthSize() int {
	if r.version.LessThanOrEquals(V1) {
		return uint32Size
	}

	return uint24Size
}

func (r *Reader) ReadFrame() (frame Frame, err error) {
	for {
		if r.buf == nil {
			r.buf = bytes.NewBuffer(make([]byte, 0, r.frameLengthSize()))
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

		if r.buf.Cap() == r.frameLengthSize() {
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

			frame, err = readFrame(r.buf, header)

			r.buf = nil

			if err == ErrUnknownType && header.CanIgnore() {
				continue
			}

			return
		}
	}
}
