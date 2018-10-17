package proto

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

const (
	rSocketNamespace   = "rsocket"
	requesterSubsystem = "requester"
	typeLabel          = "type"
)

var (
	frameSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: rSocketNamespace,
		Subsystem: requesterSubsystem,
		Name:      "frame_sent",
		Help:      "Frame sent",
	}, []string{typeLabel})
	frameReceived = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: rSocketNamespace,
		Subsystem: requesterSubsystem,
		Name:      "frame_received",
		Help:      "Frame received",
	}, []string{typeLabel})
	currentStreams = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: rSocketNamespace,
		Subsystem: requesterSubsystem,
		Name:      "current_streams",
		Help:      "Current streams",
	})
	currentChannels = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: rSocketNamespace,
		Subsystem: requesterSubsystem,
		Name:      "current_channels",
		Help:      "Current channels",
	})
)

func init() {
	prometheus.MustRegister(frameSent)
	prometheus.MustRegister(frameReceived)
	prometheus.MustRegister(currentStreams)
	prometheus.MustRegister(currentChannels)
}

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

type resultSender struct {
	c        *sync.Cond
	requests uint32
	ctx      context.Context
	cancel   context.CancelFunc
}

func (requester *rSocketRequester) newResultSender(ctx context.Context, streamID StreamID, initReqs uint) *resultSender {
	ctx, cancel := context.WithCancel(ctx)
	sender := &resultSender{sync.NewCond(new(sync.Mutex)), uint32(initReqs), ctx, cancel}

	requester.senders.Store(streamID, sender)

	return sender
}

func (sender *resultSender) Close() error {
	sender.cancel()

	return nil
}

func (sender *resultSender) Acquire(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	sender.c.L.Lock()
	for sender.requests == 0 {
		sender.c.Wait()
	}
	sender.requests--
	sender.c.L.Unlock()

	return nil
}

func (sender *resultSender) Requests(n uint32) {
	sender.c.L.Lock()
	sender.requests += n
	sender.c.L.Unlock()
	sender.c.Broadcast()
}

type resultReceiver chan *Result

func (requester *rSocketRequester) newResultReceiver(streamID StreamID, capacity uint) resultReceiver {
	receiver := make(resultReceiver, capacity)

	requester.receivers.Store(streamID, receiver)

	return receiver
}

func (receiver resultReceiver) close() {
	close(receiver)
}

func (receiver resultReceiver) receivePayload(ctx context.Context, payload *Payload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case receiver <- &Result{payload, nil}:
		return nil
	}
}
func (receiver resultReceiver) receiveErr(ctx context.Context, err error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case receiver <- &Result{nil, err}:
		return nil
	}
}

func (requester *rSocketRequester) RequestResponse(ctx context.Context, payload *Payload) (*Payload, error) {
	streamID := requester.streamIDs.Next()
	receiver := requester.newResultReceiver(streamID, 1)

	requestResponseFrame := buildRequestResponseFrame(streamID, payload)

	if err := requester.sendFrame(ctx, requestResponseFrame); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case result := <-receiver:
		if result == nil {
			return nil, context.Canceled
		}

		return result.Payload, result.Err
	}
}

func (requester *rSocketRequester) FireAndForget(ctx context.Context, payload *Payload) error {
	streamID := requester.streamIDs.Next()

	return requester.sendFrame(ctx, buildRequestFireAndForgetFrame(streamID, payload))
}

func (requester *rSocketRequester) MetadataPush(ctx context.Context, metadata Metadata) (err error) {
	return requester.sendFrame(ctx, frame.NewMetadataPushFrame(metadata))
}

func (requester *rSocketRequester) RequestStream(ctx context.Context, payload *Payload) (<-chan *Result, error) {
	streamID := requester.streamIDs.Next()
	initReqs := requester.streamRequestLimit
	receiver := requester.newResultReceiver(streamID, initReqs)

	requestStreamFrame := frame.NewRequestStreamFrame(streamID, false, uint32(initReqs),
		payload.HasMetadata, payload.Metadata, payload.Data)

	if err := requester.sendFrame(ctx, requestStreamFrame); err != nil {
		return nil, err
	}

	currentStreams.Inc()

	return requester.receivePayloads(ctx, streamID, receiver, func() {
		currentStreams.Dec()
	}), nil
}

