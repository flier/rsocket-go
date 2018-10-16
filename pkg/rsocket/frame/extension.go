package frame

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

const extTypeSize = uint32Size

// ExtensionFrame used to extend more frame types as well as extensions.
type ExtensionFrame struct {
	*Header
	ExtendedType uint32
	Data         []byte
}

func readExtensionFrame(r io.Reader, header *Header) (frame *ExtensionFrame, err error) {
	var extType uint32
	var data []byte

	if err = binary.Read(r, binary.BigEndian, &extType); err != nil {
		return
	}
	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &ExtensionFrame{
		header,
		extType,
		data,
	}

	return
}

// Size returns the encoded size of the frame.
func (ext *ExtensionFrame) Size() int {
	return ext.Header.Size() + extTypeSize + len(ext.Data)
}

// WriteTo writes the encoded frame to w.
func (ext *ExtensionFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = ext.Header.WriteTo(w); err != nil {
		return
	}

	var n int64

	if err = binary.Write(w, binary.BigEndian, ext.ExtendedType); err != nil {
		return
	}

	wrote += uint32Size

	if n, err = writeExact(w, ext.Data); err != nil {
		return
	}

	wrote += n

	return
}
