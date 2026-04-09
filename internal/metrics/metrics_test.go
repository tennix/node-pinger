package metrics

import (
	"testing"
	"time"

	"github.com/tennix/node-pinger/internal/model"
)

func TestRecordSuccessExportsGaugeHistogramAndTimestamp(t *testing.T) {
	t.Parallel()

	r := New("node-a")
	peer := model.Node{Name: "node-b", Zone: "zone-b"}
	now := time.Unix(1700000000, 0)

	r.RecordSuccess("zone-a", peer, 3*time.Millisecond, now)

	families, err := r.prom.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	var foundGauge bool
	var foundHistogram bool
	var foundLastSuccess bool

	for _, family := range families {
		switch family.GetName() {
		case "node_icmp_rtt_ms":
			foundGauge = true
			if len(family.Metric) != 1 {
				t.Fatalf("expected one RTT gauge metric, got %d", len(family.Metric))
			}
		case "node_icmp_rtt_seconds":
			foundHistogram = true
			if len(family.Metric) != 1 {
				t.Fatalf("expected one RTT histogram metric, got %d", len(family.Metric))
			}
			labels := family.Metric[0].GetLabel()
			if len(labels) != 4 {
				t.Fatalf("expected histogram to use 4 labels, got %d", len(labels))
			}
			if got := family.Metric[0].GetHistogram().GetSampleCount(); got != 1 {
				t.Fatalf("expected histogram sample count 1, got %d", got)
			}
		case "node_icmp_last_success_unixtime":
			foundLastSuccess = true
			if len(family.Metric) != 1 {
				t.Fatalf("expected one last-success metric, got %d", len(family.Metric))
			}
		}
	}

	if !foundGauge {
		t.Fatal("expected node_icmp_rtt_ms metric family")
	}
	if !foundHistogram {
		t.Fatal("expected node_icmp_rtt_seconds metric family")
	}
	if !foundLastSuccess {
		t.Fatal("expected node_icmp_last_success_unixtime metric family")
	}
}

func TestReconcileDeletesHistogramForRemovedPeer(t *testing.T) {
	t.Parallel()

	r := New("node-a")
	peer := model.Node{Name: "node-b", Zone: "zone-b"}
	r.RecordSuccess("zone-a", peer, 4*time.Millisecond, time.Now())
	r.Reconcile("zone-a", nil)

	families, err := r.prom.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() != "node_icmp_rtt_seconds" {
			continue
		}
		if len(family.Metric) != 0 {
			t.Fatalf("expected histogram metrics to be deleted, got %d", len(family.Metric))
		}
	}
}
