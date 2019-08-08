package http_test

import (
	"bytes"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/stretchr/testify/assert"
)

func TestResponse_Write(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		StatusText: "OK",
		Proto:      "HTTP/1.1",
		Header:     http.Header{
			"Host": []string{"example.com"},
			"Content-Length": []string{"4"},
		},
		Body:        bytes.NewReader([]byte("test")),
		Close:      false,
		Error:      nil,
	}

	buf := bytes.NewBuffer(nil)
	err := resp.Write(buf)

	if assert.NoError(t, err) {
		want := "HTTP/1.1 200 OK\r\nContent-Length: 4\r\nHost: example.com\r\n\r\ntest"
		assert.Equal(t, want, buf.String())
	}
}

func TestResponse_WriteNoBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		StatusText: "OK",
		Proto:      "HTTP/1.1",
		Header:     http.Header{
			"Host": []string{"example.com"},
		},
		Close:      false,
		Error:      nil,
	}

	buf := bytes.NewBuffer(nil)
	err := resp.Write(buf)

	if assert.NoError(t, err) {
		want := "HTTP/1.1 200 OK\r\nHost: example.com\r\n\r\n"
		assert.Equal(t, want, buf.String())
	}
}
