package client

import (
	"context"
	"errors"
	"net/url"

	"github.com/flier/rsocket-go/pkg/rsocket/proto"
)

var ErrUnknownScheme = errors.New("unknown scheme")

type Transport interface {
	Connect(ctx context.Context) (proto.Connection, error)
}

func clientForUri(target *url.URL) (transport Transport, err error) {
	switch target.Scheme {
	case "tcp":
		transport = &tcpTransport{"tcp", target.Host}
	case "ws":
	default:
		err = ErrUnknownScheme
	}

	return
}
