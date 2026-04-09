package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/tennix/node-pinger/internal/model"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	localNode   string
	prom        *prometheus.Registry
	rtt         *prometheus.GaugeVec
	probesTotal *prometheus.CounterVec
	lastSuccess *prometheus.GaugeVec
	mu          sync.Mutex
	tracked     map[string]string
	srcZone     string
}

func New(localNode string) *Registry {
	r := &Registry{
		localNode: localNode,
		prom:      prometheus.NewRegistry(),
		rtt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "node_icmp_rtt_ms",
			Help: "Latest successful node-to-node ICMP RTT in milliseconds.",
		}, []string{"src_node", "dst_node", "src_zone", "dst_zone"}),
		probesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "node_icmp_probes_total",
			Help: "Total ICMP probes by result.",
		}, []string{"src_node", "dst_node", "result"}),
		lastSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "node_icmp_last_success_unixtime",
			Help: "Unix timestamp of the most recent successful probe.",
		}, []string{"src_node", "dst_node"}),
		tracked: make(map[string]string),
	}
	r.prom.MustRegister(r.rtt, r.probesTotal, r.lastSuccess)
	return r
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.prom, promhttp.HandlerOpts{})
}

func (r *Registry) RecordSuccess(srcZone string, peer model.Node, rtt time.Duration, now time.Time) {
	r.trackPeer(srcZone, peer)
	r.rtt.WithLabelValues(r.localNode, peer.Name, srcZone, peer.Zone).Set(float64(rtt) / float64(time.Millisecond))
	r.probesTotal.WithLabelValues(r.localNode, peer.Name, "success").Inc()
	r.lastSuccess.WithLabelValues(r.localNode, peer.Name).Set(float64(now.Unix()))
}

func (r *Registry) RecordTimeout(peer model.Node) {
	r.trackPeer("", peer)
	r.probesTotal.WithLabelValues(r.localNode, peer.Name, "timeout").Inc()
}

func (r *Registry) RecordError(peer model.Node) {
	r.trackPeer("", peer)
	r.probesTotal.WithLabelValues(r.localNode, peer.Name, "error").Inc()
}

func (r *Registry) Reconcile(srcZone string, peers []model.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.srcZone != "" && r.srcZone != srcZone {
		for peerName, dstZone := range r.tracked {
			r.deleteLocked(peerName, dstZone, r.srcZone)
		}
		r.tracked = make(map[string]string, len(peers))
	}
	r.srcZone = srcZone

	desired := make(map[string]string, len(peers))
	for _, peer := range peers {
		desired[peer.Name] = peer.Zone
	}

	for peerName, dstZone := range r.tracked {
		wantedZone, ok := desired[peerName]
		if ok && wantedZone == dstZone {
			continue
		}
		r.deleteLocked(peerName, dstZone, srcZone)
		delete(r.tracked, peerName)
	}

	for peerName, dstZone := range desired {
		r.tracked[peerName] = dstZone
	}
}

func (r *Registry) trackPeer(srcZone string, peer model.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if srcZone != "" {
		r.srcZone = srcZone
	}
	r.tracked[peer.Name] = peer.Zone
}

func (r *Registry) deleteLocked(peerName, dstZone, srcZone string) {
	r.rtt.DeleteLabelValues(r.localNode, peerName, srcZone, dstZone)
	r.lastSuccess.DeleteLabelValues(r.localNode, peerName)
	r.probesTotal.DeleteLabelValues(r.localNode, peerName, "success")
	r.probesTotal.DeleteLabelValues(r.localNode, peerName, "timeout")
	r.probesTotal.DeleteLabelValues(r.localNode, peerName, "error")
}
