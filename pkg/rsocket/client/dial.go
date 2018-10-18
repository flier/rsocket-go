package client

import (
	"context"
	"net/url"
	"time"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	"github.com/flier/rsocket-go/pkg/rsocket/transport"
)

const defaultKeepalive = 500 * time.Millisecond
const defaultMaxLifetime = defaultKeepalive * 3
const defaultMetadataMimeType = "application/json"
const defaultDataMimeType = "application/binary"
const defaultStreamRequestLimit = 128

// DialOption configures a Dialer before it starts to connect the target URL.
type DialOption func(*Dialer)

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
		dialer.Setup.Keepalive = keepalive
	}
}

// WithMaxLifetime configure max lifetime options
func WithMaxLifetime(maxLifetime time.Duration) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.MaxLifetime = maxLifetime
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
func WithSetupPayload(payload proto.Payload) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Payload = payload
	}
}

// WithFragment configure frame fragmentation and reassembly of requester RSocket
func WithFragment(mtu uint) DialOption {
	return func(dialer *Dialer) {
		dialer.Fragment.Mtu = mtu
	}
}

// WithLease configure lease support
func WithLease(requests uint) DialOption {
	return func(dialer *Dialer) {
		dialer.Setup.Lease = requests > 0
		dialer.Lease.Requests = requests
	}
}

// Setup options
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

func (setup *Setup) buildFrame() *frame.SetupFrame {
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

// Lease options
type Lease struct {
	TimeToLive time.Duration // Time for validity of LEASE from time of reception.
	Requests   uint          // Number of Requests that may be sent until next LEASE.
}

// Fragment options
type Fragment struct {
	Mtu uint
}

// Dial connects to the target URL.
func Dial(target *url.URL, opts ...DialOption) (clnt Client, err error) {
	return new(Dialer).withOptions(opts).Dial(target)
}

// DialContext connects to the target URL using the provided context.
func DialContext(ctx context.Context, target *url.URL, opts ...DialOption) (clnt Client, err error) {
	return new(Dialer).withOptions(opts).DialContext(ctx, target)
}

// A Dialer contains options for connecting to a target URL.
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

// Dial connects to the target URL.
func (dialer *Dialer) Dial(target *url.URL) (Client, error) {
	return dialer.DialContext(context.Background(), target)
}

// DialContext connects to the target URL using the provided context.
func (dialer *Dialer) DialContext(ctx context.Context, target *url.URL) (client Client, err error) {
	var t transport.Transport

	if t, err = transport.ForURI(target); err != nil {
		return
	}

	clnt := newClient(t, dialer.Setup.Version)

	if err = clnt.doConnect(ctx); err != nil {
		return
	}

	if err = clnt.sendSetup(ctx, dialer.Setup); err != nil {
		return
	}

	if dialer.Setup.Lease {
		var f frame.Frame

		f, err = clnt.Recv(ctx)

		if err != nil {
			return
		}

		switch f := f.(type) {
		case *frame.LeaseFrame:

		case *frame.ErrorFrame:
			err = f.Err()

		default:
			err = clnt.handleFrame(ctx, f)
		}

		if err != nil {
			return
		}
	}

	client = clnt

	return
}
