package http

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrServerClosed is returned when a connection is
	// attempted on a closed server.
	ErrServerClosed = errors.New("http: server closed")
)

type atomicBool int32

func (b *atomicBool) isSet() bool {
	return atomic.LoadInt32((*int32)(b)) != 0
}

func (b *atomicBool) set() {
	atomic.StoreInt32((*int32)(b), 1)
}

// Handler represents a handler of HTTP requests.
type Handler interface {
	ServeHTTP(context.Context, *Request) *Response
}

// HandlerFunc is an adapter to allow the use of functions as HTTP handlers.
type HandlerFunc func(context.Context, *Request) *Response

// ServeHTTP serves an HTTP connection.
func (h HandlerFunc) ServeHTTP(ctx context.Context, r *Request) *Response {
	return h(ctx, r)
}

// Conn represents a network connection.
type Conn interface {
	io.ReadWriteCloser

	// RemoteAddr returns the remote network address.
	RemoteAddr() string
}

var (
	bufioReaderPool sync.Pool
	bufioWriterPool sync.Pool
)

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}

	return bufio.NewReader(r)
}

func putBufioReader(r *bufio.Reader) {
	r.Reset(nil)
	bufioReaderPool.Put(r)
}

func newBufioWriter(w io.Writer) *bufio.Writer {
	if v := bufioWriterPool.Get(); v != nil {
		br := v.(*bufio.Writer)
		br.Reset(w)
		return br
	}

	return bufio.NewWriter(w)
}

func putBufioWriter(w *bufio.Writer) {
	w.Reset(nil)
	bufioWriterPool.Put(w)
}

type connState int

const (
	stateNew connState = iota
	stateActive
	stateIdle
	stateClosed
)

type conn struct {
	server *Server

	handler Handler

	rwc      net.Conn
	tlsState *tls.ConnectionState

	bufr *bufio.Reader
	bufw *bufio.Writer

	state uint32
}

func (c *conn) setState(state connState) {
	atomic.StoreUint32(&c.state, uint32(state))

	srv := c.server
	switch state {
	case stateNew:
		srv.addConn(c)
	case stateClosed:
		srv.removeConn(c)
	}
}

func (c *conn) getState() connState {
	return connState(atomic.LoadUint32(&c.state))
}

func (c *conn) serve(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.server.logf("http: panic serving %v: %v", c.rwc.RemoteAddr(), err)
		}

		c.setState(stateClosed)
		c.close()
	}()

	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
		if d := c.server.readTimeout; d != 0 {
			_ = c.rwc.SetReadDeadline(time.Now().Add(d))
		}
		if d := c.server.writeTimeout; d != 0 {
			_ = c.rwc.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			c.server.logf("http: tls handshake error %v: %v", c.rwc.RemoteAddr(), err)
			return
		}
		c.tlsState = &tls.ConnectionState{}
		*c.tlsState = tlsConn.ConnectionState()
	}

	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	c.bufr = newBufioReader(c.rwc)
	defer putBufioReader(c.bufr)
	c.bufw = newBufioWriter(c.rwc)
	defer putBufioWriter(c.bufw)

	c.handler = c.server.handler

	for {
		if d := c.server.readTimeout; d != 0 {
			_ = c.rwc.SetReadDeadline(time.Now().Add(d))
		}

		c.setState(stateActive)

		req, err := c.readRequest(ctx)
		if err != nil {
			c.server.logf("http: error reading request %v: %v", c.rwc.RemoteAddr(), err)

			const errResp string = "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n400 Bad Request"

			c.rwc.Write([]byte(errResp))
			c.rwc.Close()
			return
		}

		if d := c.server.writeTimeout; d != 0 {
			_ = c.rwc.SetWriteDeadline(time.Now().Add(d))
		}

		resp := c.handler.ServeHTTP(ctx, req)

		if err := resp.Write(c.bufw); err != nil {
			c.server.logf("http: error writing response  %v: %v", c.rwc.RemoteAddr(), err)
			return
		}
		if err := c.bufw.Flush(); err != nil {
			c.server.logf("http: error writing response  %v: %v", c.rwc.RemoteAddr(), err)
			return
		}

		if resp == nil || resp.Close {
			return
		}

		c.setState(stateIdle)

		if d := c.server.idleTimeout; d != 0 {
			_ = c.rwc.SetReadDeadline(time.Now().Add(d))
			if _, err := c.bufr.Peek(1); err != nil {
				return
			}
		}
		_ = c.rwc.SetReadDeadline(time.Time{})
	}
}

