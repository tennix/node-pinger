package selector

import (
	"sort"

	"github.com/tennix/node-pinger/internal/model"
)

type Options struct {
	LocalNodeName       string
	ExcludeNotReady     bool
	ExcludeControlPlane bool
}

func Filter(nodes []model.Node, opts Options) []model.Node {
	selected := make([]model.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Name == opts.LocalNodeName {
			continue
		}
		if node.InternalIP == "" {
			continue
		}
		if opts.ExcludeNotReady && !node.Ready {
			continue
		}
		if opts.ExcludeControlPlane && node.ControlPlane {
			continue
		}
		selected = append(selected, node)
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Name < selected[j].Name
	})

	return selected
}

func FindByName(nodes []model.Node, name string) (model.Node, bool) {
	for _, node := range nodes {
		if node.Name == name {
			return node, true
		}
	}
	return model.Node{}, false
}
