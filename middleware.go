package proxy

import (
	"fmt"
	"time"

	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/middleware"
)

func createMiddleware(cfg []map[string]interface{}, h http.Handler) (http.Handler, error) {
	var err error

	for _, c := range cfg {
		typ := c["type"]
		switch typ {
		case "cache":
			h, err = createCacheMiddleware(c, h)
			if err != nil {
				return nil, err
			}

		case "location":
			h, err = createLocationMiddleware(c, h)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("proxy: unknown middleware %s", typ)
		}

	}

	return h, nil
}

func createCacheMiddleware(cfg map[string]interface{}, h http.Handler) (http.Handler, error) {
	expiry, err := parseDuration(cfg, "expiry")
	if err != nil {
		return nil, err
	}
	purge, err := parseDuration(cfg, "purge")
	if err != nil {
		return nil, err
	}
	ignore, err := parseBool(cfg, "ignoreHeaders")
	if err != nil {
		return nil, err
	}

	return middleware.NewCache(h, middleware.CacheOpts{
		Expiry:        expiry,
		Purge:         purge,
		IgnoreHeaders: ignore,
	}), nil
}

func createLocationMiddleware(cfg map[string]interface{}, h http.Handler) (http.Handler, error) {
	path, ok := cfg["path"].(string)
	if !ok {
		return nil, fmt.Errorf("proxy: invalid location path")
	}

	return middleware.NewLocation(h, path), nil
}

func parseBool(cfg map[string]interface{}, k string) (bool, error) {
	v, ok := cfg[k]
	if !ok {
		return false, nil
	}

	switch val := v.(type) {
	case string:
		return val == "true", nil

	case bool:
		return val, nil

	default:
		return false, fmt.Errorf("proxy: invalid boolean %s", k)
	}
}

func parseDuration(cfg map[string]interface{}, k string) (time.Duration, error) {
	v, ok := cfg[k]
	if !ok {
		return 0, nil
	}

	switch val := v.(type) {
	case string:
		return time.ParseDuration(val)

	case time.Duration:
		return val, nil

	default:
		return 0, fmt.Errorf("proxy: invalid duration %s", k)
	}
}
