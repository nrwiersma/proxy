package http

import "io"

type Response struct {
	Status string

	StatusCode int

	Proto string

	Header Header

	Body io.Reader
}

func (r *Response) Write(w io.Writer) error {
	return nil
}
