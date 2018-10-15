package proto

import (
	"sync/atomic"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

type StreamID = frame.StreamID

func ClientStreamIDs() StreamIDs { return StreamIDs{-1} }
func ServerStreamIDs() StreamIDs { return StreamIDs{0} }

type StreamIDs struct {
	streamID int32
}

func (ids *StreamIDs) Current() StreamID {
	return StreamID(atomic.LoadInt32(&ids.streamID))
}

func (ids *StreamIDs) Next() StreamID {
	return StreamID(atomic.AddInt32(&ids.streamID, 2))
}
