package frame

import (
	"encoding/binary"
	"io"
)

// Metadata of request or connection.
type Metadata []byte

func readMetadata(r io.Reader) (metadata Metadata, err error) {
	var len uint32

	len, err = readUInt24(r, binary.BigEndian)

	if err != nil {
		return
	}

	metadata, err = readExact(r, int(len))

	return
}

// Size returns the encoded size of the metadata.
func (metadata Metadata) Size() int {
	if metadata == nil {
		return 0
	}

	return uint24Size + len(metadata)
}

// WriteTo writes the metadata to w.
func (metadata Metadata) WriteTo(w io.Writer) (wrote int64, err error) {
	_, err = writeUInt24(w, binary.BigEndian, uint32(len(metadata)))

	if err != nil {
		return
	}

	var n int64

	n, err = writeExact(w, []byte(metadata))

	if err != nil {
		return
	}

	wrote = uint24Size + n

	return
}
