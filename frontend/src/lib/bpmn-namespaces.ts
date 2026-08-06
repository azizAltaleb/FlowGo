export const ARTIFICIALFLOW_BPMN_NAMESPACE = "http://artificialflow.io/schema/1.0/bpmn";
export const ARTIFICIALFLOW_BPMN_PREFIX = "artificialflow";

const EXTENSION_ATTRIBUTE_NAMES = [
  "topic",
  "taskType",
  "scriptFormat",
  "resultVariable",
  "timeout",
  "assignee",
  "candidateUsers",
  "candidateGroups",
  "dueDate",
  "correlationKey",
  "decisionRef",
  "timerDuration",
  "errorCode",
  "errorMessage",
  "condition",
  "sourceHandle",
  "targetHandle",
  "collection",
  "elementVariable",
  "calledElementVersion",
] as const;

export type ArtificialFlowAttributeName = (typeof EXTENSION_ATTRIBUTE_NAMES)[number];

type XMLRecord = Record<string, unknown>;

export interface BpmnNamespaceContext {
  canonicalPrefixes: string[];
}

const defaultContext: BpmnNamespaceContext = {
  canonicalPrefixes: [ARTIFICIALFLOW_BPMN_PREFIX],
};

function unique(values: string[]): string[] {
  return [...new Set(values.filter(Boolean))];
}

export function bpmnNamespaceContext(definitions: XMLRecord): BpmnNamespaceContext {
  const prefixesFor = (namespace: string) =>
    Object.entries(definitions)
      .filter(([key, value]) => key.startsWith("@_xmlns:") && value === namespace)
      .map(([key]) => key.slice("@_xmlns:".length));

  return {
    canonicalPrefixes: unique([ARTIFICIALFLOW_BPMN_PREFIX, ...prefixesFor(ARTIFICIALFLOW_BPMN_NAMESPACE)]),
  };
}

function attributeKeys(name: ArtificialFlowAttributeName, context: BpmnNamespaceContext): string[] {
  return [
    ...context.canonicalPrefixes.map((prefix) => `@_${prefix}:${name}`),
    `@_${name}`,
  ];
}

export function getArtificialFlowAttribute(
  data: XMLRecord,
  name: ArtificialFlowAttributeName,
  context: BpmnNamespaceContext = defaultContext,
): unknown {
  for (const key of attributeKeys(name, context)) {
    if (Object.prototype.hasOwnProperty.call(data, key)) return data[key];
  }
  return undefined;
}

export function setArtificialFlowAttribute(
  data: XMLRecord,
  name: ArtificialFlowAttributeName,
  value: unknown,
  context: BpmnNamespaceContext = defaultContext,
): XMLRecord {
  const normalized = { ...data };
  for (const key of attributeKeys(name, context)) delete normalized[key];
  normalized[`@_${ARTIFICIALFLOW_BPMN_PREFIX}:${name}`] = value;
  return normalized;
}

function extensionKeys(
  localName: "properties" | "property" | "connector",
  context: BpmnNamespaceContext,
): string[] {
  return [
    ...context.canonicalPrefixes.map((prefix) => `${prefix}:${localName}`),
    localName,
  ];
}

function asRecord(value: unknown): XMLRecord | undefined {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? value as XMLRecord
    : undefined;
}

function asRecords(value: unknown): XMLRecord[] {
  if (Array.isArray(value)) return value.map(asRecord).filter((item): item is XMLRecord => Boolean(item));
  const record = asRecord(value);
  return record ? [record] : [];
}

function valuesForKeys(record: XMLRecord, keys: string[]): unknown[] {
  return keys
    .filter((key) => Object.prototype.hasOwnProperty.call(record, key))
    .map((key) => record[key]);
}

function canonicalVendorKey(key: string, prefixes: string[]): string | undefined {
  for (const prefix of prefixes) {
    if (key.startsWith(`@_${prefix}:`)) {
      return `@_${ARTIFICIALFLOW_BPMN_PREFIX}:${key.slice(`@_${prefix}:`.length)}`;
    }
    if (key.startsWith(`${prefix}:`)) {
      return `${ARTIFICIALFLOW_BPMN_PREFIX}:${key.slice(`${prefix}:`.length)}`;
    }
  }
  return undefined;
}

