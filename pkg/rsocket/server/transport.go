package transport

import (
	"errors"
	"net/url"
)

var ErrUnknownScheme = errors.New("unknown scheme")

type Transport interface{}

func Listen(target *url.URL) (transport Transport, err error) {
	switch target.Scheme {
	case "tcp":
	case "ssl":
	case "ws":
	case "wss":
	default:
		err = ErrUnknownScheme
	}

	return
}
