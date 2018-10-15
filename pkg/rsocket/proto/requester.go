package proto

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"golang.org/x/sync/semaphore"
)

// Request APIs to submit requests on an RSocket connection.
type Requester interface {
	io.Closer

	// Send a single request and get a response stream.
	RequestStream(ctx context.Context, request Payload) (PayloadStream, error)

	// Start a channel (streams in both directions).
	RequestChannel(ctx context.Context, requests PayloadStream) (PayloadStream, error)

	// Send a single request and get a single response.
	RequestResponse(ctx context.Context, request Payload) (PayloadFuture, error)

	// Send a single Payload with no response.
	FireAndForget(ctx context.Context, request Payload) error

	// Send metadata without response.
	MetadataPush(ctx context.Context, request Metadata) error
}

type Sender interface {
	Cancel() error

	Request(n uint) error
}

type Mapper func(*Payload) frame.Frame

type frameSender struct {
	cancel context.CancelFunc
}

var _ Sender = (*frameSender)(nil)

func newFrameSender(ctx context.Context, frame frame.Frame, frames chan<- frame.Frame, destructor func()) *frameSender {
	ctx, cancel := context.WithCancel(ctx)

	go func() error {
		defer destructor()

		select {
		case <-ctx.Done():
			return ctx.Err()

		case frames <- frame:
			return nil
		}
	}()

	return &frameSender{cancel}
}

func (sender *frameSender) Cancel() error {
	sender.cancel()
	return nil
}

func (sender *frameSender) Request(n uint) error {
	return nil
}

type streamSender struct {
	cancel   context.CancelFunc
	requests *semaphore.Weighted
}

var _ Sender = (*streamSender)(nil)

func newStreamSender(
	ctx context.Context,
	payloads PayloadStream,
	mapper Mapper,
	n uint,
	frames chan<- frame.Frame,
	destructor func(),
) *streamSender {
	ctx, cancel := context.WithCancel(ctx)
	requests := semaphore.NewWeighted(int64(n))

	go func() error {
		defer destructor()

		for {
			err := requests.Acquire(ctx, 1)

			if err != nil {
				return err
			}

			var frame frame.Frame

			select {
			case <-ctx.Done():
				return ctx.Err()

			case payload := <-payloads.Stream():
				if payload == nil {
					return nil
				}

				frame = mapper(payload)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()

			case frames <- frame:
				continue
			}
		}
	}()

	return &streamSender{cancel, requests}
}

func (sender *streamSender) Cancel() error {
	sender.cancel()
	return nil
}

func (sender *streamSender) Request(n uint) error {
	sender.requests.Release(int64(n))
	return nil
}

type Receiver interface {
	Next(payload Payload) error

	Reject(err error) error

	Complete() error
}

type payloadReceiver struct {
	destructor func()
	c          *sync.Cond
	payload    *Payload
	err        error
}

var _ Receiver = (*payloadReceiver)(nil)
var _ PayloadFuture = (*payloadReceiver)(nil)

func newPayloadReceiver(destructor func()) *payloadReceiver {
	return &payloadReceiver{destructor, sync.NewCond(new(sync.Mutex)), nil, nil}
}

func (receiver *payloadReceiver) Next(payload Payload) error {
	receiver.payload = &payload
	receiver.c.Broadcast()

	return nil
}

func (receiver *payloadReceiver) Reject(err error) error {
	receiver.err = err
	receiver.c.Broadcast()
	receiver.destructor()
	return nil
}

func (receiver *payloadReceiver) Complete() error {
	receiver.c.Broadcast()
	receiver.destructor()

	return nil
}

func (receiver *payloadReceiver) Get() (*Payload, error) {
	receiver.c.L.Lock()

	for receiver.payload == nil && receiver.err == nil {
		receiver.c.Wait()
	}

	receiver.c.L.Unlock()

	return receiver.payload, receiver.err
}

func (receiver *payloadReceiver) Err() error {
	return receiver.err
}

type streamReceiver struct {
	destructor func()
	payloads   chan *Payload
	err        error
}

var _ Receiver = (*streamReceiver)(nil)
var _ PayloadStream = (*streamReceiver)(nil)

func newStreamReceiver(destructor func()) *streamReceiver {
	return &streamReceiver{destructor, make(chan *Payload), nil}
}

func (receiver *streamReceiver) Next(payload Payload) error {
	receiver.payloads <- &payload

	return nil
}

