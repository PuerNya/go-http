package http

import "io"

// The interface is implemented by the http.ResponseWriter.
type Streamer interface {
	Stream() Stream
}

type Stream interface {
	io.Reader
	io.Writer
}
