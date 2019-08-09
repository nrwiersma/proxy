package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nrwiersma/proxy/http"
)

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Public",
	"Proxy-Authenticate",
	"Transfer-Encoding",
	"Upgrade",
}

var bufioReaderPool sync.Pool

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}

	return bufio.NewReader(r)
}

func putBufioReader(r *bufio.Reader) {
	r.Reset(nil)
	bufioReaderPool.Put(r)
}

var textprotoReaderPool sync.Pool

func newTextProtoReader(r *bufio.Reader) *textproto.Reader {
	if v := textprotoReaderPool.Get(); v != nil {
		br := v.(*textproto.Reader)
		br.R = r
		return br
	}

	return textproto.NewReader(r)
}

func putTextprotoReader(r *textproto.Reader) {
	r.R = nil
	textprotoReaderPool.Put(r)
}

type ReverseProxy struct {
	addr    string
	secure  bool
	dialer  func(ctx context.Context, network, addr string) (net.Conn, error)
	tlsConf *tls.Config
}

func New(addr string, secure bool) (*ReverseProxy, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	// This should actually be a connection pool
	dialer := (&net.Dialer{
		Timeout:   1 * time.Second,
		KeepAlive: 3 * time.Second,
	}).DialContext

	return &ReverseProxy{
		addr:   tcpAddr.String(),
		secure: secure,
		dialer: dialer,
	}, nil
}

func (p *ReverseProxy) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	conn, err := p.dialer(ctx, "tcp", p.addr)
	if err != nil {
		return &http.Response{StatusCode: 502, StatusText: "Bad Gateway", Error: err}
	}

	// TODO: TLS

	reqUp := r.Header.Get("Upgrade")

	p.removeConnectionHeaders(r.Header)
	p.removeHopByHopHeaders(r.Header)
	p.addForwardedHeader(r)

	if reqUp != "" {
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", reqUp)
	}

	if err := r.Write(conn); err != nil {
		return &http.Response{StatusCode: 502, StatusText: "Bad Gateway", Error: err}
	}

	bufr := newBufioReader(conn)
	defer putBufioReader(bufr)

	resp, err := p.readResponse(bufr)
	if err != nil {
		return &http.Response{StatusCode: 502, StatusText: "Bad Gateway", Error: err}
	}

	// Handle connection upgrade
	if resp.StatusCode == 101 {
		// TODO: Handle connection upgrade

		return nil
	}

	p.removeConnectionHeaders(resp.Header)
	p.removeHopByHopHeaders(r.Header)

	return resp
}

func (p *ReverseProxy) readResponse(r *bufio.Reader) (*http.Response, error) {
	tpr := newTextProtoReader(r)
	defer putTextprotoReader(tpr)

	resp := &http.Response{}

	// Status line
	s, err := tpr.ReadLine()
	if err != nil {
		return nil, err
	}

	resp.StatusCode, resp.StatusText, resp.Proto, err = p.parseStatusLine(s)
	if err != nil {
		return nil, err
	}

	// Headers
	header, err := tpr.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	resp.Header = http.Header(header)

	n, err := p.parseContentLength(resp)
	if err != nil {
		return nil, err
	}

	// Body
	if n > 0 {
		b, err := ioutil.ReadAll(io.LimitReader(r, n))
		if err != nil {
			return nil, err
		}

		resp.Body = bytes.NewReader(b)
	}

	return resp, nil
}

// parseRequestLine parses the request line like "HTTP/1.1 200 OK".
func (p *ReverseProxy) parseStatusLine(s string) (int, string, string, error) {
	parts := strings.SplitN(s, " ", 3) // proto code text
	if len(parts) != 3 {
		return 0, "", "", errors.New("proxy: invalid response")
	}

	code, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, "", "", err
	}

	return int(code), parts[2], parts[0], nil
}

func (p *ReverseProxy) parseContentLength(r *http.Response) (int64, error) {
	cls := r.Header["Content-Length"]
	if len(cls) > 1 {
		return -1, errors.New("proxy: multiple content lengths are not allowed")
	}

	var cl string
	if len(cls) == 1 {
		cl = cls[0]
	}

	if r.StatusCode == 204 || r.StatusCode/100 == 1 {
		return 0, nil
	}

	if cl != "" {
		cl = strings.TrimSpace(cl)
		if cl == "" {
			return -1, nil
		}

		n, err := strconv.ParseInt(cl, 10, 64)
		if err != nil || n < 0 {
			return -1, errors.New("proxy: bad content length")
		}
		return n, nil
	}

	return 0, nil
}

func (p *ReverseProxy) removeConnectionHeaders(h http.Header) {
	if c := h.Get("Connection"); c != "" {
		for _, name := range strings.Split(c, ",") {
			if name = strings.TrimSpace(name); name == "" {
				continue
			}
			h.Del(name)
		}
	}
}

func (p *ReverseProxy) removeHopByHopHeaders(h http.Header) {
	for _, name := range hopByHopHeaders {
		h.Del(name)
	}
}

func (p *ReverseProxy) addForwardedHeader(r *http.Request) {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ip = xff + ", " + ip
		}
		r.Header.Set("X-Forward-For", ip)
	}
}
