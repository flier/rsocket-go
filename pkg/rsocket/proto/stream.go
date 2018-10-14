package proto

import (
	"sync/atomic"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

type StreamId = frame.StreamId

func ClientStreamIds() StreamIds { return StreamIds{-1} }
func ServerStreamIds() StreamIds { return StreamIds{0} }

type StreamIds struct {
	streamId int32
}

func (ids *StreamIds) Current() StreamId {
	return StreamId(atomic.LoadInt32(&ids.streamId))
}

func (ids *StreamIds) Next() StreamId {
	return StreamId(atomic.AddInt32(&ids.streamId, 2))
}
