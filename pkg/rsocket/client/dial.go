package client

import (
	"context"
	"net/url"
	"time"

	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	"github.com/flier/rsocket-go/pkg/rsocket/transport"
	"go.uber.org/zap"
)

const defaultStreamRequestLimit = 128

// DialOption configures a Dialer before it starts to connect the target URL.
type DialOption func(*Dialer)

// WithLogger configure the logger
func WithLogger(logger *zap.Logger) DialOption {
	return func(dialer *Dialer) {
		dialer.Logger = logger
	}
}

// WithProtocolVersion configure the protocol version
func WithProtocolVersion(version proto.Version) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Version = version
	}
}

// WithResumeToken configure the resume token
func WithResumeToken(token proto.Token) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.ResumeToken = token
	}
}

// WithKeepalive configure keep-alive options
func WithKeepalive(keepalive time.Duration) DialOption {
	return func(dialer *Dialer) {
		dialer.Keepalive.Interval = keepalive
	}
}

// WithMaxLifetime configure max lifetime options
func WithMaxLifetime(maxLifetime time.Duration) DialOption {
	return func(dialer *Dialer) {
		dialer.Keepalive.MaxLifetime = maxLifetime
	}
}

// WithMetadataMimeType configure metadata payloads MIME type of RSocket
func WithMetadataMimeType(metadataMimeType string) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.MetadataMimeType = metadataMimeType
	}
}

// WithDataMimeType configure data payloads MIME type of RSocket
func WithDataMimeType(dataMimeType string) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.DataMimeType = dataMimeType
	}
}

// WithSetupPayload sets Setup frame payload of RSocket
func WithSetupPayload(payload *proto.Payload) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Payload = payload
	}
}

// WithFragment configure frame fragmentation and reassembly of requester RSocket
func WithFragment(mtu uint) DialOption {
	return func(dialer *Dialer) {
		dialer.Fragment.MTU = mtu
	}
}

// WithLease configure lease support
func WithLease(ttl time.Duration, requests uint) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Lease = requests > 0
		dialer.Lease.TimeToLive = ttl
		dialer.Lease.Requests = requests
	}
}

// Dial connects to the target URL.
func Dial(target *url.URL, opts ...DialOption) (clnt Client, err error) {
	return newDialer(opts...).Dial(target)
}

// DialContext connects to the target URL using the provided context.
func DialContext(ctx context.Context, target *url.URL, opts ...DialOption) (clnt Client, err error) {
	return newDialer(opts...).DialContext(ctx, target)
}

// A Dialer contains options for connecting to a target URL.
type Dialer struct {
	*zap.Logger
	Setup              *proto.SetupOption
	Lease              *proto.LeaseOption
	Keepalive          *proto.KeepaliveOption
	Fragment           *proto.FragmentOption
	StreamRequestLimit uint
}

func newDialer(opts ...DialOption) *Dialer {
	dialer := &Dialer{
		zap.NewNop(),
		proto.NewSetupOption(),
		proto.NewLeaseOption(),
		proto.NewKeepaliveOption(),
		proto.NewFragmentOption(),
		defaultStreamRequestLimit,
	}

	for _, opt := range opts {
		opt(dialer)
	}

	return dialer
}

// Dial connects to the target URL.
func (dialer *Dialer) Dial(target *url.URL) (Client, error) {
	return dialer.DialContext(context.Background(), target)
}

// DialContext connects to the target URL using the provided context.
func (dialer *Dialer) DialContext(ctx context.Context, target *url.URL) (client Client, err error) {
	var t transport.Transport

	if t, err = transport.ForURI(dialer.Logger, target); err != nil {
		return
	}

	clnt := newClient(dialer, t)

	go clnt.Serve(ctx)

	client = clnt

	return
}
