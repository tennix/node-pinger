package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/tennix/node-pinger/internal/config"
	"github.com/tennix/node-pinger/internal/httpserver"
	"github.com/tennix/node-pinger/internal/identity"
	"github.com/tennix/node-pinger/internal/kube"
	"github.com/tennix/node-pinger/internal/metrics"
	"github.com/tennix/node-pinger/internal/probe"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Parse()
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}

	localIdentity, err := identity.FromEnv(cfg.LocalNodeName)
	if err != nil {
		log.Fatalf("resolve local node identity: %v", err)
	}

	clientConfig, err := kube.BuildRestConfig(cfg.KubeconfigPath)
	if err != nil {
		log.Fatalf("build kubernetes config: %v", err)
	}

	clientset, err := kube.NewClientset(clientConfig)
	if err != nil {
		log.Fatalf("create kubernetes client: %v", err)
	}

	discovery, err := kube.NewNodeDiscovery(clientset)
	if err != nil {
		log.Fatalf("create node discovery: %v", err)
	}

	if err := discovery.Start(ctx); err != nil {
		log.Fatalf("start node discovery: %v", err)
	}

	registry := metrics.New(localIdentity.Name)
	pinger, err := probe.NewPinger()
	if err != nil {
		log.Fatalf("create icmp pinger: %v", err)
	}
	defer pinger.Close()

	agent := probe.NewAgent(cfg, localIdentity, discovery, pinger, registry)

	server, err := httpserver.New(cfg.MetricsAddr, registry.Handler())
	if err != nil {
		log.Fatalf("create metrics server: %v", err)
	}
	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("metrics server stopped: %v", err)
		}
	}()

	log.Printf("starting node-pinger on node %q", localIdentity.Name)
	if err := agent.Run(ctx); err != nil {
		log.Fatalf("run agent: %v", err)
	}
	log.Printf("node-pinger stopped")
}
