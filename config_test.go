package proxy_test

import (
	"os"
	"testing"
	"time"

	"github.com/nrwiersma/proxy"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	f, err := os.Open("./testdata/config.yml")
	if err != nil {
		t.Fatal("error reading config file", err)
	}
	defer f.Close()

	got, err := proxy.ParseConfig(f)

	if assert.NoError(t, err) {
		want := &proxy.Config{
			Server: proxy.ServiceOpts{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  time.Second,
				AccessLog:    true,
			},
			Entrypoints: map[string]proxy.Entrypoint{
				"http": {
					Address: ":8080",
				},
				"https": {
					Address: ":8443",
					Certificate: &proxy.Certificate{
						CertFile: "./testdata/cert.pem",
						KeyFile:  "./testdata/key.pem",
					},
				},
			},
			Backends: map[string]proxy.Backend{
				"test-server": {
					Servers: []string{"http://127.0.0.1:9080", "http://127.0.0.1:9081"},
					Timeout: time.Second,
				},
			},
			Routes: map[string]proxy.Route{
				"test-route": {
					Pattern: "test1.dev/test",
					Backend: "test-server",
					Middleware: []map[string]interface{}{
						{
							"type":          "cache",
							"expiry":        "10s",
							"purge":         "1m",
							"ignoreHeaders": true,
						},
					},
				},
			},
		}
		assert.Equal(t, want, got)
	}
}
