# node-pinger Helm chart

This chart deploys `node-pinger` as a DaemonSet with the RBAC and Service resources required by the current MVP.

## Installing

```bash
helm install node-pinger ./charts/node-pinger --namespace kube-system --create-namespace
```

## Key values

- `image.repository`, `image.tag`, `image.pullPolicy`
- `metrics.port`
- `config.probeInterval`
- `config.probeTimeout`
- `config.probeJitterFactor`
- `config.excludeNotReady`
- `config.excludeControlPlane`
- `serviceAccount.create`, `serviceAccount.name`
- `rbac.create`
- `resources`
- `nodeSelector`, `tolerations`, `affinity`

The chart preserves the existing deployment assumptions from the plain manifests: `hostNetwork: true`, `dnsPolicy: ClusterFirstWithHostNet`, and the `NET_RAW` capability.

By default the chart creates a **headless** metrics Service so Prometheus-style discovery can see every DaemonSet pod instead of a single load-balanced virtual IP.
