package proto

import (
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

type Payload struct {
	HasMetadata bool
	Metadata    Metadata
	Data        []byte
}

func (payload Payload) buildRequestResponseFrame(streamId StreamId, follows bool) *frame.RequestResponseFrame {
	return frame.NewRequestResponseFrame(streamId, follows, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload Payload) buildRequestFireAndForgetFrame(streamId StreamId, follows bool) *frame.RequestFireAndForgetFrame {
	return frame.NewRequestFireAndForgetFrame(streamId, follows, payload.HasMetadata, payload.Metadata, payload.Data)
}

type PayloadStream interface {
	Payloads() <-chan *Payload

	Err() error
}

type PayloadChan <-chan *Payload

func (c PayloadChan) Payloads() <-chan *Payload { return c }

func (c PayloadChan) Err() error { return nil }

type payloadStream struct {
	payloads chan *Payload
	err      error
}

func newPayloadStream() *payloadStream {
	return &payloadStream{
		payloads: make(chan *Payload),
		err:      nil,
	}
}

func (stream *payloadStream) Payloads() <-chan *Payload {
	return stream.payloads
}

func (stream *payloadStream) Err() error {
	return stream.err
}

func (stream *payloadStream) Reject(err error) error {
	stream.err = err

	close(stream.payloads)

	return nil
}

func (stream *payloadStream) Next(payload Payload) error {
	stream.payloads <- &payload

	return nil
}

func (stream *payloadStream) Complete() error {
	close(stream.payloads)

	return nil
}
