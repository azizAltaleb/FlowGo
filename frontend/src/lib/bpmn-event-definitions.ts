/**
 * Helpers for reading/writing BPMN event definitions on modeler node data.
 * Keeps PropertiesPanel and generateBpmnXml aligned.
 */

export type EventDefinitionKind =
  | "none"
  | "timer"
  | "message"
  | "signal"
  | "escalation"
  | "link"
  | "conditional"
  | "terminate"
  | "error"
  | "compensate"
  | "cancel";

type XMLRecord = Record<string, unknown>;

function textValue(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "object" && value !== null && "#text" in value) {
    return String((value as XMLRecord)["#text"] ?? "");
  }
  return "";
}

function firstPresent(data: XMLRecord, keys: string[]): unknown {
  for (const key of keys) {
    if (Object.prototype.hasOwnProperty.call(data, key) && data[key] != null) {
      return data[key];
    }
  }
  return undefined;
}

function asRecord(value: unknown): XMLRecord | null {
  return typeof value === "object" && value !== null ? (value as XMLRecord) : null;
}

function sanitizeIdPart(value: string): string {
  const cleaned = value.trim().replace(/[^A-Za-z0-9_.-]+/g, "_");
  if (!cleaned) return "Item";
  return /^[A-Za-z_]/.test(cleaned) ? cleaned : `_${cleaned}`;
}

export function inferEventDefinitionKind(data: XMLRecord): EventDefinitionKind {
  const marked = String(data.eventDefinitionType || "").trim().toLowerCase();
  if (
    marked === "timer" ||
    marked === "message" ||
    marked === "signal" ||
    marked === "escalation" ||
    marked === "link" ||
    marked === "conditional" ||
    marked === "terminate" ||
    marked === "error" ||
    marked === "compensate" ||
    marked === "cancel"
  ) {
    return marked as EventDefinitionKind;
  }

  if (firstPresent(data, ["bpmn:timerEventDefinition", "timerEventDefinition"])) return "timer";
  if (firstPresent(data, ["bpmn:messageEventDefinition", "messageEventDefinition"])) return "message";
  if (firstPresent(data, ["bpmn:signalEventDefinition", "signalEventDefinition"])) return "signal";
  if (firstPresent(data, ["bpmn:escalationEventDefinition", "escalationEventDefinition"])) return "escalation";
  if (firstPresent(data, ["bpmn:linkEventDefinition", "linkEventDefinition"])) return "link";
  if (firstPresent(data, ["bpmn:conditionalEventDefinition", "conditionalEventDefinition"])) return "conditional";
  if (firstPresent(data, ["bpmn:terminateEventDefinition", "terminateEventDefinition"])) return "terminate";
  if (firstPresent(data, ["bpmn:errorEventDefinition", "errorEventDefinition"])) return "error";
  if (firstPresent(data, ["bpmn:compensateEventDefinition", "compensateEventDefinition"])) return "compensate";
  if (firstPresent(data, ["bpmn:cancelEventDefinition", "cancelEventDefinition"])) return "cancel";
  return "none";
}

export type TimerKind = "duration" | "date" | "cycle";

export function getTimerKind(data: XMLRecord): TimerKind {
  const def = asRecord(firstPresent(data, ["bpmn:timerEventDefinition", "timerEventDefinition"]));
  if (!def) return "duration";
  if (textValue(def["bpmn:timeCycle"] ?? def["timeCycle"])) return "cycle";
  if (textValue(def["bpmn:timeDate"] ?? def["timeDate"])) return "date";
  return "duration";
}

export function getTimerDuration(data: XMLRecord): string {
  const def = asRecord(firstPresent(data, ["bpmn:timerEventDefinition", "timerEventDefinition"]));
  if (!def) return "";
  return textValue(def["bpmn:timeDuration"] ?? def["timeDuration"]);
}

export function getTimerDate(data: XMLRecord): string {
  const def = asRecord(firstPresent(data, ["bpmn:timerEventDefinition", "timerEventDefinition"]));
  if (!def) return "";
  return textValue(def["bpmn:timeDate"] ?? def["timeDate"]);
}

export function getTimerCycle(data: XMLRecord): string {
  const def = asRecord(firstPresent(data, ["bpmn:timerEventDefinition", "timerEventDefinition"]));
  if (!def) return "";
  return textValue(def["bpmn:timeCycle"] ?? def["timeCycle"]);
}

export function withTimerDuration(data: XMLRecord, duration: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "timer" };
  const trimmed = duration.trim();
  delete next["timerEventDefinition"];
  if (!trimmed) {
    next["bpmn:timerEventDefinition"] = { "bpmn:timeDuration": "PT1H" };
    return next;
  }
  next["bpmn:timerEventDefinition"] = { "bpmn:timeDuration": trimmed };
  return next;
}

