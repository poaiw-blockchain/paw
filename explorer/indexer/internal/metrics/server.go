package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server exposes Prometheus metrics over HTTP.
type Server struct {
	srv *http.Server
}

// NewServer builds a metrics server on the provided port.
func NewServer(port int) *Server {
	if port == 0 {
		return nil
	}
	return &Server{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: promhttp.Handler(),
		},
	}
}

// Start serves metrics until shutdown; returns nil when disabled.
func (s *Server) Start() error {
	if s == nil {
		return nil
	}
	return s.srv.ListenAndServe()
}

// Stop gracefully shuts down the metrics server; no-op when disabled.
func (s *Server) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}
