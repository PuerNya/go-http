// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// HTTP Request reading and parsing.

package http

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/puernya/go-http/internal/ascii"

	"golang.org/x/net/idna"
)

func badStringError(what, val string) error { return fmt.Errorf("%s %q", what, val) }

// errMissingHost is returned by Write when there is no Host or URL present in
// the Request.
var errMissingHost = errors.New("http: Request.Write on Request with no Host or URL set")

// Return value if nonempty, def otherwise.
func valueOrDefault(value, def string) string {
	if value != "" {
		return value
	}
	return def
}

func idnaASCII(v string) (string, error) {
	// TODO: Consider removing this check after verifying performance is okay.
	// Right now punycode verification, length checks, context checks, and the
	// permissible character tests are all omitted. It also prevents the ToASCII
	// call from salvaging an invalid IDN, when possible. As a result it may be
	// possible to have two IDNs that appear identical to the user where the
	// ASCII-only version causes an error downstream whereas the non-ASCII
	// version does not.
	// Note that for correct ASCII IDNs ToASCII will only do considerably more
	// work, but it will not cause an allocation.
	if ascii.Is(v) {
		return v, nil
	}
	return idna.Lookup.ToASCII(v)
}

func validMethod(method string) bool {
	/*
	     Method         = "OPTIONS"                ; Section 9.2
	                    | "GET"                    ; Section 9.3
	                    | "HEAD"                   ; Section 9.4
	                    | "POST"                   ; Section 9.5
	                    | "PUT"                    ; Section 9.6
	                    | "DELETE"                 ; Section 9.7
	                    | "TRACE"                  ; Section 9.8
	                    | "CONNECT"                ; Section 9.9
	                    | extension-method
	   extension-method = token
	     token          = 1*<any CHAR except CTLs or separators>
	*/
	return isToken(method)
}

func readRequest(b *bufio.Reader) (req *http.Request, err error) {
	tp := newTextprotoReader(b)
	defer putTextprotoReader(tp)

	req = new(http.Request)

	// First line: GET /index.html HTTP/1.0
	var s string
	if s, err = tp.ReadLine(); err != nil {
		return nil, err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	var ok bool
	req.Method, req.RequestURI, req.Proto, ok = parseRequestLine(s)
	if !ok {
		return nil, badStringError("malformed HTTP request", s)
	}
	if !validMethod(req.Method) {
		return nil, badStringError("invalid method", req.Method)
	}
	rawurl := req.RequestURI
	if req.ProtoMajor, req.ProtoMinor, ok = http.ParseHTTPVersion(req.Proto); !ok {
		return nil, badStringError("malformed HTTP version", req.Proto)
	}

	// CONNECT requests are used two different ways, and neither uses a full URL:
	// The standard use is to tunnel HTTPS through an HTTP proxy.
	// It looks like "CONNECT www.google.com:443 HTTP/1.1", and the parameter is
	// just the authority section of a URL. This information should go in req.URL.Host.
	//
	// The net/rpc package also uses CONNECT, but there the parameter is a path
	// that starts with a slash. It can be parsed with the regular URL parser,
	// and the path will end up in req.URL.Path, where it needs to be in order for
	// RPC to work.
	justAuthority := req.Method == "CONNECT" && !strings.HasPrefix(rawurl, "/")
	if justAuthority {
		rawurl = "http://" + rawurl
	}

	if req.URL, err = url.ParseRequestURI(rawurl); err != nil {
		return nil, err
	}

	if justAuthority {
		// Strip the bogus "http://" back off.
		req.URL.Scheme = ""
	}

	// Subsequent lines: Key: value.
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	req.Header = http.Header(mimeHeader)
	if len(req.Header["Host"]) > 1 {
		return nil, fmt.Errorf("too many Host headers")
	}

	// RFC 7230, section 5.3: Must treat
	//	GET /index.html HTTP/1.1
	//	Host: www.google.com
	// and
	//	GET http://www.google.com/index.html HTTP/1.1
	//	Host: doesntmatter
	// the same. In the second case, any Host line is ignored.
	req.Host = req.URL.Host
	if req.Host == "" {
		req.Host = getFromHeader(req.Header, "Host")
	}

	fixPragmaCacheControl(req.Header)

	req.Close = shouldClose(req.ProtoMajor, req.ProtoMinor, req.Header, false)

	err = readTransfer(req, b)
	if err != nil {
		return nil, err
	}

	if isH2UpgradeRequest(req) {
		// Because it's neither chunked, nor declared:
		req.ContentLength = -1

		// We want to give handlers a chance to hijack the
		// connection, but we need to prevent the Server from
		// dealing with the connection further if it's not
		// hijacked. Set Close to ensure that:
		req.Close = true
	}
	return req, nil
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	method, rest, ok1 := strings.Cut(line, " ")
	requestURI, proto, ok2 := strings.Cut(rest, " ")
	if !ok1 || !ok2 {
		return "", "", "", false
	}
	return method, requestURI, proto, true
}

func requestExpectsContinue(req *http.Request) bool {
	return hasToken(getFromHeader(req.Header, "Expect"), "100-continue")
}

func requestWantsHttp10KeepAlive(r *http.Request) bool {
	if r.ProtoMajor != 1 || r.ProtoMinor != 0 {
		return false
	}
	return hasToken(getFromHeader(r.Header, "Connection"), "keep-alive")
}

func requestWantsClose(req *http.Request) bool {
	if req.Close {
		return true
	}
	return hasToken(getFromHeader(req.Header, "Connection"), "close")
}

func closeRequestBody(req *http.Request) error {
	if req.Body == nil {
		return nil
	}
	return req.Body.Close()
}

func isH2UpgradeRequest(req *http.Request) bool {
	return req.Method == "PRI" && len(req.Header) == 0 && req.URL.Path == "*" && req.Proto == "HTTP/2.0"
}

func isReplayableRequest(req *http.Request) bool {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		switch valueOrDefault(req.Method, "GET") {
		case "GET", "HEAD", "OPTIONS", "TRACE":
			return true
		}
		// The Idempotency-Key, while non-standard, is widely used to
		// mean a POST or other request is idempotent. See
		// https://golang.org/issue/19943#issuecomment-421092421
		_, exists := req.Header["Idempotency-Key"]
		if !exists {
			_, exists = req.Header["X-Idempotency-Key"]
		}
		if exists {
			return true
		}
	}
	return false
}

func requestOutgoingLength(req *http.Request) int64 {
	if req.Body == nil || req.Body == http.NoBody {
		return 0
	}
	if req.ContentLength != 0 {
		return req.ContentLength
	}
	return -1
}

func requestRequiresHTTP1(req *http.Request) bool {
	return hasToken(req.Header.Get("Connection"), "upgrade") &&
		ascii.EqualFold(req.Header.Get("Upgrade"), "websocket")
}

func checkRequestBodyError(err error) (error, bool) {
	t := reflect.TypeOf(err)
	if t == nil {
		return nil, false
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	if t.PkgPath() != "net/http" {
		return nil, false
	}
	name := t.Name()
	if name != "requestBodyReadError" {
		return nil, false
	}
	return errors.New(err.Error()), true
}
