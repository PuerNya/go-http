package http

import (
	"context"
	"crypto/tls"
	"net"
)

type TLSConn interface {
	net.Conn
	NetConn() net.Conn
	Handshake() error
	HandshakeContext(ctx context.Context) error
	ConnectionState() tls.ConnectionState
}
