package frame

import (
	"io"
	"io/ioutil"
)

type PayloadFrame struct {
	*Header
	Metadata Metadata
	Data     []byte
}

func readPayloadFrame(r io.Reader, header *Header) (frame *PayloadFrame, err error) {
	var metadata, data []byte

	if header.HasMetadata() {
		if metadata, err = readMetadata(r); err != nil {
			return
		}
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &PayloadFrame{
		header,
		metadata,
		data,
	}

	return
}

func (payload *PayloadFrame) Size() int {
	return payload.Header.Size() + payload.Metadata.Size() + len(payload.Data)
}

func (payload *PayloadFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = payload.Header.WriteTo(w); err != nil {
		return
	}

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

	return
}
