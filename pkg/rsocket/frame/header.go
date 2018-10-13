package frame

import (
	"encoding/binary"
	"io"
)

type StreamId uint32

type Header struct {
	streamId  StreamId
	frameType Type
	flags     Flags
}

const headerSize = streamIdSize + frameFlagSize
const streamIdSize = uint32Size
const frameFlagSize = uint16Size
const frameTypeShift = 10

func readHeader(r io.Reader) (header *Header, err error) {
	var streamId uint32
	var v uint16

	if err = binary.Read(r, binary.BigEndian, &streamId); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &v); err != nil {
		return
	}

	header = &Header{
		StreamId(streamId),
		Type(v >> frameTypeShift),
		Flags(v & ((1 << frameTypeShift) - 1)),
	}

	return
}

func (header *Header) StreamId() StreamId {
	return header.streamId
}

func (header *Header) Type() Type {
	return header.frameType
}

func (header *Header) Flags() Flags {
	return header.flags
}

func (header *Header) Size() int {
	return headerSize
}

func (header *Header) WriteTo(w io.Writer) (int64, error) {
	if err := binary.Write(w, binary.BigEndian, uint32(header.streamId)); err != nil {
		return 0, err
	}

	v := uint16(header.frameType)<<frameTypeShift | uint16(header.flags)

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return 0, err
	}

	return headerSize, nil
}

func (header *Header) CanIgnore() bool {
	return header.flags.IsSet(FlagIgnore)
}

func (header *Header) HasMetadata() bool {
	return header.flags.IsSet(FlagMetadata)
}

func (header *Header) HasResumeToken() bool {
	return header.flags.IsSet(FlagResumeEnable)
}
