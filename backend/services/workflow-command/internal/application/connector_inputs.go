package application

import "strings"

// knownConnectorInputKeys mirrors connectors/internal/common/descriptors.go.
// Duplicated here because Go forbids importing connectors/internal from this package.
var knownConnectorInputKeys = []string{
	"url", "method", "headers", "body", "timeoutMs", "failOnNon2xx",
	"webhookUrl", "payload", "webhookToken",
	"kafkaTopic", "kafkaKey", "kafkaValue",
	"emailTo", "emailSubject", "emailBody",
	"s3Bucket", "s3Key", "s3Body", "contentType",
}

func connectorInputsFromProperties(props map[string]any) map[string]any {
	out := map[string]any{}
	if props == nil {
		return out
	}
	for _, key := range knownConnectorInputKeys {
		v, ok := props[key]
		if !ok || v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		out[key] = v
	}
	return out
}
