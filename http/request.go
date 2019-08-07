package http

import (
	"io"
	"net/url"
)

// Header are HTTP headers.
type Header map[string][]string

//TODO: flesh our header a bit

// Request is an HTTP request.
type Request struct {
	// Method is the HTTP method on the request.
	Method string

	// URL is the request URL.
	URL *url.URL

	// Proto is the HTTP protocol version.
	Proto string

	// Header contains the request headers.
	Header Header

	// Body is the request body.
	Body io.Reader

	// RequestURI is the request URI.
	RequestURI string
}

// Write writes the request to a writer.
func (r *Request) Write(w io.Writer) error {
	return nil
}
