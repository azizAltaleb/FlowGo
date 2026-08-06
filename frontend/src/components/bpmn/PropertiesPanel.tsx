import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { type Node, type Edge } from '@xyflow/react';
import ExtensionProps from "./ExtensionProps";
import { DebouncedInput } from "@/components/ui/debounced-input";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  type ArtificialFlowAttributeName,
  getArtificialFlowAttribute,
  setArtificialFlowAttribute,
} from "@/lib/bpmn-namespaces";
import {
  displayMessageName,
  displaySignalName,
  getConditionalExpression,
  getErrorCode,
  getEscalationCode,
  getLinkName,
  getTimerCycle,
  getTimerDate,
  getTimerDuration,
  getTimerKind,
  inferEventDefinitionKind,
  withConditionalExpression,
  withErrorCode,
  withEscalationCode,
  withLinkName,
  withMessageRef,
  withSignalRef,
  withTimerCycle,
  withTimerDate,
  withTimerDuration,
  withTimerKind,
} from "@/lib/bpmn-event-definitions";
import {
  CONNECTOR_DESCRIPTORS,
  applyConnectorInputs,
  getConnectorDescriptor,
  readConnectorInputs,
} from "@/lib/connector-descriptors";

interface PropertiesPanelProps {
  element: Node | Edge | null;
  onUpdate: (id: string, newData: Record<string, unknown>) => void;
  onRenameId?: (oldId: string, newId: string) => string | null;
  existingIds?: string[];
  nodes?: Node[];
  edges?: Edge[];
}

const NCNAME_RE = /^[A-Za-z_][\w.-]*$/;

const MULTI_INSTANCE_TYPES = new Set([
  "bpmn:userTask",
  "bpmn:serviceTask",
  "bpmn:scriptTask",
  "bpmn:manualTask",
  "bpmn:receiveTask",
  "bpmn:businessRuleTask",
  "bpmn:callActivity",
  "bpmn:subProcess",
  "bpmn:transaction",
  "bpmn:sendTask",
]);

function scriptBodyFromData(data: Record<string, unknown>): string {
  const raw = data["bpmn:script"];
  if (typeof raw === "string") return raw;
  if (typeof raw === "object" && raw !== null && "#text" in raw) {
    return String((raw as Record<string, unknown>)["#text"] ?? "");
  }
  return "";
}

function DebouncedTextarea({
  value: initialValue,
  onValueChange,
  debounce = 300,
  className,
  placeholder,
  rows = 5,
}: {
  value: string;
  onValueChange: (value: string) => void;
  debounce?: number;
  className?: string;
  placeholder?: string;
  rows?: number;
}) {
  const [value, setValue] = React.useState(initialValue);
  const isLocalUpdate = React.useRef(false);

  React.useEffect(() => {
    if (!isLocalUpdate.current) {
      setValue(initialValue);
    }
  }, [initialValue]);

  React.useEffect(() => {
    if (!isLocalUpdate.current) return;
    const timeout = setTimeout(() => {
      onValueChange(value);
      isLocalUpdate.current = false;
    }, debounce);
    return () => clearTimeout(timeout);
  }, [value, debounce, onValueChange]);

  return (
    <Textarea
      value={value}
      rows={rows}
      placeholder={placeholder}
      className={className}
      onChange={(e) => {
        isLocalUpdate.current = true;
        setValue(e.target.value);
      }}
    />
  );
}

