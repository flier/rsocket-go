package proto

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// Request APIs to submit requests on an RSocket connection.
type Requester interface {
	io.Closer

	// Send a single request and get a response stream.
	RequestStream(ctx context.Context, request Payload) (PayloadStream, error)

	// Start a channel (streams in both directions).
	RequestChannel(ctx context.Context, requests PayloadStream) (PayloadStream, error)

	// Send a single request and get a single response.
	RequestResponse(ctx context.Context, request Payload) (Payload, error)

	// Send a single Payload with no response.
	FireAndForget(ctx context.Context, request Payload) error

	// Send metadata without response.
	MetadataPush(ctx context.Context, request Metadata) error
}

type OnCancel func()
type OnFailure func(error)
type OnComplete func()
type OnFinally func()

type Sender interface {
	Cancel() error

	Request(n uint) error

	OnFailure(callback OnFailure) Sender

	OnCancel(callback OnCancel) Sender

	OnFinally(callback OnFinally) Sender
}

type Receiver interface {
	Reject(err error) error

	Next(payload Payload) error

	Complete() error

	OnFailure(callback OnFailure) Receiver

	OnComplete(callback OnComplete) Receiver

	OnFinally(callback OnFinally) Receiver
}

type streamSender struct {
	*payloadStream
	onFailure OnFailure
	onCancel  OnCancel
	onFinally OnFinally
}

var _ Sender = (*streamSender)(nil)

func newStreamSender() *streamSender {
	return &streamSender{newPayloadStream(), nil, nil, nil}
}

func (sender *streamSender) Cancel() error {
	defer sender.onCancel()
	defer sender.onFinally()

	return sender.payloadStream.Reject(context.Canceled)
}

func (sender *streamSender) Request(n uint) error {
	return nil
}

func (sender *streamSender) OnFailure(callback OnFailure) Sender {
	sender.onFailure = callback

	return sender
}

func (sender *streamSender) OnCancel(callback OnCancel) Sender {
	sender.onCancel = callback

	return sender
}

func (sender *streamSender) OnFinally(callback OnFinally) Sender {
	sender.onFinally = callback

	return sender
}

type streamReceiver struct {
	*payloadStream
	onFailure  OnFailure
	onComplete OnComplete
	onFinally  OnFinally
}

var _ Receiver = (*streamReceiver)(nil)

func newStreamReceiver() *streamReceiver {
	return &streamReceiver{newPayloadStream(), nil, nil, nil}
}

func (receiver *streamReceiver) Reject(err error) error {
	defer receiver.onFailure(err)
	defer receiver.onFinally()

	return receiver.payloadStream.Reject(err)
}

func (receiver *streamReceiver) Complete() error {
	defer receiver.onComplete()
	defer receiver.onFinally()

	return receiver.payloadStream.Complete()
}

func (receiver *streamReceiver) OnFailure(callback OnFailure) Receiver {
	receiver.onFailure = callback

	return receiver
}

func (receiver *streamReceiver) OnComplete(callback OnComplete) Receiver {
	receiver.onComplete = callback

	return receiver
}

func (receiver *streamReceiver) OnFinally(callback OnFinally) Receiver {
	receiver.onFinally = callback

	return receiver
}

// Requester Side of a RSocket. Sends [Frame]s to a [RSocketResponder]
type rSocketRequester struct {
	conn               Connection
	streamIds          StreamIds
	streamRequestLimit uint
	frameSender        FrameSender
	senders            *sync.Map
	receivers          *sync.Map
}

func NewRequester(conn Connection, streamIds StreamIds, streamRequestLimit uint) Requester {
	return &rSocketRequester{
		conn:               conn,
		streamIds:          streamIds,
		streamRequestLimit: streamRequestLimit,
		frameSender:        make(FrameSender),
		senders:            new(sync.Map),
		receivers:          new(sync.Map),
	}
}

func (requester *rSocketRequester) Close() (err error) {
	return
}

func (requester *rSocketRequester) RequestStream(ctx context.Context, request Payload) (PayloadStream, error) {
	streamId := requester.streamIds.Next()
	receiver := newStreamReceiver()

	requester.receivers.Store(streamId, receiver)

	receiver.OnFinally(func() {
		requester.receivers.Delete(streamId)
	})

	requester.frameSender <- request.buildRequestResponseFrame(streamId, false)

	return receiver, nil
}

func (requester *rSocketRequester) RequestChannel(ctx context.Context, requests PayloadStream) (PayloadStream, error) {
	streamId := requester.streamIds.Next()
	receiver := newStreamReceiver()

	requester.receivers.Store(streamId, receiver)

	receiver.OnFinally(func() {
		requester.receivers.Delete(streamId)
	})

	go func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case payload := <-requests.Payloads():
				if payload == nil {
					return receiver.Complete()
				}

				receiver.payloads <- payload

				if requests.Err() != nil {
					return receiver.Reject(requests.Err())
				}
			}
		}
	}()

	return receiver, nil
}

func (requester *rSocketRequester) RequestResponse(ctx context.Context, request Payload) (response Payload, err error) {
	return
}

func (requester *rSocketRequester) FireAndForget(ctx context.Context, request Payload) (err error) {
	streamId := requester.streamIds.Next()

	requester.frameSender <- request.buildRequestFireAndForgetFrame(streamId, false)

	return
}

func (requester *rSocketRequester) MetadataPush(ctx context.Context, request Metadata) (err error) {
	return
}

func (requester *rSocketRequester) handleFrame(frm frame.Frame) error {
	streamId := frm.StreamId()

	if receiver, ok := requester.receivers.Load(streamId); ok {
		receiver := receiver.(Receiver)

		switch frm := frm.(type) {
		case *frame.ErrorFrame:
			return receiver.Reject(frm.Err())

		case *frame.CancelFrame:
			sender, ok := requester.senders.Load(streamId)

			requester.senders.Delete(streamId)
			requester.receivers.Delete(streamId)

			if ok {
				return sender.(Sender).Cancel()
			}

		case *frame.PayloadFrame:
			if frm.Next() {
				receiver.Next(Payload{
					HasMetadata: frm.HasMetadata(),
					Metadata:    frm.Metadata,
					Data:        frm.Data,
				})
			}

			if frm.Complete() {
				return receiver.Complete()
			}

		case *frame.RequestNFrame:
			if sender, ok := requester.senders.Load(streamId); ok {
				return sender.(Sender).Request(uint(frm.Requests))
			}

		default:
			return fmt.Errorf("Client received unsupported %s frame on %d stream", frm.Type(), streamId)
		}
	} else if requester.streamIds.Current() < streamId {
		if err, ok := frm.(*frame.ErrorFrame); ok {
			return fmt.Errorf("Client received error (%s) for non-existent %d stream: %s", err.Code, streamId, err.Data)
		} else {
			return fmt.Errorf("Client received %s message for non-existent %d stream", frm.Type(), streamId)
		}
	}

	return nil
}
