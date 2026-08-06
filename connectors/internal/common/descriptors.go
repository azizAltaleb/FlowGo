package common

import "strings"

// ConnectorField describes a single connector input variable.
type ConnectorField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	// Kind is "string", "number", "boolean", or "json".
	Kind    string `json:"kind"`
	Default string `json:"default,omitempty"`
}

// ConnectorDescriptor documents a shipped connector job type and its inputs.
type ConnectorDescriptor struct {
	JobType     string           `json:"jobType"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Fields      []ConnectorField `json:"fields"`
}

// Descriptors is the registry of official ArtificialFlow connectors.
var Descriptors = []ConnectorDescriptor{
	{
		JobType:     "io.artificialflow.connector.http",
		Name:        "HTTP",
		Description: "Outbound HTTP request",
		Fields: []ConnectorField{
			{Key: "url", Label: "URL", Required: true, Kind: "string", Description: "Absolute HTTPS/HTTP URL"},
			{Key: "method", Label: "Method", Required: false, Kind: "string", Default: "POST", Description: "HTTP method"},
			{Key: "headers", Label: "Headers", Required: false, Kind: "json", Description: "JSON object of headers"},
			{Key: "body", Label: "Body", Required: false, Kind: "json", Description: "JSON-serializable request body"},
			{Key: "timeoutMs", Label: "Timeout (ms)", Required: false, Kind: "number", Default: "10000", Description: "Request timeout in milliseconds"},
			{Key: "failOnNon2xx", Label: "Fail on non-2xx", Required: false, Kind: "boolean", Default: "true", Description: "Fail the job when the response status is not 2xx"},
		},
	},
	{
		JobType:     "io.artificialflow.connector.webhook",
		Name:        "Webhook",
		Description: "POST to a webhook URL (e.g. Slack)",
		Fields: []ConnectorField{
			{Key: "webhookUrl", Label: "Webhook URL", Required: true, Kind: "string", Description: "Absolute webhook URL"},
			{Key: "payload", Label: "Payload", Required: false, Kind: "json", Description: "JSON payload body"},
			{Key: "webhookToken", Label: "Webhook Token", Required: false, Kind: "string", Description: "Optional Bearer token"},
		},
	},
	{
		JobType:     "io.artificialflow.connector.kafka",
		Name:        "Kafka",
		Description: "Publish a Kafka message",
		Fields: []ConnectorField{
			{Key: "kafkaTopic", Label: "Topic", Required: true, Kind: "string", Description: "Kafka topic"},
			{Key: "kafkaKey", Label: "Key", Required: false, Kind: "string", Description: "Optional message key"},
			{Key: "kafkaValue", Label: "Value", Required: false, Kind: "json", Description: "Message value (JSON or string)"},
		},
	},
	{
		JobType:     "io.artificialflow.connector.email",
		Name:        "Email",
		Description: "Send email via SMTP",
		Fields: []ConnectorField{
			{Key: "emailTo", Label: "To", Required: true, Kind: "string", Description: "Recipient address"},
			{Key: "emailSubject", Label: "Subject", Required: true, Kind: "string", Description: "Email subject"},
			{Key: "emailBody", Label: "Body", Required: false, Kind: "string", Description: "Email body"},
		},
	},
	{
		JobType:     "io.artificialflow.connector.s3",
		Name:        "S3",
		Description: "Put object to S3-compatible storage",
		Fields: []ConnectorField{
			{Key: "s3Bucket", Label: "Bucket", Required: true, Kind: "string", Description: "S3 bucket name"},
			{Key: "s3Key", Label: "Key", Required: true, Kind: "string", Description: "Object key"},
			{Key: "s3Body", Label: "Body", Required: false, Kind: "string", Description: "Object body"},
			{Key: "contentType", Label: "Content-Type", Required: false, Kind: "string", Description: "MIME type"},
		},
	},
}

// DescriptorByJobType returns the descriptor for a job type, if known.
func DescriptorByJobType(jobType string) (ConnectorDescriptor, bool) {
	for _, d := range Descriptors {
		if d.JobType == jobType {
			return d, true
		}
	}
	return ConnectorDescriptor{}, false
}

// AllConnectorInputKeys returns every known connector input variable key.
func AllConnectorInputKeys() []string {
	seen := map[string]struct{}{}
	var keys []string
	for _, d := range Descriptors {
		for _, f := range d.Fields {
			if _, ok := seen[f.Key]; ok {
				continue
			}
			seen[f.Key] = struct{}{}
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// IsConnectorInputKey reports whether key is a known connector input variable.
func IsConnectorInputKey(key string) bool {
	for _, k := range AllConnectorInputKeys() {
		if k == key {
			return true
		}
	}
	return false
}

// ConnectorInputsFromProperties extracts known connector keys from step properties.
// Empty string values are skipped.
func ConnectorInputsFromProperties(props map[string]any) map[string]any {
	out := map[string]any{}
	if props == nil {
		return out
	}
	for _, key := range AllConnectorInputKeys() {
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
