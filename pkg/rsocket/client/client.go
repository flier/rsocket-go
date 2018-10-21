package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	"github.com/flier/rsocket-go/pkg/rsocket/transport"
	"go.uber.org/zap"
)

var (
	// ErrDisconnected is returned when use a disconnected client
	ErrDisconnected = errors.New("disconnected")
)

// Client API
type Client interface {
	io.Closer

	Requester() proto.Requester
}

type rSocketClient struct {
	*Dialer
	transport                  transport.Transport
	streamIDs                  proto.StreamIDs
	cancel                     context.CancelFunc
	requester                  proto.Requester
	c                          *sync.Cond
	LastReceivedClientPosition proto.Position
}

func newClient(opts *Dialer, transport transport.Transport) *rSocketClient {
	return &rSocketClient{
		opts,
		transport,
		proto.ClientStreamIDs(),
		nil,
		nil,
		sync.NewCond(new(sync.Mutex)),
		0,
	}
}

// Disconnect the underlying transport.
func (client *rSocketClient) Close() error {
	if client.cancel != nil {
		client.cancel()
	}

	return nil
}

func (client *rSocketClient) Requester() proto.Requester {
	client.c.L.Lock()
	defer client.c.L.Unlock()

	for client.requester == nil {
		client.c.Wait()
	}

	return client.requester
}

func (client *rSocketClient) Serve(ctx context.Context) (err error) {
	ctx, client.cancel = context.WithCancel(ctx)
	defer client.cancel()

	var current, next State

	current = &connectState{}

	for {
		next, err = current.Next(ctx, client)

		client.Debug("handle state", zap.Stringer("current", current), zap.Stringer("next", next), zap.Error(err))

		if err == nil {
			current = next
		} else {
			defer current.Close()

			if err == context.Canceled {
				return nil
			}

			if err, ok := err.(*frame.Error); ok {
				switch err.Code {
				case frame.ErrInvalidSetup, frame.ErrUnsupportedSetup, frame.ErrRejectedSetup, frame.ErrRejectedResume:
					current = &connectState{}
					continue

				default:
					// try resume connection
				}
			}

			current = &connectState{client.Setup.ResumeToken}
		}
	}
}

type State interface {
	io.Closer

	fmt.Stringer

	Next(ctx context.Context, client *rSocketClient) (next State, err error)
}

type connectState struct {
	resumeToken proto.Token
}

func (state *connectState) Close() error {
	return nil
}

func (state *connectState) String() string {
	if state.resumeToken != nil {
		return "RESUME"
	}

	return "CONNECT"
}

func (state *connectState) Next(ctx context.Context, client *rSocketClient) (next State, err error) {
	var conn proto.Conn

	if conn, err = client.transport.Connect(ctx); err != nil {
		return
	}

	keepaliveConn := proto.NewKeepaliveConn(conn, client.Keepalive)

	if err = keepaliveConn.Serve(ctx); err != nil {
		return
	}

	conn = keepaliveConn

	if state.resumeToken == nil {
		setupFrame := frame.NewSetupFrame(
			client.Setup.Version,
			client.Setup.Lease,
			client.Keepalive.Interval,
			client.Keepalive.MaxLifetime,
			client.Setup.ResumeToken,
			client.Setup.MetadataMimeType,
			client.Setup.DataMimeType,
			client.Setup.Payload.HasMetadata,
			client.Setup.Payload.Metadata,
			client.Setup.Payload.Data,
		)

		if err = conn.Send(ctx, setupFrame); err != nil {
			return
		}

		if client.Setup.Lease {
			next = &waitLeaseState{conn}
		} else {
			next = &handleFramesState{conn, nil}
		}
	} else {
		resumeFrame := frame.NewResumeFrame(
			client.Setup.Version,
			state.resumeToken,
			0,
			0,
		)

		if err = conn.Send(ctx, resumeFrame); err != nil {
			return
		}

		next = &waitResumeOkState{conn}
	}

	return
}

type waitLeaseState struct {
	proto.Conn
}

func (state *waitLeaseState) String() string {
	return "WAIT_LEASE"
}

func (state *waitLeaseState) Next(ctx context.Context, client *rSocketClient) (next State, err error) {
	var f frame.Frame

	if f, err = state.Conn.Recv(ctx); err != nil {
		return
	}

	switch f := f.(type) {
	case *frame.LeaseFrame:
		// TODO handle lease
		f = nil
		next = &handleFramesState{state.Conn, f}

	case *frame.ErrorFrame:
		err = f.Err()

	default:
		connErr := frame.ErrConnectionError.WithMessage(fmt.Sprintf("unexpected frame: %s", f))

		err = state.Conn.Send(ctx, frame.NewErrorFrame(0, connErr.Code, connErr.Data))

		if err == nil {
			err = connErr
		}
	}

	return
}

type waitResumeOkState struct {
	proto.Conn
}

func (state *waitResumeOkState) String() string {
	return "WAIT_RESUME_OK"
}

func (state *waitResumeOkState) Next(ctx context.Context, client *rSocketClient) (next State, err error) {
	var f frame.Frame

	if f, err = state.Conn.Recv(ctx); err != nil {
		return
	}

	switch f := f.(type) {
	case *frame.ResumeOkFrame:
		client.LastReceivedClientPosition = f.LastReceived

		next = &handleFramesState{state.Conn, nil}

	case *frame.ErrorFrame:
		err = f.Err()

	default:
		connErr := frame.ErrConnectionError.WithMessage(fmt.Sprintf("unexpected frame: %s", f))

		err = state.Conn.Send(ctx, frame.NewErrorFrame(0, connErr.Code, connErr.Data))

		if err == nil {
			err = connErr
		}
	}

	return
}

type handleFramesState struct {
	proto.Conn
	f frame.Frame
}

func (state *handleFramesState) String() string {
	return "FRAMES"
}

func (state *handleFramesState) Next(ctx context.Context, client *rSocketClient) (next State, err error) {
	if client.requester == nil {
		client.c.L.Lock()
		client.requester = proto.NewRequester(client.Logger, state.Conn, client.streamIDs, client.StreamRequestLimit)
		client.c.L.Unlock()

		client.c.Broadcast()
	}

	f := state.f

	if f == nil {
		if f, err = state.Conn.Recv(ctx); err != nil {
			return
		}
	}

	if err = client.requester.(proto.FrameHandler).HandleFrame(ctx, f); err != nil {
		return
	}

	state.f = nil
	next = state

	return
}
