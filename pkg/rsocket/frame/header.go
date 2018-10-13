package frame

import (
	"encoding/binary"
	"io"
)

type StreamId uint32

type Header struct {
	StreamId  StreamId
	FrameType Type
	Flags     Flags
}

const headerSize = streamIdSize + frameFlagSize
const streamIdSize = uint32Size
const frameFlagSize = uint16Size
const frameTypeShift = 10

func ReadHeader(r io.Reader) (header *Header, err error) {
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

func (header *Header) Size() int {
	return headerSize
}

func (header *Header) WriteTo(w io.Writer) (int64, error) {
	if err := binary.Write(w, binary.BigEndian, uint32(header.StreamId)); err != nil {
		return 0, err
	}

	v := uint16(header.FrameType)<<frameTypeShift | uint16(header.Flags)

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return 0, err
	}

	return headerSize, nil
}

func (header *Header) CanIgnore() bool {
	return header.Flags.IsSet(FlagIgnore)
}

func (header *Header) HasMetadata() bool {
	return header.Flags.IsSet(FlagMetadata)
}

func (header *Header) HasResumeToken() bool {
	return header.Flags.IsSet(FlagResumeEnable)
}
