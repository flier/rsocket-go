package proto

import (
	"context"
	"fmt"
	"io"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// Requester to submit requests on an RSocket connection.
type Requester interface {
	io.Closer

	// Send a single request and get a response stream.
	RequestStream(ctx context.Context, payload *Payload) (<-chan *Result, error)

	// Start a channel (streams in both directions).
	RequestChannel(ctx context.Context, payloads <-chan *Result) (<-chan *Result, error)

	// Send a single request and get a single response.
	RequestResponse(ctx context.Context, payload *Payload) (*Payload, error)

	// Send a single Payload with no response.
	FireAndForget(ctx context.Context, payload *Payload) error

	// Send metadata without response.
	MetadataPush(ctx context.Context, metadata Metadata) error
}

// Requester Side of a RSocket. Sends [Frame]s to a [RSocketResponder]
type rSocketRequester struct {
	*zap.Logger
	frameSender        FrameSender
	streamIDs          StreamIDs
	streamRequestLimit uint
	senders            *sync.Map
	receivers          *sync.Map
}

// NewRequester create a new Requester.
func NewRequester(logger *zap.Logger, frameSender FrameSender, streamIDs StreamIDs, streamRequestLimit uint) Requester {
	return &rSocketRequester{
		Logger:             logger,
		frameSender:        frameSender,
		streamIDs:          streamIDs,
		streamRequestLimit: streamRequestLimit,
		senders:            new(sync.Map),
		receivers:          new(sync.Map),
	}
}

func (requester *rSocketRequester) Close() (err error) {
	return nil
}

type resultReceiver chan *Result

func (requester *rSocketRequester) newReceiver(streamID StreamID) resultReceiver {
	receiver := make(resultReceiver)

	requester.receivers.Store(streamID, receiver)

	return receiver
}

func (requester *rSocketRequester) RequestResponse(ctx context.Context, payload *Payload) (*Payload, error) {
	streamID := requester.streamIDs.Next()
	receiver := requester.newReceiver(streamID)

	requestResponseFrame := payload.buildRequestResponseFrame(streamID)

	if err := requester.sendFrame(ctx, requestResponseFrame); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case result := <-receiver:
		return result.Payload, result.Err
	}
}

func (requester *rSocketRequester) FireAndForget(ctx context.Context, payload *Payload) error {
	streamID := requester.streamIDs.Next()

	return requester.sendFrame(ctx, payload.buildRequestFireAndForgetFrame(streamID))
}

func (requester *rSocketRequester) MetadataPush(ctx context.Context, metadata Metadata) (err error) {
	return requester.sendFrame(ctx, frame.NewMetadataPushFrame(metadata))
}

func (requester *rSocketRequester) RequestStream(ctx context.Context, payload *Payload) (<-chan *Result, error) {
	streamID := requester.streamIDs.Next()
	receiver := requester.newReceiver(streamID)

	requestStreamFrame := frame.NewRequestStreamFrame(streamID, false, uint32(requester.streamRequestLimit),
		payload.HasMetadata, payload.Metadata, payload.Data)

	if err := requester.sendFrame(ctx, requestStreamFrame); err != nil {
		return nil, err
	}

	return requester.receivePayloads(ctx, streamID, func() {}, receiver), nil
}

func (requester *rSocketRequester) RequestChannel(ctx context.Context, payloads <-chan *Result) (<-chan *Result, error) {
	streamID := requester.streamIDs.Next()
	receiver := requester.newReceiver(streamID)

	var payload Payload

	if result, ok := <-payloads; ok {
		if result.Err == nil {
			payload = *result.Payload
		} else {
			defer func() error {
				errorFrame := frame.NewErrorFrame(streamID, frame.ErrApplicationError, result.Err.Error())

				return requester.sendFrame(ctx, errorFrame)
			}()

			payloads = nil
		}
	}

	requestChannelFrame := frame.NewRequestChannelFrame(streamID, false, uint32(requester.streamRequestLimit),
		payload.HasMetadata, payload.Metadata, payload.Data)

	if err := requester.sendFrame(ctx, requestChannelFrame); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	if payloads != nil {
		requests := semaphore.NewWeighted(0)
		requester.senders.Store(streamID, requests)

		sendError := func(err error) error {
			defer cancel()
			defer close(receiver)

			var f frame.Frame

			if err == context.Canceled {
				f = frame.NewCancelFrame(streamID)
			} else {
				f = frame.NewErrorFrame(streamID, frame.ErrApplicationError, err.Error())
			}

			return requester.sendFrame(ctx, f)
		}

		go func() error {
			for {
				select {
				case <-ctx.Done():
					return sendError(ctx.Err())

				case result := <-payloads:
					if err := requests.Acquire(ctx, 1); err != nil {
						return sendError(err)
					}

					if result == nil {
						return requester.sendFrame(ctx, buildCompleteFrame(streamID))
					}

					if result.Err != nil {
						return sendError(result.Err)
					}

					if err := requester.sendFrame(ctx, result.Payload.buildPayloadFrame(streamID, false)); err != nil {
						return sendError(err)
					}
				}
			}
		}()
	}

	return requester.receivePayloads(ctx, streamID, cancel, receiver), nil
}

