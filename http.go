// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"
)

// Protocols is a set of HTTP protocols.
// The zero value is an empty set of protocols.
//
// The supported protocols are:
//
//   - HTTP1 is the HTTP/1.0 and HTTP/1.1 protocols.
//     HTTP1 is supported on both unsecured TCP and secured TLS connections.
//
//   - HTTP2 is the HTTP/2 protcol over a TLS connection.
//
//   - UnencryptedHTTP2 is the HTTP/2 protocol over an unsecured TCP connection.
type Protocols struct {
	bits uint8
}

const (
	protoHTTP1 = 1 << iota
	protoHTTP2
	protoUnencryptedHTTP2
)

// HTTP1 reports whether p includes HTTP/1.
func (p Protocols) HTTP1() bool { return p.bits&protoHTTP1 != 0 }

// SetHTTP1 adds or removes HTTP/1 from p.
func (p *Protocols) SetHTTP1(ok bool) { p.setBit(protoHTTP1, ok) }

// HTTP2 reports whether p includes HTTP/2.
func (p Protocols) HTTP2() bool { return p.bits&protoHTTP2 != 0 }

// SetHTTP2 adds or removes HTTP/2 from p.
func (p *Protocols) SetHTTP2(ok bool) { p.setBit(protoHTTP2, ok) }

// UnencryptedHTTP2 reports whether p includes unencrypted HTTP/2.
func (p Protocols) UnencryptedHTTP2() bool { return p.bits&protoUnencryptedHTTP2 != 0 }

// SetUnencryptedHTTP2 adds or removes unencrypted HTTP/2 from p.
func (p *Protocols) SetUnencryptedHTTP2(ok bool) { p.setBit(protoUnencryptedHTTP2, ok) }

func (p *Protocols) setBit(bit uint8, ok bool) {
	if ok {
		p.bits |= bit
	} else {
		p.bits &^= bit
	}
}

func (p Protocols) String() string {
	var s []string
	if p.HTTP1() {
		s = append(s, "HTTP1")
	}
	if p.HTTP2() {
		s = append(s, "HTTP2")
	}
	if p.UnencryptedHTTP2() {
		s = append(s, "UnencryptedHTTP2")
	}
	return "{" + strings.Join(s, ",") + "}"
}

// incomparable is a zero-width, non-comparable type. Adding it to a struct
// makes that struct also non-comparable, and generally doesn't add
// any size (as long as it's first).
type incomparable [0]func()

// maxInt64 is the effective "infinite" value for the Server and
// Transport's byte-limiting readers.
const maxInt64 = 1<<63 - 1

// aLongTimeAgo is a non-zero time, far in the past, used for
// immediate cancellation of network operations.
var aLongTimeAgo = time.Unix(1, 0)

// TODO(bradfitz): move common stuff here. The other files have accumulated
// generic http stuff in random places.

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "http context value " + k.name }

// isToken reports whether v is a valid token (https://www.rfc-editor.org/rfc/rfc2616#section-2.2).
func isToken(v string) bool {
	// For historical reasons, this function is called ValidHeaderFieldName (see issue #67031).
	return httpguts.ValidHeaderFieldName(v)
}

// HTTP2Config defines HTTP/2 configuration parameters common to
// both [Transport] and [Server].
type HTTP2Config struct {
	// MaxConcurrentStreams optionally specifies the number of
	// concurrent streams that a client may have open at a time.
	// If zero, MaxConcurrentStreams defaults to at least 100.
	//
	// This parameter only applies to Servers.
	MaxConcurrentStreams int

	// StrictMaxConcurrentRequests controls whether an HTTP/2 server's
	// concurrency limit should be respected across all connections
	// to that server.
	// If true, new requests sent when a connection's concurrency limit
	// has been exceeded will block until an existing request completes.
	// If false, an additional connection will be opened if all
	// existing connections are at their limit.
	//
	// This parameter only applies to Transports.
	StrictMaxConcurrentRequests bool

	// MaxDecoderHeaderTableSize optionally specifies an upper limit for the
	// size of the header compression table used for decoding headers sent
	// by the peer.
	// A valid value is less than 4MiB.
	// If zero or invalid, a default value is used.
	MaxDecoderHeaderTableSize int

	// MaxEncoderHeaderTableSize optionally specifies an upper limit for the
	// header compression table used for sending headers to the peer.
	// A valid value is less than 4MiB.
	// If zero or invalid, a default value is used.
	MaxEncoderHeaderTableSize int

	// MaxReadFrameSize optionally specifies the largest frame
	// this endpoint is willing to read.
	// A valid value is between 16KiB and 16MiB, inclusive.
	// If zero or invalid, a default value is used.
	MaxReadFrameSize int

	// MaxReceiveBufferPerConnection is the maximum size of the
	// flow control window for data received on a connection.
	// A valid value is at least 64KiB and less than 4MiB.
	// If invalid, a default value is used.
	MaxReceiveBufferPerConnection int

	// MaxReceiveBufferPerStream is the maximum size of
	// the flow control window for data received on a stream (request).
	// A valid value is less than 4MiB.
	// If zero or invalid, a default value is used.
	MaxReceiveBufferPerStream int

	// SendPingTimeout is the timeout after which a health check using a ping
	// frame will be carried out if no frame is received on a connection.
	// If zero, no health check is performed.
	SendPingTimeout time.Duration

	// PingTimeout is the timeout after which a connection will be closed
	// if a response to a ping is not received.
	// If zero, a default of 15 seconds is used.
	PingTimeout time.Duration

	// IdleTimeout specifies how long until idle clients should be
	// closed with a GOAWAY frame. PING frames are not considered
	// activity for the purposes of IdleTimeout.
	// If zero or negative, there is no timeout.
	IdleTimeout time.Duration

	// WriteByteTimeout is the timeout after which a connection will be
	// closed if no data can be written to it. The timeout begins when data is
	// available to write, and is extended whenever any bytes are written.
	WriteByteTimeout time.Duration

	// PermitProhibitedCipherSuites, if true, permits the use of
	// cipher suites prohibited by the HTTP/2 spec.
	PermitProhibitedCipherSuites bool

	// CountError, if non-nil, is called on HTTP/2 errors.
	// It is intended to increment a metric for monitoring.
	// The errType contains only lowercase letters, digits, and underscores
	// (a-z, 0-9, _).
	CountError func(errType string)
}
