package proto

import (
	"sync/atomic"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// StreamID representing the stream identifier.
type StreamID = frame.StreamID

// StreamIDs generates StreamID.
type StreamIDs struct {
	streamID int32
}

// ClientStreamIDs generate StreamID for the RSocket client.
func ClientStreamIDs() StreamIDs { return StreamIDs{-1} }

// ServerStreamIDs generate StreamID for the RSocket server.
func ServerStreamIDs() StreamIDs { return StreamIDs{0} }

// Current returns the current StreamID
func (ids *StreamIDs) Current() StreamID {
	return StreamID(atomic.LoadInt32(&ids.streamID))
}

// Next returns the next StreamID
func (ids *StreamIDs) Next() StreamID {
	return StreamID(atomic.AddInt32(&ids.streamID, 2))
}
