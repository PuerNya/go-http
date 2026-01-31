# GO-HTTP

A fork of golang std net/http, which only contains Transport && Server.

Support 3rd party tls implement.

Make sure your conn implement the `TLSConn` interface

```Golang
type TLSConn interface {
	net.Conn
	NetConn() net.Conn
	Handshake() error
	HandshakeContext(ctx context.Context) error
	ConnectionState() tls.ConnectionState
}
```