package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	ws "github.com/gorilla/websocket"
)

type wsTransport struct {
	target *url.URL
	header http.Header
}

func (transport *wsTransport) Connect(ctx context.Context) (proto.Connection, error) {
	conn, _, err := ws.DefaultDialer.Dial(transport.target.String(), transport.header)

	if err != nil {
		return nil, err
	}

	return &wsConn{conn}, nil
}

type wsConn struct {
	*ws.Conn
}

func (conn *wsConn) Sender() proto.FrameSender {
	return nil
}

func (conn *wsConn) Receiver() proto.FrameReceiver {
	return nil
}
