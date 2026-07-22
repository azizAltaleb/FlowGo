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

{{- define "flowgo.flowgoAuthPvc" -}}
{{- printf "%s-flowgo-auth" (include "flowgo.fullname" .) -}}
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

{{- define "flowgo.syncWorkerServiceName" -}}
{{- printf "%s-sync-worker" (include "flowgo.fullname" .) -}}
{{- end -}}

{{- define "flowgo.validateValues" -}}
{{- $mode := .Values.iam.mode | default "external" -}}
{{- if not (has $mode (list "external" "zitadel" "disabled")) -}}
{{- fail "iam.mode must be one of: external, zitadel, disabled" -}}
{{- end -}}
{{- if and .Values.zitadel.enabled (ne $mode "zitadel") -}}
{{- fail "zitadel.enabled=true requires iam.mode=zitadel" -}}
{{- end -}}
{{- if eq $mode "zitadel" -}}
{{- if not .Values.zitadel.enabled -}}
{{- fail "iam.mode=zitadel requires zitadel.enabled=true" -}}
{{- end -}}
{{- if and (not .Values.zitadel.existingSecret) (ne (len .Values.zitadel.masterkey) 32) -}}
{{- fail "zitadel.masterkey must be exactly 32 characters when the chart creates the ZITADEL secret" -}}
{{- end -}}
{{- $accessTokenLifetime := toString .Values.zitadel.bootstrap.accessTokenLifetime -}}
{{- if not (regexMatch "^[1-9][0-9]*(s|m|h)$" $accessTokenLifetime) -}}
{{- fail "zitadel.bootstrap.accessTokenLifetime must be a positive duration using s, m, or h" -}}
{{- end -}}
{{- $accessTokenLifetimeAmount := int (regexFind "^[0-9]+" $accessTokenLifetime) -}}
{{- if or (and (hasSuffix "s" $accessTokenLifetime) (or (lt $accessTokenLifetimeAmount 60) (gt $accessTokenLifetimeAmount 86400))) (and (hasSuffix "m" $accessTokenLifetime) (gt $accessTokenLifetimeAmount 1440)) (and (hasSuffix "h" $accessTokenLifetime) (gt $accessTokenLifetimeAmount 24)) -}}
{{- fail "zitadel.bootstrap.accessTokenLifetime must be between 60s and 24h" -}}
{{- end -}}
{{- if not (regexMatch "^[1-9][0-9]*(s|m|h)$" (toString .Values.zitadel.bootstrap.clientKeyDefaultLifetime)) -}}
{{- fail "zitadel.bootstrap.clientKeyDefaultLifetime must be a positive duration using s, m, or h" -}}
{{- end -}}
{{- if not (regexMatch "^[1-9][0-9]*(s|m|h)$" (toString .Values.zitadel.bootstrap.clientKeyMaxLifetime)) -}}
{{- fail "zitadel.bootstrap.clientKeyMaxLifetime must be a positive duration using s, m, or h" -}}
{{- end -}}
{{- if lt (int .Values.zitadel.api.machineKeySize) 2048 -}}
{{- fail "zitadel.api.machineKeySize must be at least 2048" -}}
{{- end -}}
{{- if lt (int .Values.zitadel.api.applicationKeySize) 2048 -}}
{{- fail "zitadel.api.applicationKeySize must be at least 2048" -}}
{{- end -}}
{{- if not (regexMatch "^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$" .Values.zitadel.bootstrap.ownerPatExpiration) -}}
{{- fail "zitadel.bootstrap.ownerPatExpiration must be an RFC3339 UTC timestamp" -}}
{{- end -}}
{{- if not (regexMatch "^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$" .Values.zitadel.bootstrap.loginPatExpiration) -}}
{{- fail "zitadel.bootstrap.loginPatExpiration must be an RFC3339 UTC timestamp" -}}
{{- end -}}
{{- end -}}
{{- if eq $mode "external" -}}
{{- if not .Values.iam.auth.issuerInternalUrl -}}
{{- fail "external IAM requires iam.auth.issuerInternalUrl" -}}
{{- end -}}
{{- if not .Values.iam.auth.issuerPublicUrl -}}
{{- fail "external IAM requires iam.auth.issuerPublicUrl" -}}
{{- end -}}
{{- if not .Values.iam.auth.clientId -}}
{{- fail "external IAM requires iam.auth.clientId" -}}
{{- end -}}
{{- if not .Values.iam.frontend.oidcAuthority -}}
{{- fail "external IAM requires iam.frontend.oidcAuthority" -}}
{{- end -}}
{{- if and (not .Values.iam.frontend.oidcClientId) (not .Values.iam.frontend.oidcClientIdFile) -}}
{{- fail "external IAM requires iam.frontend.oidcClientId or iam.frontend.oidcClientIdFile" -}}
{{- end -}}
{{- end -}}
{{- if and (eq .Values.iam.auth.tokenMode "introspection") (ne $mode "zitadel") (not .Values.postgresql.existingSecret) (not .Values.iam.auth.introspectionClientSecret) -}}
{{- fail "AUTH_TOKEN_MODE=introspection requires iam.auth.introspectionClientSecret or postgresql.existingSecret containing AUTH_INTROSPECTION_CLIENT_SECRET" -}}
{{- end -}}
{{- end -}}
