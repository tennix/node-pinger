# node-pinger design

**Status:** Draft  
**Created:** 2026-04-09

## Summary

`node-pinger` is a Kubernetes-native DaemonSet for measuring node-to-node ICMP round-trip time (RTT). Its purpose is to provide a clean infrastructure-level latency signal that is easier to trust than application-layer probe latency.

Each agent runs on one node, discovers peer nodes through the Kubernetes API, probes peer `InternalIP` addresses from the host network namespace, and exposes Prometheus metrics.

## Problem

Operators often need to answer questions such as:

- is node-to-node latency elevated?
- is one destination node timing out for many peers?
- is cross-AZ latency worse than normal?
- are application latency spikes caused by the underlying node network?

Existing tools do not fit this problem well:

- application probes measure too much user-space and handler overhead
- generic probers do not provide Kubernetes-native node mesh behavior
- traditional ICMP tools do not integrate cleanly with dynamic Kubernetes node membership

## Goals

- discover nodes automatically from the Kubernetes API
- measure host-level RTT with ICMP
- run natively inside Kubernetes as a DaemonSet
- export Prometheus metrics for dashboards and alerting
- support full mesh for small clusters and sampled mesh for larger ones
- preserve enough topology metadata for node and AZ analysis

## Non-goals

- replace application health checks
- validate Service VIP or kube-proxy behavior
- provide deep packet tracing or path diagnostics
- act as a general-purpose external blackbox prober

## Design

The system runs one agent per node.

Each agent is responsible for:

- watching Kubernetes `Node` objects
- identifying the local node
- selecting peers to probe
- sending ICMP echo requests and matching replies
- calculating RTT from local send/receive timing
- exporting metrics through `/metrics`

The deployment model assumes:

- `hostNetwork: true`
- `dnsPolicy: ClusterFirstWithHostNet`
- `NET_RAW`
- RBAC limited to `get`, `list`, and `watch` on `nodes`

## Measurement model

The sender timestamps each probe locally and computes RTT as local elapsed time. This avoids any need for clock synchronization between nodes.

The primary operator-facing signal should come from directly measured RTT samples rather than histogram interpolation.

Recommended core metrics:

- `node_icmp_rtt_ms`
- `node_icmp_probes_total`
- `node_icmp_last_success_unixtime`

## Topology model

Two operating modes are expected:

- **full mesh** for small clusters and validation runs
- **sampled mesh** for larger clusters, using stable peer selection and gradual rotation

Sampled mesh is the main scaling tool for reducing traffic and metric cardinality.

## Why ICMP

ICMP is the base signal because it is closer to raw network behavior than HTTP latency and avoids most application-layer noise. TCP or UDP probing can be added later for broader observability, but they are not the primary signal in this design.

## Rollout plan

### Phase 1

- DaemonSet deployment
- node discovery
- full mesh probing
- jittered scheduling
- basic RTT and probe-result metrics

### Phase 2

- sampled mesh
- topology-aware peer selection
- rolling loss and jitter metrics
- recording rules and alerting

### Phase 3

- optional TCP connect probing
- optional UDP-based loss or jitter probing

## Risks

- some clusters may restrict `NET_RAW` or `hostNetwork`
- per-pair metrics can become expensive at scale
- poor jitter settings can create synchronized bursts
- stale peer state can cause misleading metrics if node lifecycle handling is wrong

## Conclusion

`node-pinger` is intended to provide a simple, trustworthy view of Kubernetes node network latency. The core idea is straightforward: discover nodes, probe peers with ICMP, and export metrics that operators can use to understand real node-to-node RTT.
