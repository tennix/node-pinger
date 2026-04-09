package config

import (
	"testing"
	"time"
)

func TestParseEnvDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := ParseEnv(func(key string) (string, bool) {
		if key == "NODE_NAME" {
			return "node-a", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}

	if cfg.LocalNodeName != "node-a" {
		t.Fatalf("LocalNodeName = %q, want node-a", cfg.LocalNodeName)
	}
	if cfg.MetricsAddr != defaultMetricsAddr {
		t.Fatalf("MetricsAddr = %q, want %q", cfg.MetricsAddr, defaultMetricsAddr)
	}
	if cfg.ProbeInterval != defaultProbeInterval {
		t.Fatalf("ProbeInterval = %v, want %v", cfg.ProbeInterval, defaultProbeInterval)
	}
	if cfg.ProbeTimeout != defaultProbeTimeout {
		t.Fatalf("ProbeTimeout = %v, want %v", cfg.ProbeTimeout, defaultProbeTimeout)
	}
	if cfg.ProbeJitterFactor != defaultJitterFactor {
		t.Fatalf("ProbeJitterFactor = %v, want %v", cfg.ProbeJitterFactor, defaultJitterFactor)
	}
}

func TestParseEnvOverrides(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"NODE_NAME":             "node-b",
		"KUBECONFIG":            "/tmp/kubeconfig",
		"METRICS_ADDR":          ":9100",
		"PROBE_INTERVAL":        "15s",
		"PROBE_TIMEOUT":         "250ms",
		"PROBE_JITTER_FACTOR":   "0.3",
		"EXCLUDE_NOT_READY":     "true",
		"EXCLUDE_CONTROL_PLANE": "true",
	}

	cfg, err := ParseEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}

	if cfg.KubeconfigPath != "/tmp/kubeconfig" {
		t.Fatalf("KubeconfigPath = %q", cfg.KubeconfigPath)
	}
	if cfg.MetricsAddr != ":9100" {
		t.Fatalf("MetricsAddr = %q", cfg.MetricsAddr)
	}
	if cfg.ProbeInterval != 15*time.Second {
		t.Fatalf("ProbeInterval = %v", cfg.ProbeInterval)
	}
	if cfg.ProbeTimeout != 250*time.Millisecond {
		t.Fatalf("ProbeTimeout = %v", cfg.ProbeTimeout)
	}
	if cfg.ProbeJitterFactor != 0.3 {
		t.Fatalf("ProbeJitterFactor = %v", cfg.ProbeJitterFactor)
	}
	if !cfg.ExcludeNotReady || !cfg.ExcludeControlPlane {
		t.Fatalf("exclude flags = %+v", cfg)
	}
}

func TestParseEnvValidatesRangeAndTimeout(t *testing.T) {
	t.Parallel()

	_, err := ParseEnv(func(key string) (string, bool) {
		switch key {
		case "NODE_NAME":
			return "node-c", true
		case "PROBE_INTERVAL":
			return "5s", true
		case "PROBE_TIMEOUT":
			return "5s", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatalf("expected timeout validation error")
	}

	_, err = ParseEnv(func(key string) (string, bool) {
		switch key {
		case "NODE_NAME":
			return "node-c", true
		case "PROBE_JITTER_FACTOR":
			return "1.5", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatalf("expected jitter validation error")
	}

	_, err = ParseEnv(func(key string) (string, bool) {
		switch key {
		case "NODE_NAME":
			return "node-c", true
		case "PROBE_INTERVAL":
			return "1s", true
		case "PROBE_TIMEOUT":
			return "900ms", true
		case "PROBE_JITTER_FACTOR":
			return "0.2", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatalf("expected interval budget validation error")
	}
}
