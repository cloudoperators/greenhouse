{{- define "manager.params" -}}
{{- range .Values.controllerManager.args }}
- {{ . }}
{{- end }}
{{- end -}}
