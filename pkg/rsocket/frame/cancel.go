package frame

import "io"

type CancelFrame struct {
	*Header
}

func readCancelFrame(r io.Reader, header *Header) (*CancelFrame, error) {
	return &CancelFrame{header}, nil
}

func (frame *CancelFrame) Size() int {
	return frame.Header.Size()
}

func (frame *CancelFrame) WriteTo(w io.Writer) (int64, error) {
	return frame.Header.WriteTo(w)
}
