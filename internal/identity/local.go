package identity

import (
	"fmt"
	"strings"
)

type LocalNode struct {
	Name string
}

func FromEnv(nodeName string) (LocalNode, error) {
	if strings.TrimSpace(nodeName) == "" {
		return LocalNode{}, fmt.Errorf("local node name is empty")
	}
	return LocalNode{Name: strings.TrimSpace(nodeName)}, nil
}
