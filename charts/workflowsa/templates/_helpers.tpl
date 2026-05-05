{{- define "workflowsa.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "workflowsa.fullname" -}}
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

{{- define "workflowsa.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "workflowsa.selectorLabels" -}}
app.kubernetes.io/name: {{ include "workflowsa.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "workflowsa.labels" -}}
helm.sh/chart: {{ include "workflowsa.chart" . }}
{{ include "workflowsa.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{- define "workflowsa.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "workflowsa.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "workflowsa.envConfigName" -}}
{{- printf "%s-env" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.envSecretName" -}}
{{- if .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecret -}}
{{- else -}}
{{- printf "%s-env" (include "workflowsa.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "workflowsa.zitadelSecretName" -}}
{{- if .Values.zitadel.existingSecret -}}
{{- .Values.zitadel.existingSecret -}}
{{- else -}}
{{- printf "%s-zitadel" (include "workflowsa.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "workflowsa.zitadelApiUrl" -}}
{{- printf "http://%s-zitadel-api:8080" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.zitadelBootstrapPvc" -}}
{{- printf "%s-zitadel-bootstrap" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.workflowsaBootstrapPvc" -}}
{{- printf "%s-workflowsa-bootstrap" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.commandServiceName" -}}
{{- printf "%s-command" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.queryServiceName" -}}
{{- printf "%s-query" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.frontendServiceName" -}}
{{- printf "%s-frontend" (include "workflowsa.fullname" .) -}}
{{- end -}}

{{- define "workflowsa.gatewayServiceName" -}}
{{- printf "%s-gateway" (include "workflowsa.fullname" .) -}}
{{- end -}}
