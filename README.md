# node-pinger

`node-pinger` is a Kubernetes-native DaemonSet agent for measuring node-to-node ICMP round-trip time (RTT). It discovers cluster nodes, probes peer node `InternalIP` addresses from the host network namespace, and exports Prometheus metrics for RTT and probe outcomes.

The project is designed to provide a cleaner infrastructure-level latency signal than application-layer health checks. It is intended for operators who want to understand whether latency or packet loss is coming from the underlying node network rather than from application handlers or service paths.

## Metrics

- `node_icmp_rtt_ms{src_node,dst_node,src_zone,dst_zone}`: latest successful RTT in milliseconds
- `node_icmp_probes_total{src_node,dst_node,result}`: total probes grouped by `success`, `timeout`, or `error`
- `node_icmp_last_success_unixtime{src_node,dst_node}`: Unix timestamp of the latest successful probe

## Configuration

All configuration is provided by environment variables.

| Variable | Default | Description |
| --- | --- | --- |
| `NODE_NAME` | required | Local node name, usually injected from `spec.nodeName` |
| `KUBECONFIG` | `$HOME/.kube/config` | Optional kubeconfig for local development when not running in cluster |
| `METRICS_ADDR` | `:9090` | Address for the HTTP metrics server |
| `PROBE_INTERVAL` | `10s` | Interval between full peer probe rounds |
| `PROBE_TIMEOUT` | `500ms` | Timeout for a single ICMP probe |
| `PROBE_JITTER_FACTOR` | `0.2` | Random delay factor applied per peer within each round |
| `EXCLUDE_NOT_READY` | `false` | Exclude nodes that are not Ready |
| `EXCLUDE_CONTROL_PLANE` | `false` | Exclude control-plane nodes |

`PROBE_TIMEOUT` must be shorter than `PROBE_INTERVAL`, and `PROBE_TIMEOUT + (PROBE_INTERVAL * PROBE_JITTER_FACTOR)` must also remain below `PROBE_INTERVAL` so probe rounds do not collapse into one another.

## Local build and test

```bash
go test ./...
go build ./cmd/node-pinger
```

To run locally against a kubeconfig:

```bash
NODE_NAME=<your-node-name> KUBECONFIG=$HOME/.kube/config go run ./cmd/node-pinger
```

Running probes requires raw ICMP sockets. On Linux that usually means running with the `NET_RAW` capability or as a sufficiently privileged user.

## Container image

Build the container image with:

```bash
docker build -t ghcr.io/tennix/node-pinger:latest .
```

## Kubernetes deployment

Apply the manifests:

```bash
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/daemonset.yaml
kubectl apply -f deploy/service.yaml
```

Deployment notes:

- the DaemonSet uses `hostNetwork: true`
- the container needs the `NET_RAW` capability to open ICMP sockets
- the RBAC scope is limited to `get`, `list`, and `watch` on `nodes`
- the Service is headless so scrapers can discover every DaemonSet pod instead of a single load-balanced endpoint
- the manifest intentionally does not use `hostPort`; with `hostNetwork: true`, the process already binds in the host network namespace

Replace the image reference in `deploy/daemonset.yaml` with your published image before deploying.

## Helm deployment

The repository also includes a Helm chart at `charts/node-pinger`.

```bash
helm install node-pinger ./charts/node-pinger --namespace kube-system --create-namespace
```

Update `image.repository` and `image.tag` as needed for your published image.

## TODO

- add Helm linting and render checks to CI
- add optional PodMonitor or ServiceMonitor support to the Helm chart
- document Prometheus scraping patterns for the headless DaemonSet Service
- add published image build and release instructions
- extend the implementation with sampled mesh, rolling loss, and jitter metrics
