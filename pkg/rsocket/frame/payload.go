package frame

import (
	"io"
	"io/ioutil"
)

// PayloadFrame send payload on a request or stream.
type PayloadFrame struct {
	*Header
	Metadata Metadata
	Data     []byte
}

// NewPayloadFrame creates a PayloadFrame.
func NewPayloadFrame(streamID StreamID, follows bool, complete bool, next bool, hasMetadata bool, metadata Metadata, data []byte) *PayloadFrame {
	var flags Flags

	if hasMetadata {
		flags.Set(FlagMetadata)
	}
	if follows {
		flags.Set(FlagFollows)
	}
	if complete {
		flags.Set(FlagComplete)
	}
	if next {
		flags.Set(FlagNext)
	}

	return &PayloadFrame{
		&Header{streamID, TypePayload, flags},
		metadata,
		data,
	}
}

func readPayloadFrame(r io.Reader, header *Header) (frame *PayloadFrame, err error) {
	var metadata, data []byte

	if header.Next() {
		if header.HasMetadata() {
			if metadata, err = readMetadata(r); err != nil {
				return
			}
		}

		if data, err = ioutil.ReadAll(r); err != nil {
			return
		}
	}

	frame = &PayloadFrame{
		header,
		metadata,
		data,
	}

	return
}

// Size returns the encoded size of the frame.
func (payload *PayloadFrame) Size() int {
	return payload.Header.Size() + payload.Metadata.Size() + len(payload.Data)
}

// WriteTo writes the encoded frame to w.
func (payload *PayloadFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = payload.Header.WriteTo(w); err != nil {
		return
	}

	if payload.Next() {
		var n int64

		if payload.HasMetadata() {
			if n, err = payload.Metadata.WriteTo(w); err != nil {
				return
			}

			wrote += n
		}

		if n, err = writeExact(w, []byte(payload.Data)); err != nil {
			return
		}

		wrote += n
	}

	return
}
