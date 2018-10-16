package client

import (
	"context"
	"errors"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
)

var (
	ErrDisconnected = errors.New("disconnected")
)

type Client interface{}

type rSocketClient struct {
	transport   Transport
	version     proto.Version
	resumeToken proto.Token
	conn        proto.Conn
	handshaking bool
}

func newClient(transport Transport, version proto.Version) *rSocketClient {
	return &rSocketClient{
		transport:   transport,
		version:     version,
		resumeToken: nil,
		conn:        nil,
		handshaking: false,
	}
}

// Disconnect the underlying transport.
func (client *rSocketClient) Close() error {
	if client.conn != nil {
		return client.conn.Close()
	}

	return nil
}

func (client *rSocketClient) Connect(ctx context.Context) (err error) {
	if client.conn != nil {
		client.conn.Close()
	}

	client.conn, err = client.transport.Connect(ctx)

	go func() error {
		defer client.Close()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case f := <-client.conn.Receiver():
				if err := client.handleFrame(f); err != nil {
					return err
				}
			}
		}
	}()

	return
}

func (client *rSocketClient) handleFrame(frame frame.Frame) error {
	return nil
}

func (client *rSocketClient) Setup(ctx context.Context, setup Setup) (err error) {
	if client.conn == nil {
		return ErrDisconnected
	}

	client.version = setup.Version
	client.resumeToken = setup.ResumeToken
	client.handshaking = setup.Lease

	setupFrame := setup.build()

	client.conn.Sender() <- setupFrame

	return
}

// Resumes the client's connection.
//
// If the client was previously connected this will attempt a warm-resumption.
// Otherwise this will attempt a cold-resumption.
func (client *rSocketClient) Resume() error {
	return nil
}
