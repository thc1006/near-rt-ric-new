{{/*
Expand the name of the chart.
*/}}
{{- define "oran-sc-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "oran-sc-platform.fullname" -}}
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
{{- define "oran-sc-platform.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "oran-sc-platform.labels" -}}
helm.sh/chart: {{ include "oran-sc-platform.chart" . }}
{{ include "oran-sc-platform.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
platform: {{ .Values.global.labels.platform }}
release: {{ .Values.global.labels.release }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "oran-sc-platform.selectorLabels" -}}
app.kubernetes.io/name: {{ include "oran-sc-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "oran-sc-platform.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "oran-sc-platform.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate RMR routing configuration
*/}}
{{- define "oran-sc-platform.rmrRouting" -}}
{{- $components := list "e2term" "e2mgr" "submgr" "rtmgr" "a1mediator" "o1mediator" "appmgr" }}
{{- range $component := $components }}
{{- if index $.Values $component "enabled" }}
- name: {{ $component }}
  service: service-ricplt-{{ $component }}-rmr
  port: {{ $.Values.global.rmr.data_port }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Generate TLS certificate configuration
*/}}
{{- define "oran-sc-platform.tlsConfig" -}}
{{- if .Values.security.tls.enabled }}
tls:
  enabled: true
  ca_cert: /etc/ssl/certs/ca.crt
  server_cert: /etc/ssl/certs/server.crt
  server_key: /etc/ssl/private/server.key
  client_cert: /etc/ssl/certs/client.crt
  client_key: /etc/ssl/private/client.key
{{- end }}
{{- end }}