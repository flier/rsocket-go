package frame

import (
	"encoding/binary"
	"fmt"
	"io"
)

// StreamID is the stream identifiers
type StreamID uint32

func (id StreamID) String() string {
	return fmt.Sprintf("#%d", id)
}

// Header of frame
type Header struct {
	streamID  StreamID
	frameType Type
	flags     Flags
}

const headerSize = streamIDSize + frameFlagSize
const streamIDSize = uint32Size
const frameFlagSize = uint16Size
const frameTypeShift = 10

func readHeader(r io.Reader) (header *Header, err error) {
	var streamID uint32
	var v uint16

	if err = binary.Read(r, binary.BigEndian, &streamID); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &v); err != nil {
		return
	}

	header = &Header{
		StreamID(streamID),
		Type(v >> frameTypeShift),
		Flags(v & ((1 << frameTypeShift) - 1)),
	}

	return
}

// StreamID returnes the stream identifiers.
func (header *Header) StreamID() StreamID {
	return header.streamID
}

// Type of frame.
func (header *Header) Type() Type {
	return header.frameType
}

// Flags of frame.
func (header *Header) Flags() Flags {
	return header.flags
}

// CanIgnore indicates the protocol can ignore frame if not understood
func (header *Header) CanIgnore() bool {
	return header.flags.IsSet(FlagIgnore)
}

// HasMetadata indicates the metadata present
func (header *Header) HasMetadata() bool {
	return header.flags.IsSet(FlagMetadata)
}

// HasResumeToken indicates the resume identification token present.
func (header *Header) HasResumeToken() bool {
	return header.flags.IsSet(FlagResumeEnable)
}

// Lease indicates the requester will honor LEASE (or not).
func (header *Header) Lease() bool {
	return header.flags.IsSet(FlagLease)
}

// NeedRespond indicates respond with KEEPALIVE or not.
func (header *Header) NeedRespond() bool {
	return header.flags.IsSet(FlagRespond)
}

// Follows indicates more fragments follow this fragment.
func (header *Header) Follows() bool {
	return header.flags.IsSet(FlagFollows)
}

// Complete indicates stream completion.
func (header *Header) Complete() bool {
	return header.flags.IsSet(FlagComplete)
}

// Next indicates the payload data and/or metadata present.
func (header *Header) Next() bool {
	return header.flags.IsSet(FlagNext)
}

func (header *Header) String() string {
	return header.frameType.String()
}

// Size returns the encoded size of the header.
func (header *Header) Size() int {
	return headerSize
}

// WriteTo writes the header to w.
func (header *Header) WriteTo(w io.Writer) (int64, error) {
	if err := binary.Write(w, binary.BigEndian, uint32(header.streamID)); err != nil {
		return 0, err
	}

	v := uint16(header.frameType)<<frameTypeShift | uint16(header.flags)

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return 0, err
	}

	return headerSize, nil
}