func (receiver *streamReceiver) Reject(err error) error {
	receiver.err = err

	return receiver.Complete()
}

func (receiver *streamReceiver) Complete() error {
	defer receiver.destructor()

	close(receiver.payloads)

	return nil
}

func (receiver *streamReceiver) Stream() <-chan *Payload {
	return receiver.payloads
}

func (receiver *streamReceiver) Err() error {
	return receiver.err
}

// Requester Side of a RSocket. Sends [Frame]s to a [RSocketResponder]
type rSocketRequester struct {
	conn               Connection
	streamIDs          StreamIDs
	streamRequestLimit uint
	frameSender        FrameSender
	senders            *sync.Map
	receivers          *sync.Map
}

func NewRequester(conn Connection, streamIDs StreamIDs, streamRequestLimit uint) Requester {
	return &rSocketRequester{
		conn:               conn,
		streamIDs:          streamIDs,
		streamRequestLimit: streamRequestLimit,
		frameSender:        make(FrameSender),
		senders:            new(sync.Map),
		receivers:          new(sync.Map),
	}
}

func (requester *rSocketRequester) Close() (err error) {
	return
}

func (requester *rSocketRequester) RequestStream(ctx context.Context, payload Payload) (PayloadStream, error) {
	streamID := requester.streamIDs.Next()

	requester.senders.Store(streamID,
		newFrameSender(ctx, payload.buildRequestStreamFrame(streamID, 128), requester.frameSender, func() {
			requester.senders.Delete(streamID)
		}))

	receiver := newStreamReceiver(func() {
		requester.receivers.Delete(streamID)
	})

	requester.receivers.Store(streamID, receiver)

	return receiver, nil
}

func (requester *rSocketRequester) RequestChannel(ctx context.Context, requests PayloadStream) (PayloadStream, error) {
	streamID := requester.streamIDs.Next()
	requester.senders.Store(streamID,
		newStreamSender(
			ctx,
			requests,
			func(payload *Payload) frame.Frame {
				return payload.buildRequestChannelFrame(streamID, 128)
			},
			128,
			requester.frameSender,
			func() {
				requester.senders.Delete(streamID)
			},
		),
	)
	receiver := newStreamReceiver(func() {
		requester.receivers.Delete(streamID)
	})

	requester.receivers.Store(streamID, receiver)

	return receiver, nil
}

func (requester *rSocketRequester) RequestResponse(ctx context.Context, payload Payload) (PayloadFuture, error) {
	streamID := requester.streamIDs.Next()

	requester.senders.Store(streamID, newFrameSender(ctx,
		payload.buildRequestResponseFrame(streamID), requester.frameSender, func() {
			requester.senders.Delete(streamID)
		}),
	)

	receiver := newPayloadReceiver(func() {
		requester.receivers.Delete(streamID)
	})

	requester.receivers.Store(streamID, receiver)

	return receiver, nil
}

func (requester *rSocketRequester) FireAndForget(ctx context.Context, payload Payload) (err error) {
	streamID := requester.streamIDs.Next()

	requester.senders.Store(streamID, newFrameSender(ctx,
		payload.buildRequestFireAndForgetFrame(streamID), requester.frameSender, func() {
			requester.senders.Delete(streamID)
		}),
	)

	return
}

func (requester *rSocketRequester) MetadataPush(ctx context.Context, request Metadata) (err error) {
	return
}

func (requester *rSocketRequester) handleFrame(frm frame.Frame) error {
	streamID := frm.StreamID()

	if receiver, ok := requester.receivers.Load(streamID); ok {
		receiver := receiver.(Receiver)

		switch frm := frm.(type) {
		case *frame.ErrorFrame:
			return receiver.Reject(frm.Err())

		case *frame.CancelFrame:
			sender, ok := requester.senders.Load(streamID)

			requester.senders.Delete(streamID)
			requester.receivers.Delete(streamID)

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
			if sender, ok := requester.senders.Load(streamID); ok {
				return sender.(Sender).Request(uint(frm.Requests))
			}

		default:
			return fmt.Errorf("Client received unsupported %s frame on %d stream", frm.Type(), streamID)
		}
	} else if requester.streamIDs.Current() < streamID {
		if err, ok := frm.(*frame.ErrorFrame); ok {
			return fmt.Errorf("Client received error (%s) for non-existent %d stream: %s", err.Code, streamID, err.Data)
		} else {
			return fmt.Errorf("Client received %s message for non-existent %d stream", frm.Type(), streamID)
		}
	}

	return nil
}
