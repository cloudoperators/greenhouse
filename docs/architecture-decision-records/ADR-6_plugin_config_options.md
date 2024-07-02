# ADR-6 Plugin administrator / config options

## Decision Contributors

- ...

## Status

- Proposed

## Context and Problem Statement

Some plugins require or allow a lot of configuration options which are currently kubernetes CRD style situated in deeply nested object structures.
We auto generate the edit screens for the plugins and the deep nesting makes the forms hard to read and use.
Example: kubeMonitoring.prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage.

## Decision Drivers

- Stability:
  - ...

- Transparency:
  - ...

- Compatibility:
  - ...

## Decision
