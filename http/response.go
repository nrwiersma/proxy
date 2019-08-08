package http

import (
	"fmt"
	"io"
)

// Response is an HTTP response.
type Response struct {
	StatusCode int

	StatusText string

	Proto string

	Header Header

	Body io.Reader

	Close bool

	Error error
}

// Write writes the response the the writer.
func (r *Response) Write(w io.Writer) error {
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