func (requester *rSocketRequester) RequestChannel(ctx context.Context, payloads <-chan *Result) (<-chan *Result, error) {
	ctx, cancel := context.WithCancel(ctx)
	streamID := requester.streamIDs.Next()
	initReqs := requester.streamRequestLimit
	receiver := requester.newResultReceiver(streamID, initReqs)

	sendError := func(ctx context.Context, err error) error {
		defer cancel()

		var f frame.Frame

		if err == context.Canceled {
			f = frame.NewCancelFrame(streamID)
		} else if err, ok := err.(*frame.Error); ok {
			f = frame.NewErrorFrame(streamID, err.Code, err.Data)
		} else {
			f = frame.NewErrorFrame(streamID, frame.ErrApplicationError, err.Error())
		}

		return requester.sendFrame(ctx, f)
	}

	var payload Payload

	select {
	case <-ctx.Done():
		defer sendError(ctx, ctx.Err())

		payloads = nil

	case result, ok := <-payloads:
		if !ok || result == nil {
			defer requester.sendFrame(ctx, buildCompleteFrame(streamID))

			payloads = nil
		} else if result.Err != nil {
			defer sendError(ctx, result.Err)

			payloads = nil
		} else {
			payload = *result.Payload
		}

	default:
		break
	}

	requestChannelFrame := frame.NewRequestChannelFrame(streamID, false, uint32(initReqs),
		payload.HasMetadata, payload.Metadata, payload.Data)

	if err := requester.sendFrame(ctx, requestChannelFrame); err != nil {
		return nil, err
	}

	currentChannels.Inc()

	if payloads != nil {
		sender := requester.newResultSender(ctx, streamID, 0)

		go func() error {
			defer sender.Close()
			defer requester.senders.Delete(streamID)

			for {
				select {
				case <-sender.ctx.Done():
					return sendError(sender.ctx, ctx.Err())

				case result, ok := <-payloads:
					if !ok || result == nil {
						return requester.sendFrame(sender.ctx, buildCompleteFrame(streamID))
					}

					if result.Err != nil {
						return sendError(ctx, result.Err)
					}

					if err := sender.Acquire(sender.ctx); err != nil {
						return sendError(sender.ctx, err)
					}

					payloadFrame := buildPayloadFrame(streamID, false, result.Payload)

					if err := requester.sendFrame(sender.ctx, payloadFrame); err != nil {
						return sendError(sender.ctx, err)
					}
				}
			}
		}()
	}

	return requester.receivePayloads(ctx, streamID, receiver, func() {
		currentChannels.Dec()
	}), nil
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
		frameSent.With(prometheus.Labels{typeLabel: frame.Type().String()}).Inc()

		return nil
	}
}

func (requester *rSocketRequester) receivePayloads(
	ctx context.Context,
	streamID StreamID,
	receiver <-chan *Result,
	destructor func(),
) <-chan *Result {
	results := make(chan *Result)

	go func() error {
		defer destructor()
		defer close(results)
		defer requester.receivers.Delete(streamID)

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

						requester.Debug("request more payloads",
							zap.Uint32("stream", uint32(streamID)),
							zap.Uint("n", requestN))

						requestNFrame := frame.NewRequestNFrame(streamID, uint32(requestN))

						if err := requester.sendFrame(ctx, requestNFrame); err != nil {
							return err
						}
					}
				}
			}
		}
	}()

	return results
}

func (requester *rSocketRequester) findSender(streamID StreamID) (*resultSender, bool) {
	sender, ok := requester.senders.Load(streamID)

	if ok {
		return sender.(*resultSender), true
	}

	return nil, false
}

func (requester *rSocketRequester) findReceiver(streamID StreamID) (resultReceiver, bool) {
	receiver, ok := requester.receivers.Load(streamID)

	if ok {
		return receiver.(resultReceiver), true
	}

	return nil, false
}

func (requester *rSocketRequester) handleFrame(ctx context.Context, f frame.Frame) error {
	frameReceived.With(prometheus.Labels{typeLabel: f.Type().String()}).Inc()

	streamID := f.StreamID()

	requester.Debug("handle frame",
		zap.Uint32("stream", uint32(streamID)),
		zap.Stringer("type", f.Type()),
		zap.Uint16("flags", uint16(f.Flags())))

	if receiver, ok := requester.findReceiver(streamID); ok {
		complete := func(reason error) {
			requester.Debug("stream complete",
				zap.Uint32("stream", uint32(streamID)),
				zap.Error(reason))

			requester.receivers.Delete(streamID)

			close(receiver)
		}

		switch f := f.(type) {
		case *frame.ErrorFrame:
			defer complete(f.Err())

			return receiver.receiveErr(ctx, f.Err())

		case *frame.CancelFrame:
			defer complete(context.Canceled)

			if sender, ok := requester.findSender(streamID); ok {
				requester.senders.Delete(streamID)
				sender.cancel()
			}

			return receiver.receiveErr(ctx, context.Canceled)

		case *frame.PayloadFrame:
			if f.Complete() {
				defer complete(nil)
			}

			if f.Next() {
				return receiver.receivePayload(ctx, &Payload{
					HasMetadata: f.HasMetadata(),
					Metadata:    f.Metadata,
					Data:        f.Data,
				})
			}

			if !f.Complete() && !f.Next() {
				return frame.ErrInvalid
			}

		case *frame.RequestNFrame:
			if sender, ok := requester.findSender(streamID); ok {
				sender.Requests(f.Requests)

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
