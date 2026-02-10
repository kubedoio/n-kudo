{{/*
Expand the name of the chart.
*/}}
{{- define "nkudo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nkudo.fullname" -}}
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
{{- define "nkudo.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nkudo.labels" -}}
helm.sh/chart: {{ include "nkudo.chart" . }}
{{ include "nkudo.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nkudo.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nkudo.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nkudo.serviceAccountName" -}}
{{- if .Values.rbac.serviceAccount.create }}
{{- default (include "nkudo.fullname" .) .Values.rbac.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.rbac.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the secret to use for database credentials
*/}}
{{- define "nkudo.dbSecretName" -}}
{{- if .Values.database.external.existingSecret }}
{{- .Values.database.external.existingSecret }}
{{- else if .Values.postgresql.enabled }}
{{- if .Values.postgresql.auth.existingSecret }}
{{- .Values.postgresql.auth.existingSecret }}
{{- else }}
{{- include "nkudo.fullname" . }}-postgresql
{{- end }}
{{- else }}
{{- include "nkudo.fullname" . }}-db
{{- end }}
{{- end }}

{{/*
Get database password key
*/}}
{{- define "nkudo.dbPasswordKey" -}}
{{- if .Values.database.external.existingSecret }}
{{- .Values.database.external.existingSecretPasswordKey }}
{{- else if .Values.postgresql.enabled }}
{{- if .Values.postgresql.auth.existingSecret }}
password
{{- else }}
password
{{- end }}
{{- else }}
{{- .Values.database.external.existingSecretPasswordKey }}
{{- end }}
{{- end }}

{{/*
Create the database connection string
*/}}
{{- define "nkudo.databaseUrl" -}}
{{- if .Values.postgresql.enabled }}
{{- $postgresFullname := include "nkudo.postgresql.fullname" . }}
{{- $postgresPort := 5432 }}
{{- printf "postgresql://%s:%s@%s:%d/%s?sslmode=disable" .Values.postgresql.auth.username (include "nkudo.postgresql.password" .) $postgresFullname $postgresPort .Values.postgresql.auth.database }}
{{- else }}
{{- printf "postgresql://%s:$(DB_PASSWORD)@%s:%d/%s?sslmode=%s" .Values.database.external.user .Values.database.external.host (int .Values.database.external.port) .Values.database.external.database .Values.database.external.sslMode }}
{{- end }}
{{- end }}

{{/*
PostgreSQL fullname helper (when using subchart)
*/}}
{{- define "nkudo.postgresql.fullname" -}}
{{- printf "%s-postgresql" (include "nkudo.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Get PostgreSQL password
*/}}
{{- define "nkudo.postgresql.password" -}}
{{- if .Values.postgresql.auth.password }}
{{- .Values.postgresql.auth.password }}
{{- else }}
{{- randAlphaNum 16 }}
{{- end }}
{{- end }}

{{/*
Create the name of the secret for security credentials
*/}}
{{- define "nkudo.securitySecretName" -}}
{{- include "nkudo.fullname" . }}-security
{{- end }}

{{/*
Create the name of the secret for CA
*/}}
{{- define "nkudo.caSecretName" -}}
{{- if .Values.security.mtls.existingCASecret }}
{{- .Values.security.mtls.existingCASecret }}
{{- else }}
{{- include "nkudo.fullname" . }}-ca
{{- end }}
{{- end }}

{{/*
Create the name of the configmap
*/}}
{{- define "nkudo.configMapName" -}}
{{- include "nkudo.fullname" . }}-config
{{- end }}

{{/*
Create the name of the persistent volume claim
*/}}
{{- define "nkudo.pvcName" -}}
{{- if .Values.persistence.existingClaim }}
{{- .Values.persistence.existingClaim }}
{{- else }}
{{- include "nkudo.fullname" . }}-data
{{- end }}
{{- end }}

{{/*
Create the image path
*/}}
{{- define "nkudo.image" -}}
{{- $registry := .Values.global.imageRegistry | default .Values.image.registry }}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s/%s:%s" $registry .Values.image.repository $tag }}
{{- end }}

{{/*
Check if migration job should run
*/}}
{{- define "nkudo.migration.enabled" -}}
{{- if and .Values.migration.enabled .Values.postgresql.enabled }}
true
{{- else if and .Values.migration.enabled (not .Values.postgresql.enabled) }}
true
{{- else }}
false
{{- end }}
{{- end }}

{{/*
Generate CA certificate
*/}}
{{- define "nkudo.generateCA" -}}
{{- $ca := genCA "nkudo-ca" 3650 }}
{{- $_ := set . "_caCert" $ca.Cert }}
{{- $_ := set . "_caKey" $ca.Key }}
{{- end }}

{{/*
Get or generate admin key
*/}}
{{- define "nkudo.adminKey" -}}
{{- if .Values.security.adminKey }}
{{- .Values.security.adminKey }}
{{- else }}
{{- randAlphaNum 32 }}
{{- end }}
{{- end }}

{{/*
Get or generate JWT secret
*/}}
{{- define "nkudo.jwtSecret" -}}
{{- if .Values.security.jwt.secret }}
{{- .Values.security.jwt.secret }}
{{- else }}
{{- randAlphaNum 64 }}
{{- end }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "nkudo.podLabels" -}}
{{ include "nkudo.selectorLabels" . }}
{{- with .Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "nkudo.podAnnotations" -}}
{{- if .Values.monitoring.enabled }}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .Values.service.metricsPort | quote }}
prometheus.io/path: "/metrics"
{{- end }}
{{- with .Values.podAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Get S3 secret name
*/}}
{{- define "nkudo.s3SecretName" -}}
{{- if .Values.backup.s3.existingSecret }}
{{- .Values.backup.s3.existingSecret }}
{{- else }}
{{- include "nkudo.fullname" . }}-backup
{{- end }}
{{- end }}

{{/*
Volume mounts for certificates
*/}}
{{- define "nkudo.certVolumeMounts" -}}
{{- if .Values.security.mtls.enabled }}
- name: ca-certs
  mountPath: /etc/nkudo/certs
  readOnly: true
{{- end }}
{{- end }}

{{/*
Volumes for certificates
*/}}
{{- define "nkudo.certVolumes" -}}
{{- if .Values.security.mtls.enabled }}
- name: ca-certs
  secret:
    secretName: {{ include "nkudo.caSecretName" . }}
    defaultMode: 0400
{{- end }}
{{- end }}

{{/*
Create the ingress TLS secret name
*/}}
{{- define "nkudo.ingressTlsSecretName" -}}
{{- if .Values.ingress.tls.secretName }}
{{- .Values.ingress.tls.secretName }}
{{- else }}
{{- include "nkudo.fullname" . }}-tls
{{- end }}
{{- end }}

{{/*
Create the ingress gRPC TLS secret name
*/}}
{{- define "nkudo.ingressGrpcTlsSecretName" -}}
{{- if .Values.ingress.tls.secretName }}
{{- .Values.ingress.tls.secretName }}
{{- else }}
{{- include "nkudo.fullname" . }}-grpc-tls
{{- end }}
{{- end }}

{{/*
Validate configuration
*/}}
{{- define "nkudo.validate" -}}
{{- if and (not .Values.postgresql.enabled) (not .Values.database.external.host) }}
{{- fail "Either postgresql.enabled must be true or database.external.host must be set" }}
{{- end }}
{{- end }}
