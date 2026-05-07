{{- define "goflow.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goflow.fullname" -}}
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

{{- define "goflow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goflow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goflow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "goflow.labels" -}}
helm.sh/chart: {{ include "goflow.chart" . }}
{{ include "goflow.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{- define "goflow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "goflow.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "goflow.envConfigName" -}}
{{- printf "%s-env" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.envSecretName" -}}
{{- if .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecret -}}
{{- else -}}
{{- printf "%s-env" (include "goflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "goflow.zitadelSecretName" -}}
{{- if .Values.zitadel.existingSecret -}}
{{- .Values.zitadel.existingSecret -}}
{{- else -}}
{{- printf "%s-zitadel" (include "goflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "goflow.zitadelApiUrl" -}}
{{- printf "http://%s-zitadel-api:8080" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.zitadelBootstrapPvc" -}}
{{- printf "%s-zitadel-bootstrap" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.goflowBootstrapPvc" -}}
{{- printf "%s-goflow-bootstrap" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.commandServiceName" -}}
{{- printf "%s-command" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.queryServiceName" -}}
{{- printf "%s-query" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.frontendServiceName" -}}
{{- printf "%s-frontend" (include "goflow.fullname" .) -}}
{{- end -}}

{{- define "goflow.gatewayServiceName" -}}
{{- printf "%s-gateway" (include "goflow.fullname" .) -}}
{{- end -}}
