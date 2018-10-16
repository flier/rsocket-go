package proto

import (
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// Metadata holds metadata for the request.
type Metadata = frame.Metadata

// Payload is the request payload with metadata and data.
type Payload struct {
	HasMetadata bool
	Metadata    Metadata
	Data        []byte
}

// Text creates a plain/text Payload without metadata.
func Text(s string) *Payload {
	return &Payload{false, nil, []byte(s)}
}

// Text returnes the data as plain/text.
func (payload *Payload) Text() string {
	return string(payload.Data)
}

// WithMetadata returns a Payload with metadata.
func (payload *Payload) WithMetadata(metadata Metadata) *Payload {
	payload.HasMetadata = true
	payload.Metadata = metadata

	return payload
}

func (payload *Payload) buildRequestResponseFrame(streamID StreamID) *frame.RequestResponseFrame {
	return frame.NewRequestResponseFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildRequestFireAndForgetFrame(streamID StreamID) *frame.RequestFireAndForgetFrame {
	return frame.NewRequestFireAndForgetFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildRequestStreamFrame(streamID StreamID, initReqs uint32) *frame.RequestStreamFrame {
	return frame.NewRequestStreamFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildRequestChannelFrame(streamID StreamID, initReqs uint32) *frame.RequestChannelFrame {
	return frame.NewRequestChannelFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildPayloadFrame(streamID StreamID, complete bool) *frame.PayloadFrame {
	return frame.NewPayloadFrame(streamID, false, complete, true, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildCompleteFrame(streamID StreamID) *frame.PayloadFrame {
	return frame.NewPayloadFrame(streamID, false, true, false, false, nil, nil)
}
