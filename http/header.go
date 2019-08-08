package http

import (
	"fmt"
	"io"
	"net/textproto"
)

// Header are HTTP headers.
type Header map[string][]string

// Get returns the first value for the given key.
func (h Header) Get(key string) string {
	return textproto.MIMEHeader(h).Get(key)
}

// Set sets the key value pair on the header.
func (h Header) Set(key, value string) {
	textproto.MIMEHeader(h).Set(key, value)
}

func (h Header) Write(w io.Writer) error {
	for k, vv := range h {
		for _, v := range vv {
			_, err := fmt.Fprintf(w, "%s :%s\r\n", k, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
