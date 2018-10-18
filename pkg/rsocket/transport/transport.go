package transport

import (
	"context"
	"errors"
	"net/url"

	"github.com/flier/rsocket-go/pkg/rsocket/proto"
)

// ErrUnknownScheme is returned when create Client with unknown schema
var ErrUnknownScheme = errors.New("unknown scheme")

// Transport is a minimal interface that a transport stream must implement.
type Transport interface {
	Connect(ctx context.Context) (proto.Conn, error)
}

func ForURI(target *url.URL) (transport Transport, err error) {
	switch target.Scheme {
	case "tcp":
		transport = &tcpTransport{"tcp", target.Host}
	case "ws":
	default:
		err = ErrUnknownScheme
	}

	return
}
