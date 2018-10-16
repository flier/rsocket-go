package frame

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"time"
)

const timeToLiveSize = uint32Size
const numberOfRequestsSize = uint32Size

// LeaseFrame sent by Responder to grant the ability to send requests.
type LeaseFrame struct {
	*Header
	TimeToLive       time.Duration
	NumberOfRequests uint32
	Metadata         Metadata
}

func readLeaseFrame(r io.Reader, header *Header) (frame *LeaseFrame, err error) {
	var ttl, numOfReqs uint32
	var metadata Metadata

	if err = binary.Read(r, binary.BigEndian, &ttl); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &numOfReqs); err != nil {
		return
	}
	if header.HasMetadata() {
		if metadata, err = ioutil.ReadAll(r); err != nil {
			return
		}
	}

	frame = &LeaseFrame{
		header,
		time.Duration(ttl) * time.Millisecond,
		numOfReqs,
		metadata,
	}

	return
}

// Size returns the encoded size of the frame.
func (lease *LeaseFrame) Size() int {
	return lease.Header.Size() + timeToLiveSize + numberOfRequestsSize + len(lease.Metadata)
}

// WriteTo writes the encoded frame to w.
func (lease *LeaseFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	var n int64

	if n, err = lease.Header.WriteTo(w); err != nil {
		return
	}

	wrote = n

	if err = binary.Write(w, binary.BigEndian, uint32(lease.TimeToLive/time.Millisecond)); err != nil {
		return
	}
	if err = binary.Write(w, binary.BigEndian, uint32(lease.NumberOfRequests)); err != nil {
		return
	}

	wrote += timeToLiveSize + numberOfRequestsSize

	if lease.HasMetadata() {
		if n, err = lease.Metadata.WriteTo(w); err != nil {
			return
		}

		wrote += n
	}

	return
}
