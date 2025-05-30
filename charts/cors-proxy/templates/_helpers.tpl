{{/*
Expand the name of the chart.
*/}}
{{- define "cors-proxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cors-proxy.fullname" -}}
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
{{- define "cors-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Print the image reference. Digest takes precedence over tag.
*/}}
{{- define "cors-proxy.image" -}}
{{- $repository := .Values.image.repository -}}
{{- $digest := .Values.image.digest -}}
{{- if $digest -}}
  {{- printf "%s@%s" $repository $digest -}}
{{- else -}}
  {{- printf "%s:%s" $repository (.Values.image.tag | default .Chart.AppVersion) -}}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "cors-proxy.labels" -}}
helm.sh/chart: {{ include "cors-proxy.chart" . }}
{{ include "cors-proxy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cors-proxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cors-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cors-proxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cors-proxy.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/* Render the k8s api hostname */}}
{{- define "cors-proxy.api.hostname" -}}
{{- printf "%s.%s" (default "api" .Values.global.kubeAPISubDomain) (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}
