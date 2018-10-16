package frame

import "io"

// CancelFrame cancel outstanding request.
type CancelFrame struct {
	*Header
}

// NewCancelFrame creates a CancelFrame.
func NewCancelFrame(streamID StreamID) *CancelFrame {
	return &CancelFrame{&Header{streamID, TypeCancel, 0}}
}

func readCancelFrame(r io.Reader, header *Header) (*CancelFrame, error) {
	return &CancelFrame{header}, nil
}

// Size returns the encoded size of the frame.
func (frame *CancelFrame) Size() int {
	return frame.Header.Size()
}

// WriteTo writes the encoded frame to w.
func (frame *CancelFrame) WriteTo(w io.Writer) (int64, error) {
	return frame.Header.WriteTo(w)
}
