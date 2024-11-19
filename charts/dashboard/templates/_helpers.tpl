{{/* Render the dashboard hostname */}}
{{- define "dashboard.hostname" -}}
{{- printf "%s.%s" "dashboard" (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}

{{/* Render the auth hostname */}}
{{- define "auth.hostname" -}}
{{- printf "https://%s.%s" "auth" (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}

{{/* Render the k8s api hostname */}}
{{- define "api.hostname" -}}
{{- printf "https://%s.%s" "api" (required "global.dnsDomain missing" .Values.global.dnsDomain) }}
{{- end }}
