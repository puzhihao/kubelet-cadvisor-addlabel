package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// MetricsServer exposes the aggregated metrics payload over HTTP.
type MetricsServer struct {
	mu     sync.RWMutex
	data   string
	server *http.Server
}

// NewMetricsServer creates a metrics HTTP server bound to the provided port.
func NewMetricsServer(port int) *MetricsServer {
	mux := http.NewServeMux()
	srv := &MetricsServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	mux.HandleFunc("/metrics", srv.handleMetrics)
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/", srv.handleInfo)

	return srv
}

// Update replaces the metrics payload served under /metrics.
func (s *MetricsServer) Update(data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = data
	klog.InfoS("metrics payload updated", "bytes", len(data))
}

// Run starts listening for HTTP requests and blocks until the context ends.
func (s *MetricsServer) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		klog.InfoS("metrics HTTP server listening", "address", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown metrics server: %w", err)
		}
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *MetricsServer) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	if s.data == "" {
		klog.V(2).Info("metrics payload unavailable")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("# metrics temporarily unavailable\n"))
		return
	}

	klog.V(4).InfoS("serving metrics payload", "bytes", len(s.data))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.data))
}

func (s *MetricsServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *MetricsServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
	}

	_, _ = w.Write([]byte(
		"Available endpoints:\n" +
			"  GET /metrics - aggregated cadvisor metrics\n" +
			"  GET /health  - server liveness probe\n",
	))
}
