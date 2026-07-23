{{- define "artificialflow.name" -}}
{{- if .Values.compatibility.legacyResourceNames.enabled -}}
{{- required "compatibility.legacyResourceNames.selectorNameOverride is required when legacy resource names are enabled" .Values.compatibility.legacyResourceNames.selectorNameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.fullname" -}}
{{- if .Values.compatibility.legacyResourceNames.enabled -}}
{{- required "compatibility.legacyResourceNames.fullnameOverride is required when legacy resource names are enabled" .Values.compatibility.legacyResourceNames.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else if .Values.fullnameOverride -}}
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

{{- define "artificialflow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "artificialflow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "artificialflow.name" . }}
{{ if .Values.compatibility.legacyResourceNames.enabled }}
app.kubernetes.io/instance: {{ required "compatibility.legacyResourceNames.selectorInstanceOverride is required when legacy resource names are enabled" .Values.compatibility.legacyResourceNames.selectorInstanceOverride }}
{{ else }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{ end }}
{{- end -}}

{{- define "artificialflow.labels" -}}
helm.sh/chart: {{ include "artificialflow.chart" . }}
{{ include "artificialflow.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{- define "artificialflow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "artificialflow.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.envConfigName" -}}
{{- printf "%s-env" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.envSecretName" -}}
{{- if .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecret -}}
{{- else -}}
{{- printf "%s-env" (include "artificialflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.zitadelSecretName" -}}
{{- if .Values.zitadel.existingSecret -}}
{{- .Values.zitadel.existingSecret -}}
{{- else -}}
{{- printf "%s-zitadel" (include "artificialflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.zitadelApiUrl" -}}
{{- printf "http://%s-zitadel-api:8080" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.zitadelBootstrapPvc" -}}
{{- if .Values.zitadel.bootstrapStorage.existingClaim -}}
{{- .Values.zitadel.bootstrapStorage.existingClaim -}}
{{- else -}}
{{- printf "%s-zitadel-bootstrap" (include "artificialflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.bootstrapPvc" -}}
{{- if .Values.applicationState.bootstrapExistingClaim -}}
{{- .Values.applicationState.bootstrapExistingClaim -}}
{{- else if .Values.compatibility.legacyResourceNames.enabled -}}
{{- printf "%s-flowgo-bootstrap" (include "artificialflow.fullname" .) -}}
{{- else -}}
{{- printf "%s-artificialflow-bootstrap" (include "artificialflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.authPvc" -}}
{{- if .Values.applicationState.authExistingClaim -}}
{{- .Values.applicationState.authExistingClaim -}}
{{- else if .Values.compatibility.legacyResourceNames.enabled -}}
{{- printf "%s-flowgo-auth" (include "artificialflow.fullname" .) -}}
{{- else -}}
{{- printf "%s-artificialflow-auth" (include "artificialflow.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "artificialflow.applicationBootstrapPath" -}}
{{- printf "%s/bootstrap" (trimSuffix "/" .Values.applicationState.rootPath) -}}
{{- end -}}

{{- define "artificialflow.applicationAuthPath" -}}
{{- printf "%s/auth" (trimSuffix "/" .Values.applicationState.rootPath) -}}
{{- end -}}

{{- define "artificialflow.applicationStateFile" -}}
{{- printf "%s/bootstrap/%s-zitadel.json" (trimSuffix "/" .Values.applicationState.rootPath) .Values.applicationState.filePrefix -}}
{{- end -}}

{{- define "artificialflow.commandServiceName" -}}
{{- printf "%s-command" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.queryServiceName" -}}
{{- printf "%s-query" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.frontendServiceName" -}}
{{- printf "%s-frontend" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.gatewayServiceName" -}}
{{- printf "%s-gateway" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.syncWorkerServiceName" -}}
{{- printf "%s-sync-worker" (include "artificialflow.fullname" .) -}}
{{- end -}}

{{- define "artificialflow.validateValues" -}}
{{- if .Values.compatibility.legacyResourceNames.enabled -}}
{{- $_ := required "compatibility.legacyResourceNames.fullnameOverride is required when legacy resource names are enabled" .Values.compatibility.legacyResourceNames.fullnameOverride -}}
{{- $_ = required "compatibility.legacyResourceNames.selectorNameOverride is required when legacy resource names are enabled" .Values.compatibility.legacyResourceNames.selectorNameOverride -}}
{{- $_ = required "compatibility.legacyResourceNames.selectorInstanceOverride is required when legacy resource names are enabled" .Values.compatibility.legacyResourceNames.selectorInstanceOverride -}}
{{- end -}}
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
