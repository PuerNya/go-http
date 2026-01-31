package http

import (
	"bufio"
	"io"
	"net/http"
	"net/textproto"
	_ "unsafe"
)

//go:linkname readCookies net/http.readCookies
func readCookies(h http.Header, filter string) []*http.Cookie

//go:linkname sanitizeCookieName net/http.sanitizeCookieName
func sanitizeCookieName(n string) string

//go:linkname sanitizeCookieValue net/http.sanitizeCookieValue
func sanitizeCookieValue(v string, quoted bool) string

//go:linkname readSetCookies net/http.readSetCookies
func readSetCookies(h http.Header) []*http.Cookie

//go:linkname requestWrite net/http.(*Request).write
func requestWrite(req *http.Request, w io.Writer, usingProxy bool, extraHeaders http.Header, waitForContinue func() bool) (err error)

//go:linkname newTextprotoReader net/http.newTextprotoReader
func newTextprotoReader(br *bufio.Reader) *textproto.Reader

//go:linkname putTextprotoReader net/http.putTextprotoReader
func putTextprotoReader(r *textproto.Reader)

//go:linkname fixLength net/http.fixLength
func fixLength(isResponse bool, status int, requestMethod string, header http.Header, chunked bool) (n int64, err error)

//go:linkname parseContentLength net/http.parseContentLength
func parseContentLength(clHeaders []string) (int64, error)

//go:linkname fixTrailer net/http.fixTrailer
func fixTrailer(header http.Header, chunked bool) (http.Header, error)
