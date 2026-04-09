package kube

import (
	"context"
	"fmt"

	"github.com/tennix/node-pinger/internal/model"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type NodeDiscovery struct {
	informer cache.SharedIndexInformer
	lister   corev1listers.NodeLister
	started  bool
}

func NewNodeDiscovery(client kubernetes.Interface) (*NodeDiscovery, error) {
	if client == nil {
		return nil, fmt.Errorf("nil kubernetes client")
	}
	factory := informers.NewSharedInformerFactory(client, 0)
	nodes := factory.Core().V1().Nodes()
	return &NodeDiscovery{
		informer: nodes.Informer(),
		lister:   nodes.Lister(),
	}, nil
}

func (d *NodeDiscovery) Start(ctx context.Context) error {
	if d.started {
		return nil
	}
	d.started = true
	go d.informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), d.informer.HasSynced) {
		return fmt.Errorf("timed out waiting for node informer cache sync")
	}
	return nil
}

func (d *NodeDiscovery) ListNodes() ([]model.Node, error) {
	objects, err := d.lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	nodes := make([]model.Node, 0, len(objects))
	for _, object := range objects {
		node, ok := model.FromKubeNode(object.DeepCopy())
		if !ok {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (d *NodeDiscovery) GetNode(ctx context.Context, client kubernetes.Interface, name string) (model.Node, error) {
	object, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return model.Node{}, err
	}
	node, _ := model.FromKubeNode(object)
	return node, nil
}
