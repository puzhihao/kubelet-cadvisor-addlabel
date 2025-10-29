package main

import (
	"context"
	"errors"
	"flag"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/puzhihao/kubelet-cadvisor-addlabel/config"
	"github.com/puzhihao/kubelet-cadvisor-addlabel/internal/app"
	"github.com/puzhihao/kubelet-cadvisor-addlabel/internal/kube"

	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	cfg := config.NewConfig()
	if err := cfg.Validate(); err != nil {
		klog.Fatalf("invalid configuration: %v", err)
	}

	if err := flag.Set("v", strconv.Itoa(cfg.Verbosity())); err != nil {
		klog.Warningf("unable to apply log verbosity: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	factory, err := kube.NewInformerFactory()
	if err != nil {
		klog.Fatalf("create informer factory: %v", err)
	}

	application := app.New(cfg, factory)
	klog.InfoS("application starting")

	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		klog.ErrorS(err, "application terminated")
		return
	}

	klog.InfoS("application stopped")
}
