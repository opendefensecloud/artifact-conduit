{{/*
Expand the name of the chart.
*/}}
{{- define "arc.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "arc.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "arc.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "arc.labels" -}}
helm.sh/chart: {{ include "arc.chart" . }}
{{ include "arc.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "arc.selectorLabels" -}}
app.kubernetes.io/name: {{ include "arc.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "arc.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
API Server fullname
*/}}
{{- define "arc.apiserver.fullname" -}}
{{- printf "%s-apiserver" (include "arc.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
API Server component labels
*/}}
{{- define "arc.apiserver.labels" -}}
{{ include "arc.labels" . }}
app.kubernetes.io/component: apiserver
app.kubernetes.io/part-of: arc
{{- end }}

{{/*
API Server selector labels
*/}}
{{- define "arc.apiserver.selectorLabels" -}}
{{ include "arc.selectorLabels" . }}
app.kubernetes.io/component: apiserver
{{- end }}

{{/*
API Server service account name
*/}}
{{- define "arc.apiserver.serviceAccountName" -}}
{{- if .Values.apiserver.serviceAccount.create }}
{{- default (include "arc.apiserver.fullname" .) .Values.apiserver.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.apiserver.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Controller fullname
*/}}
{{- define "arc.controller.fullname" -}}
{{- printf "%s-controller-manager" (include "arc.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Controller component labels
*/}}
{{- define "arc.controller.labels" -}}
{{ include "arc.labels" . }}
app.kubernetes.io/component: controller-manager
app.kubernetes.io/part-of: arc
{{- end }}

{{/*
Controller selector labels
*/}}
{{- define "arc.controller.selectorLabels" -}}
{{ include "arc.selectorLabels" . }}
app.kubernetes.io/component: controller-manager
{{- end }}

{{/*
Controller service account name
*/}}
{{- define "arc.controller.serviceAccountName" -}}
{{- if .Values.controller.serviceAccount.create }}
{{- default (include "arc.controller.fullname" .) .Values.controller.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.controller.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
etcd fullname
*/}}
{{- define "arc.etcd.fullname" -}}
{{- printf "%s-etcd" (include "arc.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
etcd component labels
*/}}
{{- define "arc.etcd.labels" -}}
{{ include "arc.labels" . }}
app.kubernetes.io/component: etcd
app.kubernetes.io/part-of: arc
{{- end }}

{{/*
etcd selector labels
*/}}
{{- define "arc.etcd.selectorLabels" -}}
{{ include "arc.selectorLabels" . }}
app.kubernetes.io/component: etcd
{{- end }}

{{/*
etcd service name
*/}}
{{- define "arc.etcd.serviceName" -}}
{{- include "arc.etcd.fullname" . }}
{{- end }}

{{/*
etcd connection URL
*/}}
{{- define "arc.etcd.connectionUrl" -}}
{{- if .Values.apiserver.args.etcdServers }}
{{- .Values.apiserver.args.etcdServers }}
{{- else }}
{{- printf "http://%s:2379" (include "arc.etcd.serviceName" .) }}
{{- end }}
{{- end }}

{{/*
Image pull secrets
*/}}
{{- define "arc.imagePullSecrets" -}}
{{- $secrets := list }}
{{- if .Values.global.imagePullSecrets }}
{{- $secrets = concat $secrets .Values.global.imagePullSecrets }}
{{- end }}
{{- if .component.imagePullSecrets }}
{{- $secrets = concat $secrets .component.imagePullSecrets }}
{{- end }}
{{- if $secrets }}
imagePullSecrets:
{{- range $secrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
API Server image
*/}}
{{- define "arc.apiserver.image" -}}
{{- $tag := .Values.apiserver.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.apiserver.image.repository $tag }}
{{- end }}

{{/*
Controller image
*/}}
{{- define "arc.controller.image" -}}
{{- $tag := .Values.controller.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.controller.image.repository $tag }}
{{- end }}

{{/*
etcd image
*/}}
{{- define "arc.etcd.image" -}}
{{- printf "%s:%s" .Values.etcd.image.repository .Values.etcd.image.tag }}
{{- end }}

{{/*
cert-manager Issuer name
*/}}
{{- define "arc.certManager.issuerName" -}}
{{- if .Values.certManager.issuer.name }}
{{- .Values.certManager.issuer.name }}
{{- else }}
{{- printf "%s-selfsigned-issuer" (include "arc.apiserver.fullname" .) }}
{{- end }}
{{- end }}

{{/*
cert-manager Certificate name
*/}}
{{- define "arc.certManager.certificateName" -}}
{{- printf "%s-cert" (include "arc.apiserver.fullname" .) }}
{{- end }}

{{/*
cert-manager Certificate secret name
*/}}
{{- define "arc.certManager.certificateSecretName" -}}
{{- printf "%s-cert" (include "arc.apiserver.fullname" .) }}
{{- end }}

{{/*
API Server service name
*/}}
{{- define "arc.apiserver.serviceName" -}}
{{- printf "%s-service" (include "arc.apiserver.fullname" .) }}
{{- end }}

{{/*
Controller metrics service name
*/}}
{{- define "arc.controller.metricsServiceName" -}}
{{- printf "%s-metrics" (include "arc.controller.fullname" .) }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "arc.annotations" -}}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}
