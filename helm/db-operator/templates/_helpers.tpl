{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "db-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "db-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "db-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Image version definition;
*/}}
{{- define "db-operator.image_version" -}}
{{ default .Chart.AppVersion .Values.image.tag }}
{{- end -}}

{{/*
Image version definition using Github Packages format ('v' prefix);
*/}}
{{- define "db-operator.github_packages_image_version" -}}
{{- printf "v%s" (default .Chart.AppVersion .Values.image.tag) }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "db-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- printf "%s-sa" (include "db-operator.name" .) -}}
{{- else -}}
{{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "db-operator.labels" -}}
helm.sh/chart: {{ include "db-operator.chart" . }}
{{ include "db-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "db-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "db-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Merge of generated db crd resource with helm related labels
*/}}
{{- define "db-operator.crd_db" -}}
{{ toYaml (merge (.Files.Get "files/gen/crd/kci.rocks_databases.yaml" | fromYaml) (dict "metadata" (dict "labels" (include "db-operator.labels" . | fromYaml  ) ) ) ) }}
{{- end -}}


{{/*
Merge of generated dbin crd resource with helm related labels
*/}}
{{- define "db-operator.crd_dbin" -}}
{{ toYaml (merge (.Files.Get "files/gen/crd/kci.rocks_dbinstances.yaml" | fromYaml) (dict "metadata" (dict "labels" (include "db-operator.labels" . | fromYaml  ) ) ) ) }}
{{- end -}}

{{/*
Merge of generated rbac resource with helm related labels
*/}}
{{- define "db-operator.rbac_role" -}}
{{ toYaml (merge (.Files.Get "files/gen/rbac/role.yaml" | fromYaml) (dict "metadata" (dict "labels" (include "db-operator.labels" . | fromYaml ) ) ) ) }}
{{- end -}}