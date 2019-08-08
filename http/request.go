package http

import (
	"fmt"
	"io"
	"net/url"
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
