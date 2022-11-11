{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "acmesh-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "acmesh-webhook.fullname" -}}
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
{{- define "acmesh-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "acmesh-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "acmesh-webhook.fullname" .) }}
{{- end -}}

{{- define "acmesh-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "acmesh-webhook.fullname" .) }}
{{- end -}}

{{- define "acmesh-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "acmesh-webhook.fullname" .) }}
{{- end -}}

{{- define "acmesh-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "acmesh-webhook.fullname" .) }}
{{- end -}}

{{- define "acmesh-webhook.clusterIssuer" -}}
{{- if .Values.clusterIssuer.name -}}
{{ .Values.clusterIssuer.name }}
{{- else -}}
{{ printf "%s-cluster-issuer" (include "acmesh-webhook.fullname" .) }}
{{- end -}}
{{- end -}}
