package probe

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/tennix/node-pinger/internal/config"
	"github.com/tennix/node-pinger/internal/identity"
	"github.com/tennix/node-pinger/internal/kube"
	"github.com/tennix/node-pinger/internal/metrics"
	"github.com/tennix/node-pinger/internal/selector"
)

type Agent struct {
	cfg       config.Config
	local     identity.LocalNode
	discovery *kube.NodeDiscovery
	pinger    *Pinger
	metrics   *metrics.Registry
	schedule  *Scheduler
}

func NewAgent(cfg config.Config, local identity.LocalNode, discovery *kube.NodeDiscovery, pinger *Pinger, registry *metrics.Registry) *Agent {
	return &Agent{
		cfg:       cfg,
		local:     local,
		discovery: discovery,
		pinger:    pinger,
		metrics:   registry,
		schedule:  NewScheduler(time.Now().UnixNano()),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	if err := a.runRound(ctx); err != nil {
		log.Printf("initial probe round failed: %v", err)
	}

	ticker := time.NewTicker(a.cfg.ProbeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := a.runRound(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("probe round failed: %v", err)
			}
		}
	}
}

func (a *Agent) runRound(ctx context.Context) error {
	nodes, err := a.discovery.ListNodes()
	if err != nil {
		return err
	}

	localNode, _ := selector.FindByName(nodes, a.local.Name)
	peers := selector.Filter(nodes, selector.Options{
		LocalNodeName:       a.local.Name,
		ExcludeNotReady:     a.cfg.ExcludeNotReady,
		ExcludeControlPlane: a.cfg.ExcludeControlPlane,
	})
	a.metrics.Reconcile(localNode.Zone, peers)

	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			delay := a.schedule.Delay(a.cfg.ProbeInterval, a.cfg.ProbeJitterFactor)
			if !sleepWithContext(ctx, delay) {
				return
			}

			rtt, err := a.pinger.Probe(peer, a.cfg.ProbeTimeout)
			now := time.Now()
			switch {
			case err == nil:
				a.metrics.RecordSuccess(localNode.Zone, peer, rtt, now)
			case errors.Is(err, ErrTimeout):
				a.metrics.RecordTimeout(peer)
			default:
				log.Printf("probe to %s (%s) failed: %v", peer.Name, peer.InternalIP, err)
				a.metrics.RecordError(peer)
			}
		}()
	}

	wg.Wait()
	return nil
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