func (c *conn) readRequest(ctx context.Context) (*Request, error) {
	req, err := readRequest(c.bufr)
	if err != nil {
		return nil, err
	}

	req.ctx = ctx
	req.RemoteAddr = c.rwc.RemoteAddr().String()

	return req, nil
}

func (c *conn) close() {
	if c.bufr != nil {
		putBufioReader(c.bufr)
		c.bufr = nil
	}
	_ = c.rwc.Close()
}

// Opts configure the server.
type Opts struct {
	// ReadTimeout is the maximum duration to start reading a request.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration to start writing a response.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum duration to wait for the next request.
	// If IdleTimeout is zero, the value of ReadTimeout is used.
	IdleTimeout time.Duration

	// ErrorLog is an optional function that errors are written to. If
	// nil, errors are written to stdout. Calls to the function may be
	// concurrent.
	ErrorLog func(string)
}

// Server is a TCP server.
type Server struct {
	handler      Handler
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	errLog       func(string)

	inShutdown atomicBool

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	activeConn map[*conn]struct{}
}

// NewServer returns a Server configured with the given parameters.
func NewServer(h Handler, opts Opts) (*Server, error) {
	if h == nil {
		return nil, errors.New("http: Handler cannot be nil")
	}

	idleTimeout := opts.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = opts.ReadTimeout
	}

	return &Server{
		handler:      h,
		readTimeout:  opts.ReadTimeout,
		writeTimeout: opts.WriteTimeout,
		idleTimeout:  idleTimeout,
		errLog:       opts.ErrorLog,
		listeners:    map[*net.Listener]struct{}{},
		activeConn:   map[*conn]struct{}{},
	}, nil
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.errLog != nil {
		s.errLog(fmt.Sprintf(format, args...))
		return
	}

	fmt.Printf(format, args...)
}

func (s *Server) addListener(ln *net.Listener) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inShutdown.isSet() {
		return false
	}

	s.listeners[ln] = struct{}{}
	return true
}

func (s *Server) removeListener(ln *net.Listener) {
	s.mu.Lock()

	delete(s.listeners, ln)

	s.mu.Unlock()
}

// caller must hold s.mu
func (s *Server) closeListeners() error {
	var err error
	for ln := range s.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
		delete(s.listeners, ln)
	}
	return err
}

func (s *Server) addConn(c *conn) {
	s.mu.Lock()

	s.activeConn[c] = struct{}{}

	s.mu.Unlock()
}

func (s *Server) removeConn(c *conn) {
	s.mu.Lock()

	delete(s.activeConn, c)

	s.mu.Unlock()
}

func (s *Server) closeIdleConns() bool {
	s.mu.Lock()

	quiescent := true
	for c := range s.activeConn {
		state := c.getState()
		if state != stateIdle {
			quiescent = false
			continue
		}

		_ = c.rwc.Close()
		delete(s.activeConn, c)
	}

	s.mu.Unlock()

	return quiescent
}

type onceCloseListener struct {
	net.Listener
	once sync.Once
	err  error
}

func (l *onceCloseListener) close() {
	l.err = l.Listener.Close()
}

func (l *onceCloseListener) Close() error {
	l.once.Do(l.close)
	return l.err
}

// Serve serves connections on the given listener.
func (s *Server) Serve(ln net.Listener) error {
	ln = &onceCloseListener{Listener: ln}
	defer ln.Close()

	if !s.addListener(&ln) {
		return ErrServerClosed
	}
	defer s.removeListener(&ln)

	for {
		rwc, err := ln.Accept()
		if err != nil {
			if s.inShutdown.isSet() {
				return ErrServerClosed
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				s.logf("http: Accept error: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			return err
		}

		c := &conn{
			server: s,
			rwc:    rwc,
		}
		c.setState(stateNew)
		go c.serve(context.Background())
	}
}

// ListenAndServe listens to the given address and calls Serve.
func (s *Server) ListenAndServe(addr string) error {
	if s.inShutdown.isSet() {
		return ErrServerClosed
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

// ListenAndServeTLS listens to the given address with TLS and calls Serve.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if s.inShutdown.isSet() {
		return ErrServerClosed
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	ln, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

var shutdownPollInterval = 100 * time.Millisecond

// Shutdown gracefully shuts the server down, waiting from
// connections to be idle before closing them.
func (s *Server) Shutdown(ctx context.Context) error {
	s.inShutdown.set()

	s.mu.Lock()
	err := s.closeListeners()
	s.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if s.closeIdleConns() {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Close closes the server, forcefully closing all connections.
func (s *Server) Close() error {
	s.inShutdown.set()

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.closeListeners()
	for c := range s.activeConn {
		_ = c.rwc.Close()
		delete(s.activeConn, c)
	}
	return err
}