func (requester *rSocketRequester) sendFrame(ctx context.Context, frame frame.Frame) error {
	requester.Debug("send frame",
		zap.Uint32("stream", uint32(frame.StreamID())),
		zap.Stringer("type", frame.Type()),
		zap.Reflect("frame", frame))

	select {
	case <-ctx.Done():
		return ctx.Err()

	case requester.frameSender <- frame:
		return nil
	}
}

func (requester *rSocketRequester) receivePayloads(ctx context.Context, streamID StreamID, cancel context.CancelFunc, receiver <-chan *Result) <-chan *Result {
	results := make(chan *Result)

	go func() error {
		defer cancel()

		requestN := requester.streamRequestLimit

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case result := <-receiver:
				if result == nil {
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()

				case results <- result:
					requestN--

					if requestN == 0 {
						requestN = requester.streamRequestLimit

						requester.Debug("requestN", zap.Uint32("stream", uint32(streamID)), zap.Uint("n", requestN))

						select {
						case <-ctx.Done():
							return ctx.Err()

						case requester.frameSender <- frame.NewRequestNFrame(streamID, uint32(requestN)):
							break
						}
					}
				}
			}
		}
	}()

	return results
}

func (requester *rSocketRequester) handleFrame(ctx context.Context, f frame.Frame) error {
	streamID := f.StreamID()

	requester.Debug("handle frame",
		zap.Uint32("stream", uint32(streamID)),
		zap.Stringer("type", f.Type()),
		zap.Uint16("flags", uint16(f.Flags())))

	if receiver, ok := requester.receivers.Load(streamID); ok {
		receiver := receiver.(resultReceiver)

		complete := func() {
			requester.Debug("stream complete", zap.Uint32("stream", uint32(streamID)))

			requester.senders.Delete(streamID)
			requester.receivers.Delete(streamID)

			close(receiver)
		}

		receive := func(result *Result) error {
			requester.Debug("forward response", zap.Reflect("payload", result.Payload), zap.Error(result.Err))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case receiver <- result:
				return nil
			}
		}

		switch f := f.(type) {
		case *frame.ErrorFrame:
			defer complete()

			return receive(Err(&Error{f.Code, f.Data}))

		case *frame.CancelFrame:
			defer complete()

			return receive(Err(context.Canceled))

		case *frame.PayloadFrame:
			if f.Complete() {
				defer complete()
			}

			if f.Next() {
				return receive(Ok(&Payload{
					HasMetadata: f.HasMetadata(),
					Metadata:    f.Metadata,
					Data:        f.Data,
				}))
			}

			if !f.Complete() && !f.Next() {
				return frame.ErrInvalid
			}

		case *frame.RequestNFrame:
			if sender, ok := requester.senders.Load(streamID); ok {
				requester.Debug("N requests", zap.Uint32("n", f.Requests))

				sender.(*semaphore.Weighted).Release(int64(f.Requests))

				return nil
			}

		default:
			return fmt.Errorf("Client received unsupported %s frame on %d stream", f.Type(), streamID)
		}
	} else if requester.streamIDs.Current() < streamID {
		if err, ok := f.(*frame.ErrorFrame); ok {
			return fmt.Errorf("Client received error (%s) for non-existent %d stream: %s", err.Code, streamID, err.Data)
		}

		return fmt.Errorf("Client received %s message for non-existent %d stream", f.Type(), streamID)
	}

	return nil
}
