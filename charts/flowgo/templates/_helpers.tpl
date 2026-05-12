{{- define "flowgo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "flowgo.fullname" -}}
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

{{- define "flowgo.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "flowgo.selectorLabels" -}}
app.kubernetes.io/name: {{ include "flowgo.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "flowgo.labels" -}}
helm.sh/chart: {{ include "flowgo.chart" . }}
{{ include "flowgo.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{- define "flowgo.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "flowgo.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "flowgo.envConfigName" -}}
{{- printf "%s-env" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.envSecretName" -}}
{{- if .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecret -}}
{{- else -}}
{{- printf "%s-env" (include "flowgo.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "flowgo.zitadelSecretName" -}}
{{- if .Values.zitadel.existingSecret -}}
{{- .Values.zitadel.existingSecret -}}
{{- else -}}
{{- printf "%s-zitadel" (include "flowgo.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "flowgo.zitadelApiUrl" -}}
{{- printf "http://%s-zitadel-api:8080" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.zitadelBootstrapPvc" -}}
{{- printf "%s-zitadel-bootstrap" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.flowgoBootstrapPvc" -}}
{{- printf "%s-flowgo-bootstrap" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.commandServiceName" -}}
{{- printf "%s-command" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.queryServiceName" -}}
{{- printf "%s-query" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.frontendServiceName" -}}
{{- printf "%s-frontend" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.gatewayServiceName" -}}
{{- printf "%s-gateway" (include "flowgo.fullname" .) -}}
{{- end -}}
