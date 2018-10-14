package client

import (
	"context"
	"net/url"
	"time"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
)

const defaultKeepalive = 500 * time.Millisecond
const defaultMaxLifetime = defaultKeepalive * 3
const defaultMetadataMimeType = "application/json"
const defaultDataMimeType = "application/binary"
const defaultStreamRequestLimit = 128

type DialOption func(*Dialer)

func WithProtocolVersion(version proto.Version) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Version = version
	}
}

func WithResumeToken(token proto.Token) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.ResumeToken = token
	}
}

// Configure keep-alive options
func WithKeepalive(keepalive time.Duration) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Keepalive = keepalive
	}
}

// Configure max lifetime options
func WithMaxLifetime(maxLifetime time.Duration) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.MaxLifetime = maxLifetime
	}
}

// Configure metadata payloads MIME type of RSocket
func WithMetadataMimeType(metadataMimeType string) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.MetadataMimeType = metadataMimeType
	}
}

// Configure data payloads MIME type of RSocket
func WithDataMimeType(dataMimeType string) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.DataMimeType = dataMimeType
	}
}

// Sets Setup frame payload of RSocket
func WithSetupPayload(payload proto.Payload) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Payload = payload
	}
}

// Configure frame fragmentation and reassembly of requester RSocket
func WithFragment(mtu uint) DialOption {
	return func(dialer *Dialer) {
		dialer.Fragment.Mtu = mtu
	}
}

// Configure lease support
func WithLease(requests uint) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Lease = requests > 0
		dialer.Lease.Requests = requests
	}
}

type Setup struct {
	Version          proto.Version // Version number of the protocol.
	Lease            bool          // Will honor LEASE (or not).
	Keepalive        time.Duration // Time between KEEPALIVE frames that the client will send.
	MaxLifetime      time.Duration // Time that a client will allow a server to not respond to a KEEPALIVE before it is assumed to be dead.
	ResumeToken      proto.Token   // Token used for client resume identification.
	MetadataMimeType string        // MIME Type for encoding of Metadata.
	DataMimeType     string        // MIME Type for encoding of Data.
	Payload          proto.Payload // Payload describing connection capabilities of the endpoint sending the Setup header.
}

func (setup *Setup) build() *frame.SetupFrame {
	return frame.NewSetupFrame(
		setup.Version,
		setup.Lease,
		setup.Keepalive,
		setup.MaxLifetime,
		setup.ResumeToken,
		setup.MetadataMimeType,
		setup.DataMimeType,
		setup.Payload.HasMetadata,
		setup.Payload.Metadata,
		setup.Payload.Data)
}

type Lease struct {
	TimeToLive time.Duration // Time for validity of LEASE from time of reception.
	Requests   uint          // Number of Requests that may be sent until next LEASE.
}

type Fragment struct {
	Mtu uint
}

type Dialer struct {
	Setup
	Lease
	Fragment
}

func (dialer *Dialer) withOptions(opts []DialOption) *Dialer {
	for _, opt := range opts {
		opt(dialer)
	}

	return dialer
}

func (dialer *Dialer) Dial(target *url.URL) (Client, error) {
	return dialer.DialContext(context.Background(), target)
}

func Dial(target *url.URL, opts ...DialOption) (clnt Client, err error) {
	return new(Dialer).withOptions(opts).Dial(target)
}

func DialContext(ctx context.Context, target *url.URL, opts ...DialOption) (clnt Client, err error) {
	return new(Dialer).withOptions(opts).DialContext(ctx, target)
}

func (dialer *Dialer) DialContext(ctx context.Context, target *url.URL) (client Client, err error) {
	var transport Transport

	if transport, err = clientForUri(target); err != nil {
		return
	}

	clnt := newClient(transport, dialer.Setup.Version)

	if err = clnt.Connect(ctx); err != nil {
		return
	}

	if err = clnt.Setup(ctx, dialer.Setup); err != nil {
		return
	}

	client = clnt

	return
}
