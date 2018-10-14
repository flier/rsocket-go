package proto

import (
	"io"
)

type Connection interface {
	io.Closer

	Sender() FrameSender

	Receiver() FrameReceiver
}