function canonicalizeVendorKeys(value: unknown, context: BpmnNamespaceContext): unknown {
  if (Array.isArray(value)) return value.map((item) => canonicalizeVendorKeys(item, context));
  const record = asRecord(value);
  if (!record) return value;

  const normalized: XMLRecord = {};
  const canonicalPrefixes = unique([ARTIFICIALFLOW_BPMN_PREFIX, ...context.canonicalPrefixes]);
  for (const [key, item] of Object.entries(record)) {
    if (!canonicalVendorKey(key, canonicalPrefixes)) {
      normalized[key] = canonicalizeVendorKeys(item, context);
    }
  }
  for (const [key, item] of Object.entries(record)) {
    const canonicalKey = canonicalVendorKey(key, canonicalPrefixes);
    if (canonicalKey) normalized[canonicalKey] = canonicalizeVendorKeys(item, context);
  }
  return normalized;
}

function mergeProperties(extensionElements: XMLRecord, context: BpmnNamespaceContext): XMLRecord[] {
  const groups = valuesForKeys(extensionElements, extensionKeys("properties", context)).flatMap(asRecords);
  const properties = groups.flatMap((group) =>
    valuesForKeys(group, extensionKeys("property", context)).flatMap(asRecords),
  );
  const seen = new Set<string>();
  return properties.filter((property, index) => {
    const name = String(property["@_name"] ?? "").trim().toLowerCase();
    const key = name || `__unnamed_${index}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function normalizeExtensionElements(
  value: unknown,
  context: BpmnNamespaceContext = defaultContext,
): XMLRecord | undefined {
  const extensionElements = asRecord(Array.isArray(value) ? value[0] : value);
  if (!extensionElements) return undefined;

  const normalized = canonicalizeVendorKeys(extensionElements, context) as XMLRecord;
  const propertiesKeys = extensionKeys("properties", context);
  const connectorKeys = extensionKeys("connector", context);
  const properties = mergeProperties(extensionElements, context);
  const connector = valuesForKeys(extensionElements, connectorKeys).flatMap(asRecords)[0];

  for (const key of [...propertiesKeys, ...connectorKeys]) delete normalized[key];
  if (properties.length > 0) {
    normalized[`${ARTIFICIALFLOW_BPMN_PREFIX}:properties`] = {
      [`${ARTIFICIALFLOW_BPMN_PREFIX}:property`]: properties,
    };
  }
  if (connector) {
    normalized[`${ARTIFICIALFLOW_BPMN_PREFIX}:connector`] = canonicalizeVendorKeys(connector, context);
  }
  return normalized;
}

export function normalizeBpmnData(
  data: XMLRecord,
  context: BpmnNamespaceContext = defaultContext,
): XMLRecord {
  const normalized = canonicalizeVendorKeys(data, context) as XMLRecord;
  for (const name of EXTENSION_ATTRIBUTE_NAMES) {
    const value = getArtificialFlowAttribute(data, name, context);
    for (const key of attributeKeys(name, context)) delete normalized[key];
    if (value !== undefined) {
      normalized[`@_${ARTIFICIALFLOW_BPMN_PREFIX}:${name}`] = value;
    }
  }
  if (Object.prototype.hasOwnProperty.call(data, "bpmn:extensionElements")) {
    const extensionElements = normalizeExtensionElements(data["bpmn:extensionElements"], context);
    if (extensionElements && Object.keys(extensionElements).length > 0) {
      normalized["bpmn:extensionElements"] = extensionElements;
    } else {
      delete normalized["bpmn:extensionElements"];
    }
  }
  return normalized;
}

export interface ArtificialFlowProperty {
  "@_name": string;
  "@_value": string;
}

export function getArtificialFlowProperties(data: XMLRecord | null): ArtificialFlowProperty[] {
  const extensionElements = normalizeExtensionElements(data?.["bpmn:extensionElements"]);
  const properties = asRecord(extensionElements?.[`${ARTIFICIALFLOW_BPMN_PREFIX}:properties`]);
  return asRecords(properties?.[`${ARTIFICIALFLOW_BPMN_PREFIX}:property`]).map((property) => ({
    "@_name": String(property["@_name"] ?? ""),
    "@_value": String(property["@_value"] ?? ""),
  }));
}

export function setArtificialFlowProperties(
  data: XMLRecord | null,
  properties: ArtificialFlowProperty[],
): XMLRecord {
  const normalizedData = { ...data };
  const current = normalizeExtensionElements(normalizedData["bpmn:extensionElements"]) || {};
  delete current[`${ARTIFICIALFLOW_BPMN_PREFIX}:properties`];
  if (properties.length > 0) {
    current[`${ARTIFICIALFLOW_BPMN_PREFIX}:properties`] = {
      [`${ARTIFICIALFLOW_BPMN_PREFIX}:property`]: properties,
    };
  }
  if (Object.keys(current).length > 0) {
    normalizedData["bpmn:extensionElements"] = current;
  } else {
    delete normalizedData["bpmn:extensionElements"];
  }
  return normalizedData;
}
