package transport

import (
	"context"
	"errors"
	"net/url"

	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flier/rsocket-go/pkg/rsocket/proto"
)

const (
	rSocketNamespace   = "rsocket"
	transportSubsystem = "transport"
)

var (
	bytesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: rSocketNamespace,
		Subsystem: transportSubsystem,
		Name:      "bytes_sent",
		Help:      "Bytes sent",
	})
	bytesRecv = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: rSocketNamespace,
		Subsystem: transportSubsystem,
		Name:      "bytes_recv",
		Help:      "Bytes received",
	})
)

func init() {
	prometheus.MustRegister(bytesSent)
	prometheus.MustRegister(bytesRecv)
}

// ErrUnknownScheme is returned when create Client with unknown schema
var ErrUnknownScheme = errors.New("unknown scheme")

// Transport is a minimal interface that a transport stream must implement.
type Transport interface {
	Connect(ctx context.Context) (proto.Conn, error)
}

// ForURI creates transport for the target URL
func ForURI(logger *zap.Logger, target *url.URL) (transport Transport, err error) {
	switch target.Scheme {
	case "tcp":
		transport = &tcpTransport{logger.Named("tcp"), "tcp", target.Host}
	case "ws":
	default:
		err = ErrUnknownScheme
	}

	return
}
