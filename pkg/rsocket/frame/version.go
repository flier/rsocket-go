package frame

import (
	"encoding/binary"
	"io"
)

type Version struct {
	Major uint16
	Minor uint16
}

var V1 = Version{1, 0}

func (version *Version) LessThanOrEquals(other Version) bool {
	if version.Major < other.Major {
		return true
	}

	if version.Major == other.Major && version.Minor <= other.Minor {
		return true
	}

	return false
}

func (version *Version) Size() int {
	return 4
}

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
