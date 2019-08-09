package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/nrwiersma/proxy/tcp/server"
)

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

// Transfer is an HTTP transfer TCP handler.
type Transfer struct {
	handler Handler
}

// NewTransfer returns a TCP to HTTP transfer handler.
func NewTransfer(h Handler) *Transfer {
	return &Transfer{
		handler: h,
	}
}

// ServeTCP serves a TCP connection.
func (t *Transfer) ServeTCP(ctx context.Context, conn server.Conn) {
	bufr := newBufioReader(conn)
	defer putBufioReader(bufr)

	req, err := t.readRequest(bufr)
	if err != nil {
		const errResp string = "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n400 Bad Request"

		conn.Write([]byte(errResp))
		conn.Close()
		return
	}
	req.RemoteAddr = conn.RemoteAddr()

	resp := t.handler.ServeHTTP(ctx, req)
	if resp == nil {
		return
	}

	if err := resp.Write(conn); err != nil {
		conn.Close()
	}

	if resp.Close || req.Close {
		conn.Close()
	}
}

func (t *Transfer) readRequest(r *bufio.Reader) (*Request, error) {
	tpr := newTextProtoReader(r)
	defer putTextprotoReader(tpr)

	req := &Request{}

	// Request line
	s, err := tpr.ReadLine()
	if err != nil {
		return nil, err
	}

	req.Method, req.RequestURI, req.Proto, err = t.parseRequestLine(s)
	if err != nil {
		return nil, err
	}
	if !t.validMethod(req.Method) {
		return nil, errors.New("http: invalid method")
	}

	req.URL, err = url.ParseRequestURI(req.RequestURI)
	if err != nil {
		return nil, err
	}

	// Headers
	header, err := tpr.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	req.Header = Header(header)

	req.Host = req.Header.Get("Host")
	if req.Host == "" {
		req.Host = req.URL.Host
	}

	req.Close = strings.ToLower(req.Header.Get("Connection")) == "close"

	n, err := t.parseContentLength(req)
	if err != nil {
		return nil, err
	}

	// Body
	if n > 0 {
		b, err := ioutil.ReadAll(io.LimitReader(r, n))
		if err != nil {
			return nil, err
		}

		req.Body = bytes.NewReader(b)
	}

	return req, nil
}

// parseRequestLine parses the request line like "GET /test HTTP/1.1".
func (t *Transfer) parseRequestLine(s string) (string, string, string, error) {
	parts := strings.SplitN(s, " ", 3) // method uri proto
	if len(parts) != 3 {
		return "", "", "", errors.New("http: invalid request")
	}

	return parts[0], parts[1], parts[2], nil
}

func (t *Transfer) validMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE":
		return true

	default:
		return false
	}
}

func (t *Transfer) parseContentLength(r *Request) (int64, error) {
	cls := r.Header["Content-Length"]
	if len(cls) > 1 {
		return -1, errors.New("http: multiple content lengths are not allowed")
	}

	var cl string
	if len(cls) == 1 {
		cl = cls[0]
	}

	if cl != "" {
		cl = strings.TrimSpace(cl)
		if cl == "" {
			return -1, nil
		}

		n, err := strconv.ParseInt(cl, 10, 64)
		if err != nil || n < 0 {
			return -1, errors.New("http: bad content length")
		}
		return n, nil
	}

	return 0, nil
}
