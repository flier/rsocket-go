package frame

import (
	"io"
	"io/ioutil"
)

// MetadataPushFrame is an asynchronous Metadata frame.
type MetadataPushFrame struct {
	*Header
	Metadata Metadata
}

// NewMetadataPushFrame creates a MetadataPushFrame
func NewMetadataPushFrame(metadata Metadata) *MetadataPushFrame {
	return &MetadataPushFrame{
		&Header{0, TypeMetadataPush, FlagMetadata},
		metadata,
	}
}

func readMetadataPushFrame(r io.Reader, header *Header) (frame *MetadataPushFrame, err error) {
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

// Size returns the encoded size of the frame.
func (frame *MetadataPushFrame) Size() int {
	return frame.Header.Size() + len(frame.Metadata)
}

// WriteTo writes the encoded frame to w.
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
