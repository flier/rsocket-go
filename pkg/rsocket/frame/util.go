package frame

import (
	"encoding/binary"
	"errors"
	"io"
)

const byteSize = uint8Size
const uint8Size = 1
const uint16Size = 2
const uint24Size = 3
const uint32Size = 4
const uint64Size = 8

// ErrIncomplete is returned when read an incomplete field.
var ErrIncomplete = errors.New("incomplete")

func readExact(r io.Reader, size int) ([]byte, error) {
	buf := make([]byte, size)
	n, err := io.ReadFull(r, buf)

	if err != nil {
		return nil, err
	}

	if n != size {
		return nil, ErrIncomplete
	}

	return buf, nil
}

func writeExact(w io.Writer, buf []byte) (wrote int64, err error) {
	var n int

	n, err = w.Write(buf)

	if err != nil {
		return
	}

	if n != len(buf) {
		return 0, ErrIncomplete
	}

	wrote = int64(n)

	return
}

func writeByte(w io.Writer, b byte) error {
	_, err := writeExact(w, []byte{b})

	return err
}

func readUInt24(r io.Reader, order binary.ByteOrder) (n uint32, err error) {
	var buf []byte

	if buf, err = readExact(r, uint24Size); err != nil {
		return
	}

	if order == binary.BigEndian {
		n = uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
	} else {
		n = uint32(buf[2])<<16 | uint32(buf[1])<<8 | uint32(buf[0])
	}

	return n, nil
}

func writeUInt24(w io.Writer, order binary.ByteOrder, v uint32) (n int64, err error) {
	var buf []byte

	if order == binary.BigEndian {
		buf = []byte{
			byte(v >> 16),
			byte(v >> 8),
			byte(v),
		}
	} else {
		buf = []byte{byte(v),
			byte(v >> 8),
			byte(v >> 16),
		}
	}

	return writeExact(w, buf)
}
