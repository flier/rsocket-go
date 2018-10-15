package frame

import (
	"encoding/binary"
	"io"
)

type StreamID uint32

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

func (header *Header) StreamID() StreamID {
	return header.streamID
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
	if err := binary.Write(w, binary.BigEndian, uint32(header.streamID)); err != nil {
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

func (header *Header) Next() bool {
	return header.flags.IsSet(FlagNext)
}

func (header *Header) Complete() bool {
	return header.flags.IsSet(FlagComplete)
}
