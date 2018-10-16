package proto

import (
	"io"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// FrameSender sends frame.
type FrameSender chan<- frame.Frame

// FrameReceiver receives frame.
type FrameReceiver <-chan frame.Frame

// Conn is a generic frame-oriented network connection.
type Conn interface {
	io.Closer

	// Sender sends a frame.
	Sender() FrameSender

	// Receiver receives a frame.
	Receiver() FrameReceiver
}