export default function PropertiesPanel({
  element,
  onUpdate,
  onRenameId,
  existingIds = [],
  nodes = [],
  edges = [],
}: PropertiesPanelProps) {
  const [idError, setIdError] = React.useState<string | null>(null);

  const updateField = (key: string, value: string) => {
    if (!element) return;

    const newData = { ...element.data } as Record<string, unknown>;

    if (key === "label") {
      newData.label = value;
      newData["@_name"] = value;
    } else if (key.startsWith("@_")) {
      newData[key] = value;
    } else if (key === "bpmn:conditionExpression") {
      if (value) {
        newData["bpmn:conditionExpression"] = {
          "#text": value,
          "@_xsi:type": "bpmn:tFormalExpression",
        };
      } else {
        delete newData["bpmn:conditionExpression"];
      }
    } else if (key === "timerKind") {
      onUpdate(element.id, withTimerKind(newData, value as "duration" | "date" | "cycle"));
      return;
    } else if (key === "timerDuration") {
      onUpdate(element.id, withTimerDuration(newData, value));
      return;
    } else if (key === "timerDate") {
      onUpdate(element.id, withTimerDate(newData, value));
      return;
    } else if (key === "timerCycle") {
      onUpdate(element.id, withTimerCycle(newData, value));
      return;
    } else if (key === "messageName") {
      onUpdate(element.id, withMessageRef(newData, value));
      return;
    } else if (key === "signalName") {
      onUpdate(element.id, withSignalRef(newData, value));
      return;
    } else if (key === "escalationCode") {
      onUpdate(element.id, withEscalationCode(newData, value));
      return;
    } else if (key === "linkName") {
      onUpdate(element.id, withLinkName(newData, value));
      return;
    } else if (key === "conditionalExpression") {
      onUpdate(element.id, withConditionalExpression(newData, value));
      return;
    } else if (key === "errorCode") {
      onUpdate(element.id, withErrorCode(newData, value));
      return;
    } else if (key === "cancelActivity") {
      newData["@_cancelActivity"] = value === "true" ? "true" : "false";
    } else if (key === "triggeredByEvent") {
      newData["@_triggeredByEvent"] = value === "true" ? "true" : "false";
    } else if (key === "receiveMessageRef") {
      newData["@_messageRef"] = value.trim();
    } else if (key === "attachedToRef") {
      newData["@_attachedToRef"] = value.trim();
    } else if (key === "activityRef") {
      newData.activity_ref = value.trim();
      const existing = newData["bpmn:compensateEventDefinition"];
      const def =
        typeof existing === "object" && existing !== null
          ? { ...(existing as Record<string, unknown>) }
          : {};
      def["@_activityRef"] = value.trim();
      newData["bpmn:compensateEventDefinition"] = def;
      newData.eventDefinitionType = "compensate";
    } else if (key === "bpmn:script") {
      newData["bpmn:script"] = value;
    } else if (key === "miSequential") {
      const existing = newData["bpmn:multiInstanceLoopCharacteristics"];
      const def =
        typeof existing === "object" && existing !== null
          ? { ...(existing as Record<string, unknown>) }
          : {};
      def["@_isSequential"] = value === "true" ? "true" : "false";
      newData["bpmn:multiInstanceLoopCharacteristics"] = def;
    } else if (key === "miCollection") {
      if (!newData["bpmn:multiInstanceLoopCharacteristics"]) {
        newData["bpmn:multiInstanceLoopCharacteristics"] = { "@_isSequential": "false" };
      }
      onUpdate(element.id, setArtificialFlowAttribute(newData, "collection", value));
      return;
    } else if (key === "miElementVariable") {
      if (!newData["bpmn:multiInstanceLoopCharacteristics"]) {
        newData["bpmn:multiInstanceLoopCharacteristics"] = { "@_isSequential": "false" };
      }
      onUpdate(element.id, setArtificialFlowAttribute(newData, "elementVariable", value));
      return;
    } else {
      newData[key] = value;
    }

    onUpdate(element.id, newData);
  };

  const updateExtensionField = (key: ArtificialFlowAttributeName, value: string) => {
    if (!element) return;
    onUpdate(element.id, setArtificialFlowAttribute(element.data || {}, key, value));
  };

  if (!element) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center mt-10">
        <p>Select an element to edit properties.</p>
      </div>
    );
  }

  const data = (element.data || {}) as Record<string, unknown>;
  const elementId = element.id;
  const elementName = (data.label as string) || "";

  const assignee = (getArtificialFlowAttribute(data, "assignee") as string) || "";
  const candidateUsers = (getArtificialFlowAttribute(data, "candidateUsers") as string) || "";
  const candidateGroups = (getArtificialFlowAttribute(data, "candidateGroups") as string) || "";
  const dueDate = (getArtificialFlowAttribute(data, "dueDate") as string) || "";
  const decisionRef = (getArtificialFlowAttribute(data, "decisionRef") as string) || "";
  const brResultVariable = (getArtificialFlowAttribute(data, "resultVariable") as string) || "";
  const topic = (getArtificialFlowAttribute(data, "topic") as string) || "";
  const taskType = (getArtificialFlowAttribute(data, "taskType") as string) || "";
  const scriptFormat = (getArtificialFlowAttribute(data, "scriptFormat") as string) || "";
  const scriptResultVariable = (getArtificialFlowAttribute(data, "resultVariable") as string) || "";
  const scriptTimeout = (getArtificialFlowAttribute(data, "timeout") as string) || "";
  const scriptBody = scriptBodyFromData(data);
  const correlationKey = (getArtificialFlowAttribute(data, "correlationKey") as string) || "";
  const errorMessage = (getArtificialFlowAttribute(data, "errorMessage") as string) || "";
  const miCollection = (getArtificialFlowAttribute(data, "collection") as string) || "";
  const miElementVariable = (getArtificialFlowAttribute(data, "elementVariable") as string) || "";
  const calledElementVersion =
    (getArtificialFlowAttribute(data, "calledElementVersion") as string) ||
    (data["@_artificialflow:calledElementVersion"] as string) ||
    "";

  let condition = "";
  const cond = data["bpmn:conditionExpression"];
  if (typeof cond === "object" && cond !== null) {
    condition = ((cond as Record<string, unknown>)["#text"] as string) || "";
  } else {
    condition = (cond as string) || "";
  }

  const calledElement = (data["@_calledElement"] as string) || "";
  const originalType =
    (element.data?.originalType as string) ||
    (element.type === "floating" || element.type === "smoothstep" ? "bpmn:sequenceFlow" : "");

  const isUserTask = originalType === "bpmn:userTask";
  const isBusinessRuleTask = originalType === "bpmn:businessRuleTask";
  const isServiceTask = originalType === "bpmn:serviceTask" || originalType === "bpmn:sendTask";
  const isScriptTask = originalType === "bpmn:scriptTask";
  const isCallActivity = originalType === "bpmn:callActivity";
  const isReceiveTask = originalType === "bpmn:receiveTask";
  const isSubProcess = originalType === "bpmn:subProcess" || originalType === "bpmn:transaction";
  const isTransaction =
    originalType === "bpmn:transaction" || String(data.transaction ?? "") === "true";
  const isSequenceFlow =
    originalType === "bpmn:sequenceFlow" || element.type === "floating" || element.type === "smoothstep";
  const isEventNode =
    originalType === "bpmn:startEvent" ||
    originalType === "bpmn:endEvent" ||
    originalType === "bpmn:intermediateCatchEvent" ||
    originalType === "bpmn:intermediateThrowEvent" ||
    originalType === "bpmn:boundaryEvent";
  const supportsMultiInstance = MULTI_INSTANCE_TYPES.has(originalType);

  const eventKind = isEventNode ? inferEventDefinitionKind(data) : "none";
  // Palette "Message Throw" without a marker should still expose message fields.
  const effectiveEventKind =
    eventKind !== "none"
      ? eventKind
      : originalType === "bpmn:intermediateThrowEvent"
        ? "message"
        : originalType === "bpmn:intermediateCatchEvent"
          ? "timer"
          : "none";

  const isGateway = originalType.includes("Gateway");
  const isTextAnnotation =
    originalType === "bpmn:textAnnotation" || data.visualKind === "textAnnotation";
  const annotationText = String(data.text || "");
  const defaultFlow = (data["@_default"] as string) || "";
  const cancelActivity = String(data["@_cancelActivity"] ?? "true");
  const triggeredByEvent = String(data["@_triggeredByEvent"] ?? "false");
  const receiveMessageRef = (data["@_messageRef"] as string) || "";
  const attachedToRef = (data["@_attachedToRef"] as string) || "";
  const activityRef =
    (typeof data.activity_ref === "string" && data.activity_ref) ||
    (() => {
      const def = data["bpmn:compensateEventDefinition"];
      if (typeof def === "object" && def !== null) {
        return String((def as Record<string, unknown>)["@_activityRef"] ?? "");
      }
      return "";
    })();
  const miLoop = data["bpmn:multiInstanceLoopCharacteristics"];
  const miIsSequential =
    typeof miLoop === "object" && miLoop !== null
      ? String((miLoop as Record<string, unknown>)["@_isSequential"] ?? "false") === "true"
      : false;
  const attachableNodes = nodes.filter((n) => {
    const t = String(n.data?.originalType || n.type || "");
    if (!t || t.includes("Event") || t.includes("Gateway") || n.data?.visualOnly) return false;
    return n.id !== element.id;
  });
  const outgoingEdgeIds = edges.filter((e) => e.source === element.id).map((e) => e.id);

  return (
    <div className="h-full bg-gray-50 border-l overflow-y-auto">
      <Card className="rounded-none border-0 shadow-none bg-transparent h-full flex flex-col">
        <CardHeader className="py-3 px-4 border-b bg-white">
          <CardTitle className="text-sm font-semibold">Properties</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 p-0">
          <Tabs defaultValue="general" className="w-full h-full flex flex-col">
            <div className="px-4 py-2 border-b bg-white">
              <TabsList className="w-full grid grid-cols-2">
                <TabsTrigger value="general">General</TabsTrigger>
                <TabsTrigger value="extensions">Extensions</TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="general" className="flex-1 overflow-y-auto p-4 space-y-4 m-0">
              <div className="space-y-3">
                <div className="space-y-2">
                  <Label>Name</Label>
                  <DebouncedInput
                    value={elementName}
                    onValueChange={(val) => updateField("label", val)}
                    placeholder="e.g. Approve Request"
                    className="bg-white"
                  />
                </div>

                <div className="space-y-2">
                  <Label>ID</Label>
                  <DebouncedInput
                    value={elementId}
                    onValueChange={(val) => {
                      const next = val.trim();
                      if (!next) {
                        setIdError("ID is required");
                        return;
                      }
                      if (!NCNAME_RE.test(next)) {
                        setIdError("ID must be a valid NCName (letter or _ first)");
                        return;
                      }
                      if (next !== elementId && existingIds.includes(next)) {
                        setIdError("ID must be unique in this diagram");
                        return;
                      }
                      if (next === elementId) {
                        setIdError(null);
                        return;
                      }
                      if (onRenameId) {
                        const err = onRenameId(elementId, next);
                        setIdError(err);
                        return;
                      }
                      setIdError(null);
                    }}
                    placeholder="e.g. validateOrder"
                    className="bg-white font-mono text-xs"
                  />
                  {idError ? (
                    <p className="text-xs text-destructive">{idError}</p>
                  ) : (
                    <p className="text-[10px] text-muted-foreground">BPMN element id used in XML and runtime.</p>
                  )}
                </div>
              </div>

              {isTextAnnotation && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Annotation
                  </h4>
                  <div className="space-y-2">
                    <Label>Text</Label>
                    <DebouncedTextarea
                      value={annotationText}
                      onValueChange={(val) => updateField("text", val)}
                      placeholder="Annotation body exported as bpmn:text"
                      className="bg-white font-mono text-xs"
                      rows={4}
                    />
                  </div>
                </div>
              )}

              {isUserTask && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">User Task</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Assignee</Label>
                      <DebouncedInput value={assignee} onValueChange={(val) => updateExtensionField("assignee", val)} placeholder="e.g. user@example.com" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Candidate Users</Label>
                      <DebouncedInput value={candidateUsers} onValueChange={(val) => updateExtensionField("candidateUsers", val)} placeholder="e.g. user1,user2" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Candidate Groups</Label>
                      <DebouncedInput value={candidateGroups} onValueChange={(val) => updateExtensionField("candidateGroups", val)} placeholder="e.g. managers,hr" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Due Date</Label>
                      <DebouncedInput value={dueDate} onValueChange={(val) => updateExtensionField("dueDate", val)} placeholder="e.g. PT1H or 2026-08-02T15:00:00Z" className="bg-white font-mono text-xs" />
                      <p className="text-[10px] text-muted-foreground">ISO-8601 duration or absolute datetime.</p>
                    </div>
                  </div>
                </div>
              )}

              {isBusinessRuleTask && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Business Rule</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Decision Ref</Label>
                      <DebouncedInput value={decisionRef} onValueChange={(val) => updateExtensionField("decisionRef", val)} placeholder="e.g. approve_decision" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Result Variable</Label>
                      <DebouncedInput value={brResultVariable} onValueChange={(val) => updateExtensionField("resultVariable", val)} placeholder="e.g. decisionResult" className="bg-white" />
                    </div>
                  </div>
                </div>
              )}

              {isServiceTask && (() => {
                const connector = getConnectorDescriptor(taskType);
                const connectorInputs = connector ? readConnectorInputs(data, taskType) : {};
                const updateConnectorInput = (key: string, value: string) => {
                  if (!element || !connector) return;
                  const nextInputs = { ...connectorInputs, [key]: value };
                  onUpdate(element.id, applyConnectorInputs(data, taskType, nextInputs));
                };
                return (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    {originalType === "bpmn:sendTask" ? "Send Task" : "Service Task"}
                  </h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Topic</Label>
                      <DebouncedInput value={topic} onValueChange={(val) => updateExtensionField("topic", val)} placeholder="e.g. payment_processing" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Task Type / Connector</Label>
                      <select
                        className="flex h-10 w-full rounded-md border border-input bg-white px-3 py-2 text-sm"
                        value={CONNECTOR_DESCRIPTORS.some((d) => d.jobType === taskType) ? taskType : ""}
                        onChange={(e) => {
                          const next = e.target.value;
                          updateExtensionField("taskType", next);
                        }}
                      >
                        <option value="">Custom / other…</option>
                        {CONNECTOR_DESCRIPTORS.map((d) => (
                          <option key={d.jobType} value={d.jobType}>
                            {d.name} ({d.jobType})
                          </option>
                        ))}
                      </select>
                      <DebouncedInput value={taskType} onValueChange={(val) => updateExtensionField("taskType", val)} placeholder="e.g. io.artificialflow.connector.http" className="bg-white font-mono text-xs" />
                      <p className="text-[10px] text-muted-foreground">
                        Connector inputs are exported as extension properties and become process variables at job creation. Start variables with the same keys can supply or override them.
                      </p>
                    </div>
                    {connector && (
                      <div className="space-y-3 pt-2 border-t">
                        <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          {connector.name} connector
                        </h4>
                        {connector.fields.map((field) => (
                          <div key={field.key} className="space-y-2">
                            <Label>
                              {field.label}
                              {field.required ? " *" : ""}
                            </Label>
                            {field.kind === "boolean" ? (
                              <select
                                className="flex h-10 w-full rounded-md border border-input bg-white px-3 py-2 text-sm"
                                value={connectorInputs[field.key] || field.default || "true"}
                                onChange={(e) => updateConnectorInput(field.key, e.target.value)}
                              >
                                <option value="true">true</option>
                                <option value="false">false</option>
                              </select>
                            ) : (
                              <DebouncedInput
                                value={connectorInputs[field.key] || ""}
                                onValueChange={(val) => updateConnectorInput(field.key, val)}
                                placeholder={field.kind === "json" ? '{"key":"value"}' : field.default || field.description}
                                className="bg-white font-mono text-xs"
                              />
                            )}
                            <p className="text-[10px] text-muted-foreground">{field.description}</p>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                );
              })()}

              {isScriptTask && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Script Task</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Script Format</Label>
                      <DebouncedInput value={scriptFormat} onValueChange={(val) => updateExtensionField("scriptFormat", val)} placeholder="e.g. javascript" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Script</Label>
                      <DebouncedTextarea
                        value={scriptBody}
                        onValueChange={(val) => updateField("bpmn:script", val)}
                        placeholder="e.g. return variables.get('x') + 1;"
                        className="bg-white font-mono text-xs"
                        rows={6}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Result Variable</Label>
                      <DebouncedInput value={scriptResultVariable} onValueChange={(val) => updateExtensionField("resultVariable", val)} placeholder="e.g. scriptResult" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Timeout (optional)</Label>
                      <DebouncedInput value={scriptTimeout} onValueChange={(val) => updateExtensionField("timeout", val)} placeholder="e.g. PT30S" className="bg-white font-mono text-xs" />
                    </div>
                  </div>
                </div>
              )}

              {isCallActivity && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Call Activity</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Called Element</Label>
                      <DebouncedInput value={calledElement} onValueChange={(val) => updateField("@_calledElement", val)} placeholder="e.g. process_id" className="bg-white" />
                    </div>
                    <div className="space-y-2">
                      <Label>Called Element Version (optional)</Label>
                      <DebouncedInput
                        value={calledElementVersion}
                        onValueChange={(val) => updateExtensionField("calledElementVersion", val)}
                        placeholder="e.g. 2 or latest"
                        className="bg-white font-mono text-xs"
                      />
                    </div>
                  </div>
                </div>
              )}

              {isReceiveTask && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Receive Task</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Message Ref / Name</Label>
                      <DebouncedInput value={receiveMessageRef} onValueChange={(val) => updateField("receiveMessageRef", val)} placeholder="e.g. DocumentsReceived" className="bg-white font-mono text-xs" />
                    </div>
                    <div className="space-y-2">
                      <Label>Correlation Key</Label>
                      <DebouncedInput value={correlationKey} onValueChange={(val) => updateExtensionField("correlationKey", val)} placeholder="e.g. orderId" className="bg-white" />
                    </div>
                  </div>
                </div>
              )}

              {isSubProcess && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    {isTransaction ? "Transaction" : "Sub Process"}
                  </h4>
                  <div className="space-y-2">
                    <Label>Transaction</Label>
                    <select
                      className="flex h-9 w-full rounded-md border border-input bg-white px-3 py-1 text-sm"
                      value={isTransaction ? "true" : "false"}
                      onChange={(e) => {
                        if (!element) return;
                        const next = e.target.value === "true";
                        onUpdate(element.id, {
                          ...data,
                          transaction: next,
                          originalType: next ? "bpmn:transaction" : "bpmn:subProcess",
                        });
                      }}
                    >
                      <option value="false">No (standard sub-process)</option>
                      <option value="true">Yes (BPMN transaction)</option>
                    </select>
                  </div>
                  {!isTransaction && (
                    <div className="space-y-2">
                      <Label>Triggered by Event</Label>
                      <select
                        className="flex h-9 w-full rounded-md border border-input bg-white px-3 py-1 text-sm"
                        value={triggeredByEvent === "true" ? "true" : "false"}
                        onChange={(e) => updateField("triggeredByEvent", e.target.value)}
                      >
                        <option value="false">No (embedded, sequence-flow entered)</option>
                        <option value="true">Yes (event sub-process)</option>
                      </select>
                    </div>
                  )}
                </div>
              )}

              {isEventNode && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Event ({effectiveEventKind === "none" ? "none" : effectiveEventKind})
                  </h4>

                  {originalType === "bpmn:boundaryEvent" && (
                    <div className="space-y-3">
                      <div className="space-y-2">
                        <Label>Attached To</Label>
                        <select
                          className="flex h-9 w-full rounded-md border border-input bg-white px-3 py-1 text-sm"
                          value={attachedToRef}
                          onChange={(e) => updateField("attachedToRef", e.target.value)}
                        >
                          <option value="">— select activity —</option>
                          {attachableNodes.map((n) => (
                            <option key={n.id} value={n.id}>
                              {String(n.data?.label || n.id)} ({n.id})
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="space-y-2">
                        <Label>Cancel Activity (interrupting)</Label>
                        <select
                          className="flex h-9 w-full rounded-md border border-input bg-white px-3 py-1 text-sm"
                          value={cancelActivity === "false" ? "false" : "true"}
                          onChange={(e) => updateField("cancelActivity", e.target.value)}
                        >
                          <option value="true">Yes — interrupting</option>
                          <option value="false">No — non-interrupting</option>
                        </select>
                      </div>
                    </div>
                  )}

                  {effectiveEventKind === "timer" && (
                    <div className="space-y-2">
                      <Label>Timer type</Label>
                      <select
                        className="w-full rounded-md border border-input bg-white px-3 py-2 text-xs"
                        value={getTimerKind(data)}
                        onChange={(e) => updateField("timerKind", e.target.value)}
                      >
                        <option value="duration">Duration (PT…)</option>
                        <option value="date">Date (timeDate)</option>
                        <option value="cycle">Cycle (R[n]/PT…)</option>
                      </select>
                      {getTimerKind(data) === "duration" && (
                        <>
                          <Label>Time Duration</Label>
                          <DebouncedInput
                            value={getTimerDuration(data)}
                            onValueChange={(val) => updateField("timerDuration", val)}
                            placeholder="e.g. PT1H, PT5M, PT30S"
                            className="bg-white font-mono text-xs"
                          />
                          <p className="text-[10px] text-muted-foreground">ISO-8601 duration (PT1M, PT1H).</p>
                        </>
                      )}
                      {getTimerKind(data) === "date" && (
                        <>
                          <Label>Time Date</Label>
                          <DebouncedInput
                            value={getTimerDate(data)}
                            onValueChange={(val) => updateField("timerDate", val)}
                            placeholder="e.g. 2026-08-02T15:00:00Z"
                            className="bg-white font-mono text-xs"
                          />
                          <p className="text-[10px] text-muted-foreground">RFC3339 absolute timestamp.</p>
                        </>
                      )}
                      {getTimerKind(data) === "cycle" && (
                        <>
                          <Label>Time Cycle</Label>
                          <DebouncedInput
                            value={getTimerCycle(data)}
                            onValueChange={(val) => updateField("timerCycle", val)}
                            placeholder="e.g. R3/PT10S or R/PT1H"
                            className="bg-white font-mono text-xs"
                          />
                          <p className="text-[10px] text-muted-foreground">
                            Repeating ISO cycle: R3/PT10S (three times) or R/PT1H (infinite).
                          </p>
                        </>
                      )}
                    </div>
                  )}

                  {effectiveEventKind === "message" && (
                    <>
                      <div className="space-y-2">
                        <Label>Message Name</Label>
                        <DebouncedInput
                          value={displayMessageName(data)}
                          onValueChange={(val) => updateField("messageName", val)}
                          placeholder="e.g. DocumentsReceived"
                          className="bg-white font-mono text-xs"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Correlation Key</Label>
                        <DebouncedInput
                          value={correlationKey}
                          onValueChange={(val) => updateExtensionField("correlationKey", val)}
                          placeholder="e.g. orderId"
                          className="bg-white"
                        />
                      </div>
                    </>
                  )}

                  {effectiveEventKind === "signal" && (
                    <div className="space-y-2">
                      <Label>Signal Name</Label>
                      <DebouncedInput
                        value={displaySignalName(data)}
                        onValueChange={(val) => updateField("signalName", val)}
                        placeholder="e.g. InvoiceApproved"
                        className="bg-white font-mono text-xs"
                      />
                    </div>
                  )}

                  {effectiveEventKind === "escalation" && (
                    <div className="space-y-2">
                      <Label>Escalation Code</Label>
                      <DebouncedInput
                        value={getEscalationCode(data)}
                        onValueChange={(val) => updateField("escalationCode", val)}
                        placeholder="e.g. ESC_TIMEOUT"
                        className="bg-white font-mono text-xs"
                      />
                    </div>
                  )}

                  {effectiveEventKind === "link" && (
                    <div className="space-y-2">
                      <Label>Link Name</Label>
                      <DebouncedInput
                        value={getLinkName(data)}
                        onValueChange={(val) => updateField("linkName", val)}
                        placeholder="e.g. ContinueHere"
                        className="bg-white font-mono text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground">Throw and catch links must share the same name.</p>
                    </div>
                  )}

                  {effectiveEventKind === "conditional" && (
                    <div className="space-y-2">
                      <Label>Condition Expression</Label>
                      <DebouncedInput
                        value={getConditionalExpression(data)}
                        onValueChange={(val) => updateField("conditionalExpression", val)}
                        placeholder="e.g. approved == true"
                        className="bg-white"
                      />
                    </div>
                  )}

                  {effectiveEventKind === "error" && (
                    <>
                      <div className="space-y-2">
                        <Label>Error Code</Label>
                        <DebouncedInput
                          value={getErrorCode(data)}
                          onValueChange={(val) => updateField("errorCode", val)}
                          placeholder="e.g. ERR_VALIDATION"
                          className="bg-white font-mono text-xs"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Error Message</Label>
                        <DebouncedInput
                          value={errorMessage}
                          onValueChange={(val) => updateExtensionField("errorMessage", val)}
                          placeholder="Optional error message"
                          className="bg-white"
                        />
                      </div>
                    </>
                  )}

                  {effectiveEventKind === "terminate" && (
                    <p className="text-xs text-muted-foreground">
                      Terminate end cancels sibling tokens and completes the scope. No extra configuration.
                    </p>
                  )}

                  {effectiveEventKind === "compensate" && (
                    <div className="space-y-2">
                      <Label>Activity Ref</Label>
                      <DebouncedInput
                        value={activityRef}
                        onValueChange={(val) => updateField("activityRef", val)}
                        placeholder="e.g. Task_to_compensate"
                        className="bg-white font-mono text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground">
                        Optional activity to compensate; leave empty for the default compensation handler.
                      </p>
                    </div>
                  )}

                  {effectiveEventKind === "cancel" && (
                    <p className="text-xs text-muted-foreground">
                      Cancel end is used inside a transaction sub-process. No extra configuration.
                    </p>
                  )}

                  {effectiveEventKind === "none" && (
                    <p className="text-xs text-muted-foreground">None event — no event definition.</p>
                  )}
                </div>
              )}

              {supportsMultiInstance && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Multi-instance
                  </h4>
                  <div className="space-y-3">
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={miIsSequential}
                        onChange={(e) => updateField("miSequential", e.target.checked ? "true" : "false")}
                      />
                      Sequential
                    </label>
                    <div className="space-y-2">
                      <Label>Collection</Label>
                      <DebouncedInput
                        value={miCollection}
                        onValueChange={(val) => updateField("miCollection", val)}
                        placeholder="e.g. items"
                        className="bg-white font-mono text-xs"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Element Variable</Label>
                      <DebouncedInput
                        value={miElementVariable}
                        onValueChange={(val) => updateField("miElementVariable", val)}
                        placeholder="e.g. item"
                        className="bg-white font-mono text-xs"
                      />
                    </div>
                  </div>
                </div>
              )}

              {isSequenceFlow && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Sequence Flow</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Condition Expression</Label>
                      <DebouncedInput
                        value={condition}
                        onValueChange={(val) => updateField("bpmn:conditionExpression", val)}
                        placeholder="e.g. amount > 1000"
                        className="bg-white"
                      />
                    </div>
                  </div>
                </div>
              )}

              {isGateway && (originalType === "bpmn:exclusiveGateway" || originalType === "bpmn:inclusiveGateway") && (
                <div className="space-y-3 pt-2 border-t">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Gateway</h4>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Default Flow ID</Label>
                      <select
                        className="flex h-9 w-full rounded-md border border-input bg-white px-3 py-1 text-sm"
                        value={defaultFlow}
                        onChange={(e) => updateField("@_default", e.target.value)}
                      >
                        <option value="">— none —</option>
                        {outgoingEdgeIds.map((id) => (
                          <option key={id} value={id}>
                            {id}
                          </option>
                        ))}
                        {defaultFlow && !outgoingEdgeIds.includes(defaultFlow) ? (
                          <option value={defaultFlow}>{defaultFlow} (missing)</option>
                        ) : null}
                      </select>
                      <p className="text-[10px] text-muted-foreground">The flow to take if no other conditions match.</p>
                    </div>
                  </div>
                </div>
              )}
            </TabsContent>

            <TabsContent value="extensions" className="flex-1 overflow-y-auto p-4 m-0">
              <ExtensionProps data={element.data || null} onUpdate={(newData) => onUpdate(element.id, newData)} />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
