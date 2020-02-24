{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "db-instances.name" -}}
{{- default .Release.Namespace .Values.operatorNamespace -}}
{{- end -}}
