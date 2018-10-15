package proto

import (
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

type Payload struct {
	HasMetadata bool
	Metadata    Metadata
	Data        []byte
}

func (payload Payload) buildRequestResponseFrame(streamID StreamID) *frame.RequestResponseFrame {
	return frame.NewRequestResponseFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload Payload) buildRequestFireAndForgetFrame(streamID StreamID) *frame.RequestFireAndForgetFrame {
	return frame.NewRequestFireAndForgetFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload Payload) buildRequestStreamFrame(streamID StreamID, initReqs uint32) *frame.RequestStreamFrame {
	return frame.NewRequestStreamFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload Payload) buildRequestChannelFrame(streamID StreamID, initReqs uint32) *frame.RequestChannelFrame {
	return frame.NewRequestChannelFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

type PayloadStream interface {
	Stream() <-chan *Payload

	Err() error
}

type PayloadFuture interface {
	Get() (*Payload, error)
}
