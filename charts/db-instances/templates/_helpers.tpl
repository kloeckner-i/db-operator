{{/* vim: set filetype=mustache: */}}

{{- define "db-instances.smName" -}}
{{- default .Release.Name .Values.serviceMonitorName | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "db-instances.operatorNs" -}}
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