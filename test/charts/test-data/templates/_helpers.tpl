{{/*
Expand the name of the chart.
*/}}
{{- define "test-data.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}
