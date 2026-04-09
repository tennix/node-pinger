package model

import (
	"net"

	corev1 "k8s.io/api/core/v1"
)

const (
	labelZone               = "topology.kubernetes.io/zone"
	labelControlPlane       = "node-role.kubernetes.io/control-plane"
	labelControlPlaneLegacy = "node-role.kubernetes.io/master"
)

type Node struct {
	Name         string
	InternalIP   string
	Zone         string
	Ready        bool
	ControlPlane bool
}

func FromKubeNode(node *corev1.Node) (Node, bool) {
	if node == nil {
		return Node{}, false
	}

	internalIP := ""
	for _, address := range node.Status.Addresses {
		if address.Type != corev1.NodeInternalIP {
			continue
		}
		if ip := net.ParseIP(address.Address); ip != nil {
			internalIP = ip.String()
			break
		}
	}

	return Node{
		Name:         node.Name,
		InternalIP:   internalIP,
		Zone:         node.Labels[labelZone],
		Ready:        isNodeReady(node),
		ControlPlane: hasControlPlaneRole(node),
	}, true
}

func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func hasControlPlaneRole(node *corev1.Node) bool {
	if node.Labels == nil {
		return false
	}
	if _, ok := node.Labels[labelControlPlane]; ok {
		return true
	}
	_, ok := node.Labels[labelControlPlaneLegacy]
	return ok
}
