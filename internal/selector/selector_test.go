package selector

import (
	"reflect"
	"testing"

	"github.com/tennix/node-pinger/internal/model"
)

func TestFilter(t *testing.T) {
	t.Parallel()

	nodes := []model.Node{
		{Name: "node-b", InternalIP: "10.0.0.2", Ready: true},
		{Name: "node-a", InternalIP: "10.0.0.1", Ready: true},
		{Name: "node-self", InternalIP: "10.0.0.9", Ready: true},
		{Name: "node-no-ip", Ready: true},
		{Name: "node-not-ready", InternalIP: "10.0.0.5", Ready: false},
		{Name: "node-cp", InternalIP: "10.0.0.6", Ready: true, ControlPlane: true},
	}

	got := Filter(nodes, Options{
		LocalNodeName:       "node-self",
		ExcludeNotReady:     true,
		ExcludeControlPlane: true,
	})

	want := []model.Node{
		{Name: "node-a", InternalIP: "10.0.0.1", Ready: true},
		{Name: "node-b", InternalIP: "10.0.0.2", Ready: true},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filter() = %#v, want %#v", got, want)
	}
}

func TestFindByName(t *testing.T) {
	t.Parallel()

	node, ok := FindByName([]model.Node{{Name: "node-a"}}, "node-a")
	if !ok {
		t.Fatalf("expected node to be found")
	}
	if node.Name != "node-a" {
		t.Fatalf("node.Name = %q", node.Name)
	}
}
