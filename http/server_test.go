package http_test

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/nrwiersma/proxy/http"
	"github.com/stretchr/testify/assert"
)

func newTestServer(t testing.TB, h http.Handler, opts http.Opts) (net.Addr, *http.Server) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv, err := http.NewServer(h, opts)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err = srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}()

	return ln.Addr(), srv
}

func newTestTLSServer(t testing.TB, h http.Handler, opts http.Opts) (net.Addr, *tls.Config, *http.Server) {
	cert, err := tls.LoadX509KeyPair("../testdata/cert.pem", "../testdata/key.pem")
	if err != nil {
		t.Fatal(err)
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	ln, err := tls.Listen("tcp", "localhost:0", config)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := http.NewServer(h, opts)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err = srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}()

	return ln.Addr(), config, srv
}

func TestNewServer_ErrorsOnNilHandler(t *testing.T) {
	_, err := http.NewServer(nil, http.Opts{})

	if err == nil {
		t.Fatal("expected error, got none")
	}
}

func TestServer_ServesConnectionCloses(t *testing.T) {
	addr, srv := newTestServer(t, pingHandler{close: true}, http.Opts{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
	})
	defer srv.Close()

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal("dial error", err)
	}

	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal("write error", err)
	}

	done := make(chan bool, 1)
	go func() {
		select {
		case <-time.After(5 * time.Second):
			t.Error("body not closed after 5s")
			return
		case <-done:
		}
	}()

	if _, err := ioutil.ReadAll(conn); err != nil {
		t.Fatal("read error", err)
	}
	done <- true
}

func TestServer_ServesConnectionStaysOpen(t *testing.T) {
	addr, srv := newTestServer(t, pingHandler{}, http.Opts{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
	})
	defer srv.Close()

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal("dial error", err)
	}
	defer conn.Close()

	for i := 0; i < 3; i++ {
		if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n"); err != nil {
			t.Fatal("write error", err)
		}

		pong := make([]byte, 1024)
		n, err := conn.Read(pong)
		if err != nil {
			t.Fatal("read error", err)
		}

		want := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
		assert.Equal(t, want, pong[:n])
	}
}

func TestServer_ServesTLS(t *testing.T) {
	addr, tlsConfig, srv := newTestTLSServer(t, pingHandler{}, http.Opts{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
	})
	defer srv.Close()

	conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
	if err != nil {
		t.Fatal("dial error", err)
	}
	defer conn.Close()

	for i := 0; i < 3; i++ {
		if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n"); err != nil {
			t.Fatal("write error", err)
		}

		pong := make([]byte, 1024)
		n, err := conn.Read(pong)
		if err != nil {
			t.Fatal("read error", err)
		}

		want := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
		assert.Equal(t, want, pong[:n])
	}
}

func TestServer_Shutdown(t *testing.T) {
	addr, srv := newTestServer(t, pingHandler{}, http.Opts{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
	})
	defer srv.Close()

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal("dial error", err)
	}

	go func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			t.Fatal("shutdown error", err)
		}
	}()

	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal("write error", err)
	}

	done := make(chan bool, 1)
	go func() {
		select {
		case <-time.After(5 * time.Second):
			t.Error("body not closed after 5s")
			return
		case <-done:
		}
	}()

	if _, err := ioutil.ReadAll(conn); err != nil {
		t.Fatal("read error", err)
	}
	done <- true
}

type pingHandler struct {
	close bool
}

func (h pingHandler) ServeHTTP(context.Context, *http.Request) *http.Response {
	resp := &http.Response{
		StatusCode: 200,
		StatusText: "OK",
		Header: http.Header{
			"Content-Type": []string{"text/plain; charset=utf-8"},
		},
	}

	if h.close {
		resp.Header.Set("Connection", "close")
	}

	return resp
}
