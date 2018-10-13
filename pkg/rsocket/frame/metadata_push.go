package frame

import (
	"io"
	"io/ioutil"
)

type MetadataPushFrame struct {
	*Header
	Metadata Metadata
}

func ReadMetadataPushFrame(r io.Reader, header *Header) (frame *MetadataPushFrame, err error) {
	var metadata Metadata

	if metadata, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &MetadataPushFrame{
		header,
		metadata,
	}

	return
}

func (frame *MetadataPushFrame) Size() int {
	return frame.Header.Size() + len(frame.Metadata)
}

func (frame *MetadataPushFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = frame.Header.WriteTo(w); err != nil {
		return
	}

	var n int64

	if n, err = frame.Metadata.WriteTo(w); err != nil {
		return
	}

	wrote += n

	return
}
