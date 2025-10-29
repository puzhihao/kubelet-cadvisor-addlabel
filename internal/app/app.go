package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/puzhihao/kubelet-cadvisor-addlabel/config"
	"github.com/puzhihao/kubelet-cadvisor-addlabel/internal/metrics"
	"github.com/puzhihao/kubelet-cadvisor-addlabel/internal/server"

	"k8s.io/client-go/informers"
	"k8s.io/klog/v2"
)

// Application wires together the informer service, metrics collector, and HTTP exporter.
type Application struct {
	cfg           *config.Config
	service       *metrics.Service
	collector     *metrics.Collector
	httpServer    *server.MetricsServer
	fetchInterval time.Duration
}

// New creates a new Application instance.
func New(cfg *config.Config, factory informers.SharedInformerFactory) *Application {
	service := metrics.NewService(factory)
	return &Application{
		cfg:           cfg,
		service:       service,
		collector:     metrics.NewCollector(service, cfg.TokenFile),
		httpServer:    server.NewMetricsServer(cfg.Port),
		fetchInterval: time.Duration(cfg.FetchInterval) * time.Second,
	}
}

// Run starts all components and blocks until the context is cancelled or one component fails.
func (a *Application) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.service.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("metrics service stopped: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.httpServer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("http server stopped: %w", err)
		}
	}()

	if err := a.service.WaitForSync(ctx); err != nil {
		cancel()
		wg.Wait()
		return fmt.Errorf("wait for informer sync: %w", err)
	}

	if err := a.collectAndPublish(ctx, true); err != nil {
		klog.ErrorS(err, "initial metrics collection failed")
	}

	ticker := time.NewTicker(a.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cancel()
			wg.Wait()
			return ctx.Err()
		case err := <-errCh:
			cancel()
			wg.Wait()
			return err
		case <-ticker.C:
			if err := a.collectAndPublish(ctx, false); err != nil {
				klog.ErrorS(err, "metrics collection failed")
			}
		}
	}
}

func (a *Application) collectAndPublish(ctx context.Context, initial bool) error {
	payload, err := a.collector.Collect(ctx, a.cfg.AddLabels, a.cfg.LabelDefaults)
	if err != nil {
		return err
	}

	if payload == "" {
		return fmt.Errorf("collector returned an empty payload")
	}

	a.httpServer.Update(payload)
	if initial {
		klog.InfoS("published initial metrics snapshot", "bytes", len(payload))
	} else {
		klog.V(2).InfoS("metrics snapshot refreshed", "bytes", len(payload))
	}
	return nil
}
