package proxy

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/hamba/pkg/log"
	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/http/proxy"
	"github.com/nrwiersma/proxy/http/router"
	"github.com/nrwiersma/proxy/middleware"
)

// ServiceOpts are options to configure the service.
type ServiceOpts struct {
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
	AccessLog    bool          `yaml:"accessLog"`
}

// Service is a reverse proxy service.
type Service struct {
	mu     sync.Mutex
	bkends map[string]http.Handler
	rtr    *router.Router
	srv    *http.Server
	log    log.Logger
}

// NewServiceFromConfig returns a reverse proxy service with the given configuration.
func NewServiceFromConfig(labl log.Loggable, c *Config) (*Service, error) {
	svc, err := NewService(labl, c.Server)
	if err != nil {
		return nil, err
	}

	// Backends
	for name, bkend := range c.Backends {
		if err := svc.AddBackend(name, bkend); err != nil {
			return nil, err
		}
	}

	// Route
	for name, route := range c.Routes {
		if err := svc.AddRoute(name, route); err != nil {
			return nil, err
		}
	}

	// Entrypoint
	for name, ep := range c.Entrypoints {
		if err := svc.AddEndpoint(name, ep); err != nil {
			return nil, err
		}
	}

	return svc, nil
}

// NewService returns a configured reverse proxy service.
func NewService(labl log.Loggable, opts ServiceOpts) (*Service, error) {
	svc := &Service{
		bkends: map[string]http.Handler{},
		rtr:    &router.Router{},
		log:    labl.Logger(),
	}

	var h http.Handler = svc.rtr
	if opts.AccessLog {
		h = middleware.NewLogger(h, svc.log)
	}

	srv, err := http.NewServer(h, http.Opts{
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	})
	if err != nil {
		return nil, err
	}
	svc.srv = srv

	return svc, nil
}

// Backend represents a service backend.
type Backend struct {
	Servers []string      `yaml:"servers"`
	Timeout time.Duration `yaml:"timeout"`
}

// AddBackend adds a backend to the service.
func (s *Service) AddBackend(name string, bkend Backend) error {
	if len(bkend.Servers) == 0 {
		return fmt.Errorf("proxy: backend %s must have at least 1 backend", name)
	}

	srvs := make([]http.Handler, len(bkend.Servers))
	for i, srv := range bkend.Servers {
		u, err := url.Parse(srv)
		if err != nil {
			return fmt.Errorf("proxy: invalid server '%s' in backend %s", srv, name)
		}

		var h http.Handler
		opts := proxy.Opts{Timeout: bkend.Timeout}
		switch u.Scheme {
		case "http", "":
			h, err = proxy.New(u.Host, opts)

		case "https":
			h, err = proxy.NewTLS(u.Host, "", "", opts)

		default:
			return fmt.Errorf("proxy: unknown scheme '%s' in backend %s", u.Scheme, name)
		}
		if err != nil {
			return fmt.Errorf("proxy: invalid server '%s' in backend %s", srv, name)
		}

		srvs[i] = h
	}

	s.mu.Lock()
	s.bkends[name] = proxy.NewRRLoadBalancer(srvs)
	s.mu.Unlock()

	return nil
}

// Route represents a service route.
type Route struct {
	Pattern    string                   `yaml:"pattern"`
	Backend    string                   `yaml:"backend"`
	Middleware []map[string]interface{} `yaml:"middleware"`
}

// AddRoute adds a route to the service.
func (s *Service) AddRoute(name string, route Route) error {
	backend, ok := s.bkends[route.Backend]
	if !ok {
		return fmt.Errorf("proxy: unknown backend %s in route %s", route.Backend, name)
	}

	h, err := createMiddleware(route.Middleware, backend)
	if err != nil {
		return err
	}

	s.rtr.AddHandler(route.Pattern, h)

	return nil
}

// Endpoint represents a service endpoint.
type Entrypoint struct {
	Address     string       `yaml:"address"`
	Certificate *Certificate `yaml:"tls"`
}

func (e *Entrypoint) isTLS() bool {
	return e.Certificate != nil &&
		e.Certificate.CertFile != "" &&
		e.Certificate.KeyFile != ""
}

// Certificate represents a service certificate.
type Certificate struct {
	CertFile string `yaml:"cert"`
	KeyFile  string `yaml:"key"`
}

// AddEndpoint adds an endpoint to the service.
func (s *Service) AddEndpoint(name string, ep Entrypoint) error {
	if ep.isTLS() {
		go func() {
			s.log.Info(fmt.Sprintf("Starting tls server on address %s", ep.Address))
			err := s.srv.ListenAndServeTLS(ep.Address, ep.Certificate.CertFile, ep.Certificate.KeyFile)
			if err != nil {
				s.log.Error("service: server error", "error", err)
			}
		}()
		return nil
	}

	go func() {
		s.log.Info(fmt.Sprintf("Starting server on address %s", ep.Address))
		err := s.srv.ListenAndServe(ep.Address)
		if err != nil {
			s.log.Error("service: server error", "error", err)
		}
	}()

	return nil
}

// Shutdown attempts to shut the service down in the given timeout.
func (s *Service) Shutdown(d time.Duration) error {
	ctx := context.Background()
	var cancelFn context.CancelFunc = func() {}
	if d > 0 {
		ctx, cancelFn = context.WithTimeout(context.Background(), d)
		defer cancelFn()
	}
	return s.srv.Shutdown(ctx)
}

// Close will forcefully close the service.
func (s *Service) Close() error {
	return s.srv.Close()
}
