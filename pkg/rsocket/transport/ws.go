package transport

import (
	"context"
	"net/http"
	"net/url"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	ws "github.com/gorilla/websocket"
)

type wsTransport struct {
	target *url.URL
	header http.Header
}

func (transport *wsTransport) Connect(ctx context.Context) (proto.Conn, error) {
	conn, _, err := ws.DefaultDialer.Dial(transport.target.String(), transport.header)

	if err != nil {
		return nil, err
	}

	return &wsConn{conn}, nil
}

type wsConn struct {
	*ws.Conn
}

var _ proto.Conn = (*wsConn)(nil)

func (conn *wsConn) Send(ctx context.Context, frame frame.Frame) error {
	return nil
}

func (conn *wsConn) Recv(ctx context.Context) (frame.Frame, error) {
	return nil, nil
}
