{{- define "demoUserSecretName" -}}
{{- required ".Values.demoUser.firstName missing" .Values.demoUser.firstName }}-token
{{- end }}
