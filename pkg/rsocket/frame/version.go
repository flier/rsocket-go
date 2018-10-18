package frame

import (
	"encoding/binary"
	"io"
)

// Version of protocol
type Version struct {
	Major uint16
	Minor uint16
}

// V1 version protocol
var V1 = Version{1, 0}

func (version *Version) lessThanOrEquals(other Version) bool {
	if version.Major < other.Major {
		return true
	}

	if version.Major == other.Major && version.Minor <= other.Minor {
		return true
	}

	return false
}

// Size returns the encoded size of the version.
func (version *Version) Size() int {
	return 4
}

// WriteTo writes the encoded frame to w.
func (version *Version) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.BigEndian, uint16(version.Major)); err != nil {
		return
	}
	if err = binary.Write(w, binary.BigEndian, uint16(version.Minor)); err != nil {
		return
	}

	n = int64(version.Size())

	return
}
