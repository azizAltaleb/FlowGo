export type ConnectorFieldKind = "string" | "number" | "boolean" | "json";

export interface ConnectorField {
  key: string;
  label: string;
  required: boolean;
  description: string;
  kind: ConnectorFieldKind;
  default?: string;
}

export interface ConnectorDescriptor {
  jobType: string;
  name: string;
  description: string;
  fields: ConnectorField[];
}

/** Official ArtificialFlow connector job types and input schemas. */
export const CONNECTOR_DESCRIPTORS: ConnectorDescriptor[] = [
  {
    jobType: "io.artificialflow.connector.http",
    name: "HTTP",
    description: "Outbound HTTP request",
    fields: [
      { key: "url", label: "URL", required: true, kind: "string", description: "Absolute HTTPS/HTTP URL" },
      { key: "method", label: "Method", required: false, kind: "string", default: "POST", description: "HTTP method" },
      { key: "headers", label: "Headers", required: false, kind: "json", description: "JSON object of headers" },
      { key: "body", label: "Body", required: false, kind: "json", description: "JSON-serializable request body" },
      { key: "timeoutMs", label: "Timeout (ms)", required: false, kind: "number", default: "10000", description: "Request timeout in milliseconds" },
      { key: "failOnNon2xx", label: "Fail on non-2xx", required: false, kind: "boolean", default: "true", description: "Fail the job when the response status is not 2xx" },
    ],
  },
  {
    jobType: "io.artificialflow.connector.webhook",
    name: "Webhook",
    description: "POST to a webhook URL (e.g. Slack)",
    fields: [
      { key: "webhookUrl", label: "Webhook URL", required: true, kind: "string", description: "Absolute webhook URL" },
      { key: "payload", label: "Payload", required: false, kind: "json", description: "JSON payload body" },
      { key: "webhookToken", label: "Webhook Token", required: false, kind: "string", description: "Optional Bearer token" },
    ],
  },
  {
    jobType: "io.artificialflow.connector.kafka",
    name: "Kafka",
    description: "Publish a Kafka message",
    fields: [
      { key: "kafkaTopic", label: "Topic", required: true, kind: "string", description: "Kafka topic" },
      { key: "kafkaKey", label: "Key", required: false, kind: "string", description: "Optional message key" },
      { key: "kafkaValue", label: "Value", required: false, kind: "json", description: "Message value (JSON or string)" },
    ],
  },
  {
    jobType: "io.artificialflow.connector.email",
    name: "Email",
    description: "Send email via SMTP",
    fields: [
      { key: "emailTo", label: "To", required: true, kind: "string", description: "Recipient address" },
      { key: "emailSubject", label: "Subject", required: true, kind: "string", description: "Email subject" },
      { key: "emailBody", label: "Body", required: false, kind: "string", description: "Email body" },
    ],
  },
  {
    jobType: "io.artificialflow.connector.s3",
    name: "S3",
    description: "Put object to S3-compatible storage",
    fields: [
      { key: "s3Bucket", label: "Bucket", required: true, kind: "string", description: "S3 bucket name" },
      { key: "s3Key", label: "Key", required: true, kind: "string", description: "Object key" },
      { key: "s3Body", label: "Body", required: false, kind: "string", description: "Object body" },
      { key: "contentType", label: "Content-Type", required: false, kind: "string", description: "MIME type" },
    ],
  },
];

export function getConnectorDescriptor(jobType: string): ConnectorDescriptor | undefined {
  return CONNECTOR_DESCRIPTORS.find((d) => d.jobType === jobType);
}

export function allConnectorInputKeys(): string[] {
  const seen = new Set<string>();
  const keys: string[] = [];
  for (const d of CONNECTOR_DESCRIPTORS) {
    for (const f of d.fields) {
      if (seen.has(f.key)) continue;
      seen.add(f.key);
      keys.push(f.key);
    }
  }
  return keys;
}

/** Read connector inputs from node data (connectorInputs) or extension properties. */
export function readConnectorInputs(
  data: Record<string, unknown> | null | undefined,
  jobType: string,
): Record<string, string> {
  const descriptor = getConnectorDescriptor(jobType);
  if (!descriptor) return {};

  const fromNode = (data?.connectorInputs as Record<string, unknown> | undefined) || {};
  const props = getExtensionPropertyMap(data);
  const out: Record<string, string> = {};
  for (const field of descriptor.fields) {
    const raw = fromNode[field.key] ?? props[field.key] ?? field.default ?? "";
    out[field.key] = raw == null ? "" : String(raw);
  }
  return out;
}

function getExtensionPropertyMap(data: Record<string, unknown> | null | undefined): Record<string, string> {
  const ext = data?.["bpmn:extensionElements"] as Record<string, unknown> | undefined;
  const properties = ext?.["artificialflow:properties"] as Record<string, unknown> | undefined;
  const property = properties?.["artificialflow:property"];
  const list = Array.isArray(property) ? property : property ? [property] : [];
  const out: Record<string, string> = {};
  for (const item of list) {
    if (!item || typeof item !== "object") continue;
    const rec = item as Record<string, unknown>;
    const name = String(rec["@_name"] ?? "").trim();
    if (!name) continue;
    out[name] = String(rec["@_value"] ?? "");
  }
  return out;
}

/**
 * Persist connector field values under node `connectorInputs` and as
 * `artificialflow:property` name/value pairs (so deploy/parser merges them onto step.Properties).
 * Start-instance variables with the same keys can still supply or override at runtime.
 */
export function applyConnectorInputs(
  data: Record<string, unknown>,
  jobType: string,
  inputs: Record<string, string>,
): Record<string, unknown> {
  const descriptor = getConnectorDescriptor(jobType);
  const keys = new Set(descriptor?.fields.map((f) => f.key) ?? Object.keys(inputs));
  const cleaned: Record<string, string> = {};
  for (const [k, v] of Object.entries(inputs)) {
    if (!keys.has(k)) continue;
    cleaned[k] = v;
  }

  const next: Record<string, unknown> = {
    ...data,
    connectorInputs: cleaned,
  };

  const ext = (next["bpmn:extensionElements"] as Record<string, unknown> | undefined) || {};
  const properties = (ext["artificialflow:properties"] as Record<string, unknown> | undefined) || {};
  const existing = properties["artificialflow:property"];
  const list = Array.isArray(existing) ? [...existing] : existing ? [existing] : [];

  const kept = list.filter((item) => {
    if (!item || typeof item !== "object") return true;
    const name = String((item as Record<string, unknown>)["@_name"] ?? "");
    return !keys.has(name);
  });

  for (const [name, value] of Object.entries(cleaned)) {
    if (!value.trim()) continue;
    kept.push({ "@_name": name, "@_value": value });
  }

  const nextExt = { ...ext };
  if (kept.length > 0) {
    nextExt["artificialflow:properties"] = { "artificialflow:property": kept };
    next["bpmn:extensionElements"] = nextExt;
  } else {
    delete nextExt["artificialflow:properties"];
    if (Object.keys(nextExt).length > 0) {
      next["bpmn:extensionElements"] = nextExt;
    } else {
      delete next["bpmn:extensionElements"];
    }
  }

  return next;
}
