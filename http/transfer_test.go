package http_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransfer_ServeTCP(t *testing.T) {
	req := "GET /blah?foo=bar#frag HTTP/1.1\r\nContent-Length: 4\r\nHost: example.com\r\n\r\ntest"
	buf := bytes.NewBuffer(nil)

	conn := new(MockConn)
	conn.On("Read", mock.Anything).Run(func(args mock.Arguments) {
		p := args.Get(0).([]byte)
		for i, r := range req {
			p[i] = byte(r)
		}
	}).Return(len(req), nil)
	conn.On("RemoteAddr").Return("localhost:8080")
	call := conn.On("Write", mock.Anything)
	call.Run(func(args mock.Arguments) {
		p := args.Get(0).([]byte)
		buf.Write(p)
		call.Return(len(p), nil)
	})

	h := http.HandlerFunc(func(ctx context.Context, r *http.Request) *http.Response {
		uri := r.URL.RequestURI()
		body, _ := ioutil.ReadAll(r.Body)
		r.Body = nil
		r.URL = nil

		wantReq := &http.Request{
			Method: "GET",
			Host:  "example.com",
			Proto: "HTTP/1.1",
			Header: http.Header{
				"Host":           []string{"example.com"},
				"Content-Length": []string{"4"},
			},
			RequestURI: "/blah?foo=bar#frag",
			RemoteAddr: "localhost:8080",
		}
		assert.Equal(t, wantReq, r)
		assert.Equal(t, "/blah?foo=bar#frag", uri)
		assert.Equal(t, []byte("test"), body)

		return &http.Response{
			StatusCode: 200,
			StatusText: "OK",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"Host":           []string{"example.com"},
				"Content-Length": []string{"4"},
			},
			Body:  bytes.NewReader([]byte("test")),
			Close: false,
			Error: nil,
		}
	})

	trans := http.NewTransfer(h)

	trans.ServeTCP(context.Background(), conn)

	conn.AssertExpectations(t)

	resp := "HTTP/1.1 200 OK\r\nContent-Length: 4\r\nHost: example.com\r\n\r\ntest"
	assert.Equal(t, resp, buf.String())
}

func TestTransfer_ServeTCPBadRequest(t *testing.T) {
	req := "something\r\n"
	buf := bytes.NewBuffer(nil)

	conn := new(MockConn)
	conn.On("Read", mock.Anything).Run(func(args mock.Arguments) {
		p := args.Get(0).([]byte)
		for i, r := range req {
			p[i] = byte(r)
		}
	}).Return(len(req), nil)
	call := conn.On("Write", mock.Anything)
	call.Run(func(args mock.Arguments) {
		p := args.Get(0).([]byte)
		buf.Write(p)
		call.Return(len(p), nil)
	})
	conn.On("Close").Return(nil)

	h := http.HandlerFunc(func(ctx context.Context, r *http.Request) *http.Response {
		return &http.Response{}
	})

	trans := http.NewTransfer(h)

	trans.ServeTCP(context.Background(), conn)

	conn.AssertExpectations(t)

	resp := "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n400 Bad Request"
	assert.Equal(t, resp, buf.String())
}

type MockConn struct {
	mock.Mock
}

func (m *MockConn) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockConn) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConn) RemoteAddr() string {
	args := m.Called()
	return args.Get(0).(string)
}
