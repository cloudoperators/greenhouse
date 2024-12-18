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
