{{/*
Expand the name of the chart.
*/}}
{{- define "manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "manager.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Print the image reference. Digest takes precedence over tag.
*/}}
{{- define "manager.image" -}}
{{- $repository := .Values.controllerManager.image.repository -}}
{{- $digest := .Values.controllerManager.image.digest -}}
{{- if $digest -}}
  {{- printf "%s@%s" $repository $digest -}}
{{- else -}}
  {{- printf "%s:%s" $repository (.Values.controllerManager.image.tag | default .Chart.AppVersion) -}}
{{- end -}}
{{- end -}}

{{/*
common helm lables
*/}}
{{- define "common.labels" -}}
helm.sh/chart: {{ include "manager.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
manager labels
*/}}
{{- define "manager.labels" -}}
{{ include "common.labels" . }}
{{ include "common.selectorLabels" . }}
{{ include "manager.selectorLabels" . }}
{{- end }}

{{/*
webhook labels
*/}}
{{- define "webhook.labels" -}}
{{ include "common.labels" . }}
{{ include "common.selectorLabels" . }}
{{ include "webhook.selectorLabels" . }}
{{- end}}

{{- define "common.selectorLabels" -}}
app: {{ include "manager.name" . }}
app.kubernetes.io/name: {{ include "manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end}}

{{- define "manager.selectorLabels" -}}
app.kubernetes.io/component: manager
{{- end}}

{{- define "webhook.selectorLabels" -}}
app.kubernetes.io/component: webhook
{{- end}}

{{/*
Create the name of the service account to use
*/}}
{{- define "manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "manager.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Define postgresql helpers
*/}}
{{- define "featureFlag.fullname" -}}
  {{- printf "%s-feature-flags" .Release.Name | trunc 48 | replace "_" "-" -}}
{{- end -}}
{{/* Render the backend */}}
{{- define "dex.backend" -}}
  {{- printf "%s" (required "global.dex.backend missing" .Values.global.dex.backend) }}
{{- end }}
{{/* Render the plugin option value templating flag */}}
{{- define "plugin.optionValueTemplating" -}}
  {{- printf "%t" (required "global.plugin.optionValueTemplating missing" .Values.global.plugin.optionValueTemplating) }}
{{- end }}
