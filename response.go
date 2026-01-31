// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// HTTP Response reading and parsing.

package http

import (
	"io"
	"net/http"

	"golang.org/x/net/http/httpguts"
)

func fixPragmaCacheControl(header http.Header) {
	if hp, ok := header["Pragma"]; ok && len(hp) > 0 && hp[0] == "no-cache" {
		if _, presentcc := header["Cache-Control"]; !presentcc {
			header["Cache-Control"] = []string{"no-cache"}
		}
	}
}

func isResponseBodyWritable(res *http.Response) bool {
	_, ok := res.Body.(io.Writer)
	return ok
}

// isProtocolSwitchResponse reports whether the response code and
// response header indicate a successful protocol upgrade response.
func isProtocolSwitchResp(resp *http.Response) bool {
	return isProtocolSwitchResponse(resp.StatusCode, resp.Header)
}

// isProtocolSwitchResponse reports whether the response code and
// response header indicate a successful protocol upgrade response.
func isProtocolSwitchResponse(code int, h http.Header) bool {
	return code == http.StatusSwitchingProtocols && isProtocolSwitchHeader(h)
}

// isProtocolSwitchHeader reports whether the request or response header
// is for a protocol switch.
func isProtocolSwitchHeader(h http.Header) bool {
	return h.Get("Upgrade") != "" &&
		httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade")
}
