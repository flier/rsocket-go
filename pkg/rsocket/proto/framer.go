package proto

import (
	"io"

	"go.uber.org/zap"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// A Framer reads and writes Frames.
type Framer struct {
	*zap.Logger
	*frame.Reader
	*frame.Writer
}

// NewFramer creates a Framer reads and writes Frames.
func NewFramer(logger *zap.Logger, s io.ReadWriteCloser) *Framer {
	return &Framer{
		logger.Named("framer"),
		frame.NewReader(logger, s),
		frame.NewWriter(logger, s),
	}
}
