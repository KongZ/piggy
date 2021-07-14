{{/*
Expand the name of the chart.
*/}}
{{- define "piggy-webhooks.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "piggy-webhooks.fullname" -}}
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
{{- define "piggy-webhooks.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "piggy-webhooks.labels" -}}
helm.sh/chart: {{ include "piggy-webhooks.chart" . }}
{{ include "piggy-webhooks.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "piggy-webhooks.selectorLabels" -}}
app.kubernetes.io/name: {{ include "piggy-webhooks.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "piggy-webhooks.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "piggy-webhooks.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the cert manager
*/}}
{{- define "piggy-webhooks.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "piggy-webhooks.fullname" .) }}
{{- end -}}

{{- define "piggy-webhooks.rootCAIssuer" -}}
{{ printf "%s-ca" (include "piggy-webhooks.fullname" .) }}
{{- end -}}

{{- define "piggy-webhooks.rootCACertificate" -}}
{{ printf "%s-ca" (include "piggy-webhooks.fullname" .) }}
{{- end -}}

{{- define "piggy-webhooks.certificate" -}}
{{ printf "%s-webhook-tls" (include "piggy-webhooks.fullname" .) }}
{{- end -}}

{{/*
Overrideable version for container image tags.
*/}}
{{- define "piggy-webhooks.piggy-env.version" -}}
{{- .Values.mutate.image.tag | default (printf "%s" .Chart.AppVersion) -}}
{{- end -}}
