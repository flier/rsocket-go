package client

import (
	"context"
	"net"

	"github.com/flier/rsocket-go/pkg/rsocket/proto"
)

type tcpTransport struct {
	network string
	address string
}

func (transport *tcpTransport) Connect(ctx context.Context) (proto.Conn, error) {
	addr, err := net.ResolveTCPAddr(transport.network, transport.address)

	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)

	if err != nil {
		return nil, err
	}

	return &tcpConn{conn}, nil
}

type tcpConn struct {
	*net.TCPConn
}

var _ proto.Conn = (*tcpConn)(nil)

func (conn *tcpConn) Sender() proto.FrameSender {
	return nil
}

func (conn *tcpConn) Receiver() proto.FrameReceiver {
	return nil
}
