package proto

import (
	"encoding/json"

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

// JSON creates a application/json Payload without metadata.
func JSON(v interface{}) (*Payload, error) {
	data, err := json.Marshal(v)

	if err != nil {
		return nil, err
	}

	return &Payload{false, nil, []byte(data)}, nil
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

func buildRequestResponseFrame(streamID StreamID, payload *Payload) *frame.RequestResponseFrame {
	return frame.NewRequestResponseFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildRequestFireAndForgetFrame(streamID StreamID, payload *Payload) *frame.RequestFireAndForgetFrame {
	return frame.NewRequestFireAndForgetFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildRequestStreamFrame(streamID StreamID, initReqs uint32, payload *Payload) *frame.RequestStreamFrame {
	return frame.NewRequestStreamFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildRequestChannelFrame(streamID StreamID, initReqs uint32, payload *Payload) *frame.RequestChannelFrame {
	return frame.NewRequestChannelFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildPayloadFrame(streamID StreamID, complete bool, payload *Payload) *frame.PayloadFrame {
	return frame.NewPayloadFrame(streamID, false, complete, true, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildCompleteFrame(streamID StreamID) *frame.PayloadFrame {
	return frame.NewPayloadFrame(streamID, false, true, false, false, nil, nil)
}
