package handler

import (
	"log/slog"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

// RatelProxy reverse-proxies requests under BasePath+"/dgraph" to Ratel's
// in-cluster address (cfg.RatelInternalURL), stripping the "/dgraph" prefix
// first — the Go-code equivalent of the mesh's `rewrite: uri: /`. This lets a
// direct pod port-forward (which bypasses Istio entirely) reach Ratel through
// the same host:port as orbital itself, via a relative navbar link.
type RatelProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
	logger *slog.Logger
}

// NewRatelProxy builds a RatelProxy forwarding to internalURL.
func NewRatelProxy(internalURL string, logger *slog.Logger) (*RatelProxy, error) {
	target, err := url.Parse(internalURL)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorLog = slog.NewLogLogger(logger.Handler(), slog.LevelError)
	return &RatelProxy{target: target, proxy: proxy, logger: logger}, nil
}

// Handle strips the "/dgraph" prefix from the request path and forwards
// everything else (method, headers, body) to Ratel as-is.
func (h *RatelProxy) Handle(c echo.Context) error {
	req := c.Request()
	path := req.URL.Path
	if i := strings.Index(path, "/dgraph"); i >= 0 {
		req.URL.Path = path[i+len("/dgraph"):]
	}
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}
	h.proxy.ServeHTTP(c.Response(), req)
	return nil
}
