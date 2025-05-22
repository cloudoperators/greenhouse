{{/* Render the dashboard hostname */}}
{{- define "dashboard.hostname" -}}
{{- printf "%s.%s" "dashboard" (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}

{{/* Render the auth hostname */}}
{{- define "dashboard.auth.hostname" -}}
{{- printf "https://%s.%s" "auth" (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}

{{/* Render the k8s api hostname */}}
{{- define "dashboard.api.hostname" -}}
{{- printf "https://%s.%s" (default "api" .Values.global.kubeAPISubDomain) (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}

{{/*
Print the image reference. Digest takes precedence over tag.
*/}}
{{- define "dashboard.image" -}}
{{- $repository := ( required ".Values.image.repository missing" .Values.image.repository ) -}}
{{- $digest := .Values.image.digest -}}
{{- if $digest -}}
  {{- printf "%s@%s" $repository $digest -}}
{{- else -}}
  {{- printf "%s:%s" $repository ( required ".Values.image.tag missing" .Values.image.tag ) -}}
{{- end -}}
{{- end -}}