export function withTimerDate(data: XMLRecord, date: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "timer" };
  const trimmed = date.trim();
  delete next["timerEventDefinition"];
  next["bpmn:timerEventDefinition"] = {
    "bpmn:timeDate": trimmed || new Date().toISOString(),
  };
  return next;
}

export function withTimerCycle(data: XMLRecord, cycle: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "timer" };
  const trimmed = cycle.trim();
  delete next["timerEventDefinition"];
  next["bpmn:timerEventDefinition"] = {
    "bpmn:timeCycle": trimmed || "R3/PT1H",
  };
  return next;
}

export function withTimerKind(data: XMLRecord, kind: TimerKind): XMLRecord {
  if (kind === "date") {
    return withTimerDate(data, getTimerDate(data) || new Date().toISOString());
  }
  if (kind === "cycle") {
    return withTimerCycle(data, getTimerCycle(data) || "R3/PT1H");
  }
  return withTimerDuration(data, getTimerDuration(data) || "PT1H");
}

function getRefFromDefinition(data: XMLRecord, defKeys: string[], attrKeys: string[]): string {
  const def = asRecord(firstPresent(data, defKeys));
  if (!def) return "";
  for (const key of attrKeys) {
    const value = def[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return "";
}

export function getMessageRef(data: XMLRecord): string {
  return getRefFromDefinition(
    data,
    ["bpmn:messageEventDefinition", "messageEventDefinition"],
    ["@_messageRef", "messageRef", "@_name"],
  );
}

export function withMessageRef(data: XMLRecord, messageName: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "message" };
  const name = messageName.trim() || "Message";
  const id = `Message_${sanitizeIdPart(name)}`;
  delete next["messageEventDefinition"];
  next["bpmn:messageEventDefinition"] = { "@_messageRef": id };
  next.message_name = name;
  next.message_id = id;
  return next;
}

export function getSignalRef(data: XMLRecord): string {
  return getRefFromDefinition(
    data,
    ["bpmn:signalEventDefinition", "signalEventDefinition"],
    ["@_signalRef", "signalRef", "@_name"],
  );
}

export function withSignalRef(data: XMLRecord, signalName: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "signal" };
  const name = signalName.trim() || "Signal";
  const id = `Signal_${sanitizeIdPart(name)}`;
  delete next["signalEventDefinition"];
  next["bpmn:signalEventDefinition"] = { "@_signalRef": id };
  next.signal_name = name;
  next.signal_id = id;
  return next;
}

export function getEscalationCode(data: XMLRecord): string {
  if (typeof data.escalation_code === "string" && data.escalation_code.trim()) {
    return data.escalation_code.trim();
  }
  const fromAttr =
    data["@_artificialflow:escalationCode"] ??
    data["@_escalationCode"] ??
    data.escalationCode;
  if (typeof fromAttr === "string" && fromAttr.trim()) return fromAttr.trim();
  return getRefFromDefinition(
    data,
    ["bpmn:escalationEventDefinition", "escalationEventDefinition"],
    ["@_escalationRef", "escalationRef"],
  );
}

export function withEscalationCode(data: XMLRecord, code: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "escalation" };
  const value = code.trim() || "ESC_1";
  const id = `Escalation_${sanitizeIdPart(value)}`;
  next.escalation_code = value;
  next.escalation_id = id;
  next["@_artificialflow:escalationCode"] = value;
  delete next["escalationEventDefinition"];
  next["bpmn:escalationEventDefinition"] = { "@_escalationRef": id };
  return next;
}

export function getLinkName(data: XMLRecord): string {
  if (typeof data.link_name === "string" && data.link_name.trim()) {
    return data.link_name.trim();
  }
  const def = asRecord(firstPresent(data, ["bpmn:linkEventDefinition", "linkEventDefinition"]));
  if (!def) return "";
  const attr = def["@_name"];
  if (typeof attr === "string") return attr.trim();
  return textValue(def["bpmn:name"] ?? def["name"]);
}

export function withLinkName(data: XMLRecord, linkName: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "link" };
  const value = linkName.trim() || "Link_1";
  next.link_name = value;
  delete next["linkEventDefinition"];
  next["bpmn:linkEventDefinition"] = { "@_name": value };
  return next;
}

export function getConditionalExpression(data: XMLRecord): string {
  if (typeof data.condition === "string" && data.condition.trim()) {
    return data.condition.trim();
  }
  const def = asRecord(firstPresent(data, ["bpmn:conditionalEventDefinition", "conditionalEventDefinition"]));
  if (!def) return "";
  const condition = def["bpmn:condition"] ?? def["condition"];
  return textValue(condition);
}

