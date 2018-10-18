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
	RequestStream(ctx context.Context, payload *Payload) (*PayloadStream, error)

	// Start a channel (streams in both directions).
	RequestChannel(ctx context.Context, payloads *PayloadStream) (*PayloadStream, error)

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

type resultReceiver struct {
	C chan *Result
}

func (requester *rSocketRequester) newResultReceiver(streamID StreamID, capacity uint) *resultReceiver {
	receiver := &resultReceiver{make(chan *Result, capacity)}

	requester.receivers.Store(streamID, receiver)

	return receiver
}

func (receiver *resultReceiver) Recv(ctx context.Context) (*Payload, error) {
	stream := &PayloadStream{receiver.C}

	return stream.Recv(ctx)
}

func (receiver *resultReceiver) Close() {
	close(receiver.C)
}

func (receiver *resultReceiver) Send(ctx context.Context, result *Result) error {
	sink := &PayloadSink{receiver.C}

	return sink.Send(ctx, result)
}

func (requester *rSocketRequester) RequestResponse(ctx context.Context, payload *Payload) (*Payload, error) {
	streamID := requester.streamIDs.Next()
	receiver := requester.newResultReceiver(streamID, 1)

	requestResponseFrame := buildRequestResponseFrame(streamID, payload)

	if err := requester.sendFrame(ctx, requestResponseFrame); err != nil {
		return nil, err
	}

	payload, err := receiver.Recv(ctx)

	if err == context.Canceled {
		return nil, requester.sendFrame(ctx, frame.NewCancelFrame(streamID))
	}

	return payload, err
}

func (requester *rSocketRequester) FireAndForget(ctx context.Context, payload *Payload) error {
	streamID := requester.streamIDs.Next()

	return requester.sendFrame(ctx, buildRequestFireAndForgetFrame(streamID, payload))
}

func (requester *rSocketRequester) MetadataPush(ctx context.Context, metadata Metadata) (err error) {
	return requester.sendFrame(ctx, frame.NewMetadataPushFrame(metadata))
}

func (requester *rSocketRequester) RequestStream(ctx context.Context, payload *Payload) (*PayloadStream, error) {
	streamID := requester.streamIDs.Next()
	initReqs := requester.streamRequestLimit
	receiver := requester.newResultReceiver(streamID, initReqs)

	requestStreamFrame := frame.NewRequestStreamFrame(streamID, false, uint32(initReqs),
		payload.HasMetadata, payload.Metadata, payload.Data)

	if err := requester.sendFrame(ctx, requestStreamFrame); err != nil {
		return nil, err
	}

	currentStreams.Inc()

	return requester.receivePayloads(ctx, streamID, receiver.C, func() {
		currentStreams.Dec()
	}), nil
}

func (requester *rSocketRequester) RequestChannel(ctx context.Context, payloads *PayloadStream) (*PayloadStream, error) {
	ctx, cancel := context.WithCancel(ctx)
	streamID := requester.streamIDs.Next()
	initReqs := requester.streamRequestLimit
	receiver := requester.newResultReceiver(streamID, initReqs)

	sendError := func(ctx context.Context, err error) error {
		defer cancel()

		var f frame.Frame

		if err == context.Canceled {
			f = frame.NewCancelFrame(streamID)
		} else if errorFrame, ok := err.(*frame.Error); ok {
			f = frame.NewErrorFrame(streamID, errorFrame.Code, errorFrame.Data)
		} else {
			f = frame.NewErrorFrame(streamID, frame.ErrApplicationError, err.Error())
		}

		return requester.sendFrame(ctx, f)
	}

	payload := new(Payload)

	if result, ok := payloads.TryRecv(ctx); ok {
		if result.Payload != nil {
			payload = result.Payload
		} else {
			if result.Err != nil {
				defer sendError(ctx, result.Err)
			} else if result.Payload == nil {
				defer requester.sendFrame(ctx, buildCompleteFrame(streamID))
			}

			payloads = nil
		}
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
				payload, err := payloads.Recv(sender.ctx)

				if err != nil {
					return sendError(sender.ctx, err)
				} else if payload == nil {
					return requester.sendFrame(sender.ctx, buildCompleteFrame(streamID))
				}

				if err := sender.Acquire(sender.ctx); err != nil {
					return sendError(sender.ctx, err)
				}

				payloadFrame := buildPayloadFrame(streamID, false, payload)

				if err := requester.sendFrame(sender.ctx, payloadFrame); err != nil {
					return sendError(sender.ctx, err)
				}
			}
		}()
	}

	return requester.receivePayloads(ctx, streamID, receiver.C, func() {
		currentChannels.Dec()
	}), nil
}

func (requester *rSocketRequester) sendFrame(ctx context.Context, frame frame.Frame) error {
	requester.Debug("send frame",
		zap.Uint32("stream", uint32(frame.StreamID())),
		zap.Stringer("type", frame.Type()),
		zap.Reflect("frame", frame))

	err := requester.frameSender.Send(ctx, frame)

	if err == nil {
		frameSent.With(prometheus.Labels{typeLabel: frame.Type().String()}).Inc()
	}

	return err
}

func (requester *rSocketRequester) receivePayloads(
	ctx context.Context,
	streamID StreamID,
	receiver <-chan *Result,
	destructor func(),
) *PayloadStream {
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

	return &PayloadStream{results}
}

func (requester *rSocketRequester) findSender(streamID StreamID) (*resultSender, bool) {
	sender, ok := requester.senders.Load(streamID)

	if ok {
		return sender.(*resultSender), true
	}

	return nil, false
}

func (requester *rSocketRequester) findReceiver(streamID StreamID) (*resultReceiver, bool) {
	receiver, ok := requester.receivers.Load(streamID)

	if ok {
		return receiver.(*resultReceiver), true
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
			receiver.Close()
		}

		switch f := f.(type) {
		case *frame.ErrorFrame:
			defer complete(f.Err())

			return receiver.Send(ctx, Err(f.Err()))

		case *frame.CancelFrame:
			defer complete(context.Canceled)

			if sender, ok := requester.findSender(streamID); ok {
				requester.senders.Delete(streamID)
				sender.cancel()
			}

			return receiver.Send(ctx, Err(context.Canceled))

		case *frame.PayloadFrame:
			if f.Complete() {
				defer complete(nil)
			}

			if f.Next() {
				return receiver.Send(ctx, Ok(&Payload{
					HasMetadata: f.HasMetadata(),
					Metadata:    f.Metadata,
					Data:        f.Data,
				}))
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
