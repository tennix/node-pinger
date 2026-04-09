# AGENTS.md

## Quick facts
- This repo builds a single Go binary from `./cmd/node-pinger`; there is no monorepo/task-runner layer or Makefile in the repo root.
- Primary verification is `go test ./...` and then `go build ./cmd/node-pinger`.
- Local runtime needs `NODE_NAME` and usually `KUBECONFIG`, and actual probing requires raw ICMP privileges (`NET_RAW` or equivalent).
- `docs/design.md` is useful for intent, but current behavior should be taken from `cmd/node-pinger/main.go`, `internal/*`, and the deploy/chart manifests.

## Repository shape
- `cmd/node-pinger/main.go` is the authoritative startup path: config parsing → local identity → Kubernetes client/discovery → metrics registry → ICMP pinger → `/metrics` HTTP server → probe agent loop.
- `internal/config` is where env validation lives. Do not rely on README defaults alone; code enforces that `NODE_NAME` is required, `PROBE_JITTER_FACTOR` is `0..1`, `PROBE_TIMEOUT < PROBE_INTERVAL`, and `PROBE_TIMEOUT + max jitter < PROBE_INTERVAL`.
- `internal/kube` owns cluster access and node discovery, `internal/selector` owns peer filtering/sorting, `internal/probe` owns scheduling and ICMP probes, `internal/metrics` owns Prometheus metric registration/reconciliation, and `internal/httpserver` only serves `/metrics`.
- `internal/probe/agent.go` runs one probe round immediately on startup before the interval ticker begins; peer probes then fan out concurrently with per-peer jitter delays.
- `internal/selector` always excludes the local node and nodes without `InternalIP`, optionally excludes not-ready/control-plane nodes, and sorts peers by node name.

## Commands agents should actually use
- Run all Go tests: `go test ./...`
- Build the shipped binary: `go build ./cmd/node-pinger`
- Run locally against a kubeconfig: `NODE_NAME=<node-name> KUBECONFIG=$HOME/.kube/config go run ./cmd/node-pinger`
- Build the container image: `docker build -t ghcr.io/tennix/node-pinger:latest .`
- Apply raw manifests: `kubectl apply -f deploy/rbac.yaml && kubectl apply -f deploy/daemonset.yaml && kubectl apply -f deploy/service.yaml`
- Install Helm chart: `helm install node-pinger ./charts/node-pinger --namespace kube-system --create-namespace`

## Deployment invariants
- `deploy/daemonset.yaml` and `charts/node-pinger/` are two faces of the same deployment. If you change runtime env vars, ports, security context, or networking assumptions in one, update the other.
- The deployment model depends on `hostNetwork: true`, `dnsPolicy: ClusterFirstWithHostNet`, and `NET_RAW`; do not remove those casually or local/node-to-node ICMP behavior changes.
- The Service is intentionally headless (`clusterIP: None`) so scrapers can see every DaemonSet pod.
- RBAC is intentionally narrow: only `get`, `list`, and `watch` on `nodes`.
- The Helm chart exposes extra deployment knobs that raw manifests do not, including `extraEnv`, optional service creation, and `service.headless`; check `charts/node-pinger/templates/*` before changing chart behavior.

## Testing and edit strategy
- Test coverage is narrow today: only `internal/config`, `internal/probe/scheduler`, and `internal/selector` have `_test.go` files. If you change behavior elsewhere, at minimum run `go test ./...` and consider adding focused tests near the changed package.
- For config or scheduling changes, read the code in `internal/config/config.go` and `internal/probe/*` before editing docs or chart values; the executable validation is stricter than the prose.
- For deploy/chart changes, verify against both `deploy/*.yaml` and `charts/node-pinger/{values.yaml,README.md,templates/*}` instead of assuming one is generated from the other.

## Known gaps
- GitHub Actions CI/CD does exist: `.github/workflows/ci.yaml` runs `go test ./...`, `go build ./cmd/node-pinger`, and a Docker build validation on pushes/PRs to `main`, and `.github/workflows/cd.yaml` publishes the GHCR image on semver tags or `workflow_dispatch`.
- `README.md` still lists Helm lint/render checks as TODO work, so do not assume the chart is validated by CI today.
