package proto

import (
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// Version number of the protocol.
type Version = frame.Version

// Token used for client resume identification.
type Token = frame.Token

// Error of the protocol
type Error = frame.Error

var LatestVersion = frame.V1
