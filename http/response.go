package http

import (
	"fmt"
	"io"
)

// Response is an HTTP response.
type Response struct {
	// StatusCode is the status of the response.
	StatusCode int

	// StatusText is the status text of the response.
	StatusText string

	// Proto is the HTTP protocol version.
	Proto string

	// Header contains the response headers.
	Header Header

	// Body is the response body.
	Body io.Reader

	// Close indicates that the response want to close the connection.
	Close bool

	// Error is the error associated with the response. This is only
	// set internally.
	Error error
}

// Write writes the response the the writer.
func (r *Response) Write(w io.Writer) error {
	if r.Proto == "" {
		r.Proto = "HTTP/1.1"
	}

	if len(r.Header) == 0 {
		r.Header = Header{
			"Content-Type": []string{"text/plain; charset=utf-8"},
			"Connection":   []string{"close"},
		}

		if r.Body == nil {
			r.Header.Set("Content-Length", "0")
		}
	}

	// Status Line
	_, err := fmt.Fprintf(w, "%s %d %s\r\n", r.Proto, r.StatusCode, r.StatusText)
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
