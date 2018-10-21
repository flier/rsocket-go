package transport

import (
	"context"
	"net"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	"go.uber.org/zap"
)

type tcpTransport struct {
	*zap.Logger
	network string
	address string
}

func (transport *tcpTransport) Connect(ctx context.Context) (proto.Conn, error) {
	dialer := new(net.Dialer)

	conn, err := dialer.DialContext(ctx, transport.network, transport.address)

	if err != nil {
		return nil, err
	}

	return &tcpConn{
		transport.Logger,
		conn.(*net.TCPConn),
		proto.NewFramer(transport.Logger, conn),
	}, nil
}

type tcpConn struct {
	*zap.Logger
	*net.TCPConn
	*proto.Framer
}

var _ proto.Conn = (*tcpConn)(nil)

func (conn *tcpConn) Send(ctx context.Context, f frame.Frame) error {
	conn.Info("send frame", zap.Stringer("type", f.Type()), zap.Stringer("stream", f.StreamID()))

	n, err := conn.WriteFrame(f)

	bytesSent.Add(float64(n))

	return err
}

func (conn *tcpConn) Recv(ctx context.Context) (frame.Frame, error) {
	f, err := conn.ReadFrame()

	if err != nil {
		conn.Info("receive frame failed", zap.Error(err))
	} else {
		conn.Info("received frame", zap.Stringer("type", f.Type()), zap.Stringer("stream", f.StreamID()))
	}

	bytesRecv.Add(float64(f.Size()))

	return f, err
}
