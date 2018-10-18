package client

import (
	"context"
	"errors"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	"github.com/flier/rsocket-go/pkg/rsocket/transport"
)

var (
	// ErrDisconnected is returned when use a disconnected client
	ErrDisconnected = errors.New("disconnected")
)

// Client API
type Client interface{}

type rSocketClient struct {
	proto.Conn
	transport   transport.Transport
	version     proto.Version
	resumeToken proto.Token
	handshaking bool
}

func newClient(transport transport.Transport, version proto.Version) *rSocketClient {
	return &rSocketClient{
		Conn:        nil,
		transport:   transport,
		version:     version,
		resumeToken: nil,
		handshaking: false,
	}
}

// Disconnect the underlying transport.
func (client *rSocketClient) Close() error {
	if client.Conn != nil {
		return client.Conn.Close()
	}

	return nil
}

func (client *rSocketClient) doConnect(ctx context.Context) (err error) {
	if client.Conn != nil {
		client.Conn.Close()
	}

	client.Conn, err = client.transport.Connect(ctx)

	return
}

func (client *rSocketClient) sendSetup(ctx context.Context, setup Setup) (err error) {
	if client.Conn == nil {
		return ErrDisconnected
	}

	client.version = setup.Version
	client.resumeToken = setup.ResumeToken
	client.handshaking = setup.Lease

	setupFrame := setup.buildFrame()

	return client.Send(ctx, setupFrame)
}

func (client *rSocketClient) handleFrame(ctx context.Context, f frame.Frame) error {
	return nil
}
