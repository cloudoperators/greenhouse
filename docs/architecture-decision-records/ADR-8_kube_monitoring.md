# ADR-8 Kube-montoring next steps

## Decision Contributors

- ...

## Status

- Proposed

## Context and Problem Statement

Short/Mid-term:
Generalize alerting and aggreation rules to make them useful to non-ccloud use-cases.
Discuss mandatory labels, owner-info, etc. for alerting routing and operations. Should be enforced, defaulted and integrated across Greenhouse.
Discuss public (or at least configurable) place for Grafana dashboards, playbooks, etc.
Clarify ownership of plugin sub-components, e.g. Kubernetes rules going forward.

Long-term:
Unified Thanos-style solution for metrics and logs. Local storage of data and querier abstractions for data aggregation from multiple buckets.
Not for customer. Greenhouse is for operators.
Configurable policies for data access.





## Decision Drivers

- Stability:
  - ...

- Transparency:
  - ...

- Compatibility:
  - ...

## Decision

