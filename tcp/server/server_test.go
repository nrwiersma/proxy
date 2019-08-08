package server_test

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/nrwiersma/proxy/tcp/server"
)

func newTestServer(t testing.TB, h server.Handler, opts server.Opts) (net.Addr, *server.Server) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv, err := server.New(h, opts)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err = srv.Serve(ln)
		if err != nil && err != server.ErrServerClosed {
			t.Fatal(err)
		}
	}()

	return ln.Addr(), srv
}

func TestNewServer_ErrorsOnNilHandler(t *testing.T) {
	_, err := server.New(nil, server.Opts{})

	if err == nil {
		t.Fatal("expected error, got none")
	}
}

func TestServer_ServesConnectionCloses(t *testing.T) {
	addr, srv := newTestServer(t, pingHandler{close: true}, server.Opts{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
	})
	defer srv.Close()

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal("dial error", err)
	}

	if _, err := io.WriteString(conn, "ping"); err != nil {
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
	addr, srv := newTestServer(t, pingHandler{}, server.Opts{
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
		if _, err := io.WriteString(conn, "ping"); err != nil {
			t.Fatal("write error", err)
		}

		var pong [4]byte
		if _, err := conn.Read(pong[:]); err != nil || string(pong[:]) != "pong" {
			t.Fatal("read error", err)
		}
	}
}

func TestServer_Shutdown(t *testing.T) {
	addr, srv := newTestServer(t, pingHandler{}, server.Opts{
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

	if _, err := io.WriteString(conn, "ping"); err != nil {
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

func (h pingHandler) ServeTCP(ctx context.Context, conn server.Conn) {
	var ping [4]byte
	if _, err := conn.Read(ping[:]); err != nil || string(ping[:]) != "ping" {
		conn.Close()
	}

	conn.Write([]byte("pong"))

	if h.close {
		conn.Close()
	}
}
