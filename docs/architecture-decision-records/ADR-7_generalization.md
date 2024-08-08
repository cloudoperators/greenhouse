# ADR-7 SAP/CCloud-specific leftovers, Generalizations for public use

## Decision Contributors

- ...

## Status

- Proposed

## Context and Problem Statement

Identify where we currently still have CCloud or SAP specific configurations, helm charts, pipelines, … that we might need to generalize and open source properly so that outside people can install/run our stuff
Provide a semver-stable API.

DOOP vs Heureka

DOOP has a shiny new UI, but the gatekeeper backend is deployed manually and is CCloud-specific.
Before investing more, alignment with David Rochow and team is required on the holistic security & compliance package and its integration in Greenhouse.

Monitoring

Alerting, Routing, Labels, owner-label injector, etc.
Discuss public (or at least configurable) place for Grafana dashboards, playbooks

Opinionated vs. too specific



## Decision Drivers

- Stability:
  - ...

- Transparency:
  - ...

- Compatibility:
  - ...

## Decision

