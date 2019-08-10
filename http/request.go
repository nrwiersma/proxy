package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Request is an HTTP request.
type Request struct {
	// Method is the HTTP method on the request.
	Method string

	// URL is the request URL.
	URL *url.URL

	// Host is the request host.
	Host string

	// Proto is the HTTP protocol version.
	Proto string

	// Header contains the request headers.
	Header Header

	// Body is the request body.
	Body io.Reader

	// RequestURI is the request URI.
	RequestURI string

	// RemoteAddr is the remote address of the request.
	RemoteAddr string

	// Close indicates that the request wants to close the connection.
	Close bool

	ctx context.Context
}

// Write writes the request to a writer.
func (r *Request) Write(w io.Writer) error {
	uri := r.URL.RequestURI()

	// Request Line
	_, err := fmt.Fprintf(w, "%s %s %s\r\n", r.Method, uri, r.Proto)
	if err != nil {
		return err
	}

	// Header
	if err := r.Header.Write(w); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\r\n")
	if err != nil {
		return err
	}

	// Body
	if r.Body != nil {
		if _, err := io.Copy(w, r.Body); err != nil {
			return err
		}
	}

	return nil
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

func readRequest(r *bufio.Reader) (*Request, error) {
	tpr := newTextProtoReader(r)
	defer putTextprotoReader(tpr)

	req := &Request{}

	// Request line
	s, err := tpr.ReadLine()
	if err != nil {
		return nil, err
	}

	req.Method, req.RequestURI, req.Proto, err = parseRequestLine(s)
	if err != nil {
		return nil, err
	}
	if !validMethod(req.Method) {
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

	n, err := parseContentLength(req)
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
func parseRequestLine(s string) (string, string, string, error) {
	parts := strings.SplitN(s, " ", 3) // method uri proto
	if len(parts) != 3 {
		return "", "", "", errors.New("http: invalid request")
	}

	return parts[0], parts[1], parts[2], nil
}

func validMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE":
		return true

	default:
		return false
	}
}

func parseContentLength(r *Request) (int64, error) {
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
