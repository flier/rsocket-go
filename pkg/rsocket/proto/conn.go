package proto

import (
	"context"
	"io"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// FrameSender sends frame.
type FrameSender interface {
	Send(ctx context.Context, frame frame.Frame) error
}

// FrameReceiver receives frame.
type FrameReceiver interface {
	Recv(ctx context.Context) (frame.Frame, error)
}

type frameChan chan frame.Frame

// Send frame to channel
func (c frameChan) Send(ctx context.Context, frame frame.Frame) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c <- frame:
		return nil
	}
}

// Recv frame from channel
func (c frameChan) Recv(ctx context.Context) (frame.Frame, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case frame := <-c:
		return frame, nil
	}
}

// Conn is a generic frame-oriented network connection.
type Conn interface {
	io.Closer

	// Sender sends a frame.
	Sender() FrameSender

	// Receiver receives a frame.
	Receiver() FrameReceiver
}
