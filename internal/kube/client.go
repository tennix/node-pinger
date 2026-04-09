package kube

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func BuildRestConfig(kubeconfigPath string) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	if kubeconfigPath == "" {
		return nil, fmt.Errorf("in-cluster config unavailable and kubeconfig path is empty: %w", err)
	}
	if _, statErr := os.Stat(kubeconfigPath); statErr != nil {
		return nil, fmt.Errorf("stat kubeconfig %q: %w", kubeconfigPath, statErr)
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

func NewClientset(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}
