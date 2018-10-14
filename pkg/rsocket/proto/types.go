package proto

import (
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

type Version = frame.Version
type Metadata = frame.Metadata
type Token = frame.Token

type FrameSender chan<- frame.Frame
type FrameReceiver <-chan frame.Frame
