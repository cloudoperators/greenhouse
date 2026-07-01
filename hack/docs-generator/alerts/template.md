---
title: "Playbooks"
linkTitle: "Playbooks"
landingSectionIndex: false
weight: 2
description: >
  Playbooks for the alerts produced by PrometheusRules deployed with the
  Greenhouse manager chart, grouped by which team is paged.
---

<!-- This page is auto-generated from charts/manager/alerts/. Do not edit by hand — run `make generate-alerts-doc` to regenerate. -->

This page is auto-generated from `charts/manager/alerts/`. Do not edit by hand — run `make generate-alerts-doc` to regenerate.
{{- range .Sections }}

## {{ .Heading }}

{{ .Description }}
{{ range .Rules }}
- {{ if eq .Playbook "—" }}`{{ .Alert }}`{{ else }}[`{{ .Alert }}`]({{ .Playbook }}){{ end }}  
  {{ .Summary }}
{{- end }}
{{- end }}
