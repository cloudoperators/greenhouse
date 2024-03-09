{{/*
Expand the name of the chart.
*/}}
{{- define "headscale.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "headscale.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "headscale.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "common.labels" -}}
helm.sh/chart: {{ include "headscale.chart" . }}
{{ include "common.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Headscale labels
*/}}
{{- define "headscale.labels" -}}
{{- include "common.labels" .}}
app.kubernetes.io/component: headscale
{{- end -}}

{{/*
Common selector labels
*/}}
{{- define "common.selectorLabels" -}}
app.kubernetes.io/name: {{ include "headscale.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Headscale selector labels
*/}}
{{- define "headscale.selectorLabels" -}}
{{- include "common.selectorLabels" .}}
app.kubernetes.io/component: headscale
{{- end -}}

{{/*
Postgres labels
*/}}
{{- define "postgres.labels"}}
{{- include "common.labels" . }}
app.kubernetes.io/component: postgres
{{- end }}

{{- define "postgres.fullname" -}}
{{- include "headscale.fullname" . }}-postgres
{{- end }}

{{/*
Postgres selector labels
*/}}
{{- define "postgres.selectorLabels" -}}
{{- include "common.selectorLabels" . }}
app.kubernetes.io/component: postgres
{{- end }}
