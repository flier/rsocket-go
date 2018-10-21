package proto

import (
	"context"
	"time"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

const defaultKeepaliveInterval = 500 * time.Millisecond
const defaultMaxLifetime = defaultKeepaliveInterval * 3

type KeepaliveOption struct {
	Interval    time.Duration // Time between KEEPALIVE frames that the client will send.
	MaxLifetime time.Duration // Time that a client will allow a server to not respond to a KEEPALIVE before it is assumed to be dead.
	Data        []byte
}

func NewKeepaliveOption() *KeepaliveOption {
	return &KeepaliveOption{
		defaultKeepaliveInterval, defaultMaxLifetime, nil,
	}
}

func (keepalive *KeepaliveOption) Enabled() bool {
	return keepalive.Interval > 0 && keepalive.MaxLifetime > 0
}

type KeepaliveConn struct {
	Conn
	Keepalive          *KeepaliveOption
	LastClientReceived Position
	LastServerReceived Position
	deadline           time.Time
	ticker             *time.Ticker
}

func NewKeepaliveConn(conn Conn, opts *KeepaliveOption) *KeepaliveConn {
	return &KeepaliveConn{
		conn,
		opts,
		0,
		0,
		time.Now().Add(opts.MaxLifetime),
		time.NewTicker(opts.Interval),
	}
}

func (conn *KeepaliveConn) Serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-conn.ticker.C:
			if now.Add(conn.Keepalive.Interval).After(conn.deadline) {
				go conn.Send(ctx, frame.NewKeepaliveFrame(true, conn.LastClientReceived, conn.Keepalive.Data))
			}
		}
	}
}

func (conn *KeepaliveConn) Close() error {
	conn.ticker.Stop()

	return conn.Conn.Close()
}

func (conn *KeepaliveConn) Recv(ctx context.Context) (f frame.Frame, err error) {
	ctx, cancel := context.WithDeadline(ctx, conn.deadline)
	defer cancel()

	f, err = conn.Conn.Recv(ctx)

	if err != nil {
		return
	}

	if keepaliveFrame, ok := f.(*frame.KeepaliveFrame); ok {
		conn.LastServerReceived = keepaliveFrame.LastReceived
		conn.deadline = time.Now().Add(conn.Keepalive.MaxLifetime)

		if keepaliveFrame.NeedRespond() {
			go conn.Send(ctx, frame.NewKeepaliveFrame(false, conn.LastClientReceived, keepaliveFrame.Data))
		}
	}

	return
}