export function withConditionalExpression(data: XMLRecord, expression: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "conditional" };
  const value = expression.trim() || "true";
  next.condition = value;
  delete next["conditionalEventDefinition"];
  next["bpmn:conditionalEventDefinition"] = {
    "bpmn:condition": { "#text": value },
  };
  return next;
}

export function getErrorCode(data: XMLRecord): string {
  if (typeof data.error_code === "string" && data.error_code.trim()) return data.error_code.trim();
  const fromAttr = data["@_artificialflow:errorCode"] ?? data["@_errorCode"];
  if (typeof fromAttr === "string" && fromAttr.trim()) return fromAttr.trim();
  return getRefFromDefinition(
    data,
    ["bpmn:errorEventDefinition", "errorEventDefinition"],
    ["@_errorRef", "errorRef"],
  );
}

export function withErrorCode(data: XMLRecord, code: string): XMLRecord {
  const next: XMLRecord = { ...data, eventDefinitionType: "error" };
  const value = code.trim() || "ERROR_1";
  const id = `Error_${sanitizeIdPart(value)}`;
  next.error_code = value;
  next.error_id = id;
  next["@_artificialflow:errorCode"] = value;
  delete next["errorEventDefinition"];
  next["bpmn:errorEventDefinition"] = { "@_errorRef": id };
  return next;
}

export function displayMessageName(data: XMLRecord): string {
  if (typeof data.message_name === "string" && data.message_name.trim()) {
    return data.message_name.trim();
  }
  const ref = getMessageRef(data);
  return ref.replace(/^Message_/, "").replace(/_/g, " ") || ref;
}

export function displaySignalName(data: XMLRecord): string {
  if (typeof data.signal_name === "string" && data.signal_name.trim()) {
    return data.signal_name.trim();
  }
  const ref = getSignalRef(data);
  return ref.replace(/^Signal_/, "").replace(/_/g, " ") || ref;
}

/** Collect root-level message/signal/escalation/error elements needed by event refs on nodes. */
export function collectRootEventCatalog(nodes: Array<{ data?: XMLRecord }>): {
  messages: Array<{ id: string; name: string }>;
  signals: Array<{ id: string; name: string }>;
  escalations: Array<{ id: string; name: string }>;
  errors: Array<{ id: string; errorCode: string }>;
} {
  const messages = new Map<string, string>();
  const signals = new Map<string, string>();
  const escalations = new Map<string, string>();
  const errors = new Map<string, string>();

  for (const node of nodes) {
    const data = node.data || {};
    const kind = inferEventDefinitionKind(data);
    if (kind === "message") {
      const id =
        (typeof data.message_id === "string" && data.message_id) ||
        getMessageRef(data) ||
        `Message_${sanitizeIdPart(displayMessageName(data) || "Message")}`;
      const name = displayMessageName(data) || id;
      messages.set(id, name);
    }
    if (kind === "signal") {
      const id =
        (typeof data.signal_id === "string" && data.signal_id) ||
        getSignalRef(data) ||
        `Signal_${sanitizeIdPart(displaySignalName(data) || "Signal")}`;
      const name = displaySignalName(data) || id;
      signals.set(id, name);
    }
    if (kind === "escalation") {
      const code = getEscalationCode(data) || "ESC_1";
      const id =
        (typeof data.escalation_id === "string" && data.escalation_id) ||
        getRefFromDefinition(
          data,
          ["bpmn:escalationEventDefinition", "escalationEventDefinition"],
          ["@_escalationRef", "escalationRef"],
        ) ||
        `Escalation_${sanitizeIdPart(code)}`;
      escalations.set(id, code);
    }
    if (kind === "error") {
      const code = getErrorCode(data) || "ERROR_1";
      const id =
        (typeof data.error_id === "string" && data.error_id) ||
        getRefFromDefinition(
          data,
          ["bpmn:errorEventDefinition", "errorEventDefinition"],
          ["@_errorRef", "errorRef"],
        ) ||
        `Error_${sanitizeIdPart(code)}`;
      errors.set(id, code);
    }
    // Receive tasks use @_messageRef
    const recvRef = data["@_messageRef"];
    if (typeof recvRef === "string" && recvRef.trim()) {
      messages.set(recvRef.trim(), recvRef.trim().replace(/^Message_/, ""));
    }
  }

  return {
    messages: [...messages.entries()].map(([id, name]) => ({ id, name })),
    signals: [...signals.entries()].map(([id, name]) => ({ id, name })),
    escalations: [...escalations.entries()].map(([id, name]) => ({ id, name })),
    errors: [...errors.entries()].map(([id, errorCode]) => ({ id, errorCode })),
  };
}
