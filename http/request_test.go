package http_test

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/stretchr/testify/assert"
)

func TestRequest_Write(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Path:     "/blah",
			RawPath:  "/blah",
			RawQuery: "foo=bar#frag",
		},
		Host:  "example.com",
		Proto: "HTTP/1.1",
		Header: http.Header{
			"Host":           []string{"example.com"},
			"Content-Length": []string{"4"},
		},
		Body:       bytes.NewReader([]byte("test")),
		RequestURI: "/blah?foo=bar#frag",
		RemoteAddr: "localhost:8080",
	}

	buf := bytes.NewBuffer(nil)
	err := req.Write(buf)

	if assert.NoError(t, err) {
		want := "GET /blah?foo=bar#frag HTTP/1.1\r\nContent-Length: 4\r\nHost: example.com\r\n\r\ntest"
		assert.Equal(t, want, buf.String())
	}
}

func TestRequest_WriteNoBody(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Path:     "/blah",
			RawPath:  "/blah",
			RawQuery: "foo=bar",
			Fragment: "frag",
		},
		Host:  "example.com",
		Proto: "HTTP/1.1",
		Header: http.Header{
			"Host": []string{"example.com"},
		},
		RequestURI: "/blah?foo=bar#frag",
		RemoteAddr: "localhost:8080",
	}

	buf := bytes.NewBuffer(nil)
	err := req.Write(buf)

	if assert.NoError(t, err) {
		want := "GET /blah?foo=bar HTTP/1.1\r\nHost: example.com\r\n\r\n"
		assert.Equal(t, want, buf.String())
	}
}
