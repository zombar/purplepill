{{/*
Expand the name of the chart.
*/}}
{{- define "docutag.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "docutag.fullname" -}}
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
{{- define "docutag.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "docutag.labels" -}}
helm.sh/chart: {{ include "docutag.chart" . }}
{{ include "docutag.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "docutag.selectorLabels" -}}
app.kubernetes.io/name: {{ include "docutag.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component-specific labels
Usage: {{ include "docutag.componentLabels" (dict "component" "controller" "context" .) }}
*/}}
{{- define "docutag.componentLabels" -}}
{{ include "docutag.labels" .context }}
app.kubernetes.io/component: {{ .component }}
app: docutag
app.type: {{ .type | default "backend" }}
{{- end }}

{{/*
Component-specific selector labels
Usage: {{ include "docutag.componentSelectorLabels" (dict "component" "controller" "context" .) }}
*/}}
{{- define "docutag.componentSelectorLabels" -}}
{{ include "docutag.selectorLabels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "docutag.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "docutag.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
Usage: {{ include "docutag.image" (dict "imageRoot" .Values.controller.image "global" .Values.global "context" .) }}
*/}}
{{- define "docutag.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | default .context.Chart.AppVersion | toString -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
     {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
    {{- if .global.imageVersion }}
     {{- $tag = .global.imageVersion | toString -}}
    {{- end -}}
{{- end -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper image pull policy
*/}}
{{- define "docutag.imagePullPolicy" -}}
{{- if .global }}
{{- .global.imagePullPolicy | default "IfNotPresent" -}}
{{- else -}}
{{- "IfNotPresent" -}}
{{- end -}}
{{- end -}}

{{/*
Return the storage class name
*/}}
{{- define "docutag.storageClass" -}}
{{- if .global }}
{{- .global.storageClass | default "do-block-storage" -}}
{{- else -}}
{{- "do-block-storage" -}}
{{- end -}}
{{- end -}}

{{/*
Return the domain name
*/}}
{{- define "docutag.domain" -}}
{{- if .Values.global }}
{{- .Values.global.domain | default "docutag.local" -}}
{{- else -}}
{{- "docutag.local" -}}
{{- end -}}
{{- end -}}

{{/*
Return Redis address
*/}}
{{- define "docutag.redisAddr" -}}
{{- if .Values.redis.enabled -}}
{{- printf "%s-redis-master:6379" (include "docutag.fullname" .) -}}
{{- else -}}
{{- .Values.externalRedis.host -}}:{{- .Values.externalRedis.port -}}
{{- end -}}
{{- end -}}

{{/*
Return PostgreSQL host
*/}}
{{- define "docutag.postgresqlHost" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-postgresql" (include "docutag.fullname" .) -}}
{{- else -}}
{{- .Values.externalDatabase.host -}}
{{- end -}}
{{- end -}}

{{/*
Return PostgreSQL port
*/}}
{{- define "docutag.postgresqlPort" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.primary.service.ports.postgresql | default 5432 -}}
{{- else -}}
{{- .Values.externalDatabase.port -}}
{{- end -}}
{{- end -}}

{{/*
Return PostgreSQL database name
*/}}
{{- define "docutag.postgresqlDatabase" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.database -}}
{{- else -}}
{{- .Values.externalDatabase.database -}}
{{- end -}}
{{- end -}}

{{/*
Return PostgreSQL username
*/}}
{{- define "docutag.postgresqlUsername" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.username -}}
{{- else -}}
{{- .Values.externalDatabase.username -}}
{{- end -}}
{{- end -}}

{{/*
Return PostgreSQL password secret name
*/}}
{{- define "docutag.postgresqlSecretName" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-postgresql" (include "docutag.fullname" .) -}}
{{- else -}}
{{- printf "%s-external-db" (include "docutag.fullname" .) -}}
{{- end -}}
{{- end -}}
