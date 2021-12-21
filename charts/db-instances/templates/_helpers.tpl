{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "db-instances.name" -}}
{{- default .Release.Namespace .Values.operatorNamespace -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "db-instances.labels" -}}
{{- include "db-instances.selectorLabels" . -}}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "db-instances.selectorLabels" -}}
chart: {{ .Chart.Name }}
release: {{ $.Release.Name | quote}}
heritage: {{ $.Release.Service | quote}}
{{- end -}}