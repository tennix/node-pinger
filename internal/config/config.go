package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMetricsAddr   = ":9095"
	defaultProbeInterval = 10 * time.Second
	defaultProbeTimeout  = 500 * time.Millisecond
	defaultJitterFactor  = 0.2
)

type Config struct {
	LocalNodeName       string
	KubeconfigPath      string
	MetricsAddr         string
	ProbeInterval       time.Duration
	ProbeTimeout        time.Duration
	ProbeJitterFactor   float64
	ExcludeNotReady     bool
	ExcludeControlPlane bool
}

func Parse() (Config, error) {
	return ParseEnv(os.LookupEnv)
}

func ParseEnv(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		MetricsAddr:       defaultMetricsAddr,
		ProbeInterval:     defaultProbeInterval,
		ProbeTimeout:      defaultProbeTimeout,
		ProbeJitterFactor: defaultJitterFactor,
		KubeconfigPath:    defaultKubeconfigPath(),
	}

	localNodeName, err := requiredString(lookup, "NODE_NAME")
	if err != nil {
		return Config{}, err
	}
	cfg.LocalNodeName = localNodeName

	if value, ok := lookup("KUBECONFIG"); ok && strings.TrimSpace(value) != "" {
		cfg.KubeconfigPath = strings.TrimSpace(value)
	}

	if value, ok := lookup("METRICS_ADDR"); ok && strings.TrimSpace(value) != "" {
		cfg.MetricsAddr = strings.TrimSpace(value)
	}

	if cfg.ProbeInterval, err = durationEnv(lookup, "PROBE_INTERVAL", cfg.ProbeInterval); err != nil {
		return Config{}, err
	}
	if cfg.ProbeTimeout, err = durationEnv(lookup, "PROBE_TIMEOUT", cfg.ProbeTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ProbeJitterFactor, err = floatEnv(lookup, "PROBE_JITTER_FACTOR", cfg.ProbeJitterFactor); err != nil {
		return Config{}, err
	}
	if cfg.ProbeJitterFactor < 0 || cfg.ProbeJitterFactor > 1 {
		return Config{}, fmt.Errorf("PROBE_JITTER_FACTOR must be between 0 and 1")
	}
	if cfg.ExcludeNotReady, err = boolEnv(lookup, "EXCLUDE_NOT_READY", false); err != nil {
		return Config{}, err
	}
	if cfg.ExcludeControlPlane, err = boolEnv(lookup, "EXCLUDE_CONTROL_PLANE", false); err != nil {
		return Config{}, err
	}
	if cfg.ProbeTimeout >= cfg.ProbeInterval {
		return Config{}, fmt.Errorf("PROBE_TIMEOUT must be less than PROBE_INTERVAL")
	}
	maxJitter := time.Duration(float64(cfg.ProbeInterval) * cfg.ProbeJitterFactor)
	if cfg.ProbeTimeout+maxJitter >= cfg.ProbeInterval {
		return Config{}, fmt.Errorf("PROBE_TIMEOUT plus max jitter must be less than PROBE_INTERVAL")
	}

	return cfg, nil
}

func defaultKubeconfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

func requiredString(lookup func(string) (string, bool), key string) (string, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return strings.TrimSpace(value), nil
}

func durationEnv(lookup func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return parsed, nil
}

func floatEnv(lookup func(string) (string, bool), key string, fallback float64) (float64, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func boolEnv(lookup func(string) (string, bool), key string, fallback bool) (bool, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
