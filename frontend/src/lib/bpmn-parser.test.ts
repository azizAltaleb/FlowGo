import { describe, expect, it } from "vitest";
import { type Edge, type Node } from "@xyflow/react";
import { generateBpmnXml, parseBpmnXml } from "./bpmn-parser";
import { withErrorCode, withEscalationCode } from "./bpmn-event-definitions";

const wrapProcess = (body: string, namespaces = "") => `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions
  xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  ${namespaces}
  id="Definitions_1"
  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" name="ArtificialFlow Attributes" isExecutable="true">
    <bpmn:startEvent id="start"/>
    ${body}
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="task"/>
    <bpmn:sequenceFlow id="f2" sourceRef="task" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`;

describe("ArtificialFlow BPMN attributes", () => {
  it("imports prefixed ArtificialFlow attributes into canonical model keys", () => {
    const result = parseBpmnXml(
      wrapProcess(
        `<bpmn:userTask id="task" af:assignee="canonical-user"/>`,
        'xmlns:af="http://artificialflow.io/schema/1.0/bpmn"',
      ),
    );
    const task = result.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe("canonical-user");
  });

  it("imports ArtificialFlow-prefixed attributes", () => {
    const result = parseBpmnXml(
      wrapProcess(
        `<bpmn:userTask id="task" artificialflow:assignee="modeler-user"/>`,
        'xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"',
      ),
    );
    const task = result.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe("modeler-user");
  });

  it("prefers ArtificialFlow attributes over plain attributes", () => {
    const xml = wrapProcess(
      `<bpmn:userTask
        id="task"
        artificialflow:assignee="canonical-user"
        assignee="plain-user">
        <bpmn:extensionElements>
          <artificialflow:properties>
            <artificialflow:property name="source" value="canonical"/>
            <artificialflow:property name="priority" value="high"/>
          </artificialflow:properties>
        </bpmn:extensionElements>
      </bpmn:userTask>`,
      'xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"',
    );

    const result = parseBpmnXml(xml);
    const task = result.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe("canonical-user");
    const extensionElements = task?.data["bpmn:extensionElements"] as Record<string, unknown>;
    const properties = extensionElements["artificialflow:properties"] as Record<string, unknown>;
    expect(properties["artificialflow:property"]).toEqual([
      { "@_name": "source", "@_value": "canonical" },
      { "@_name": "priority", "@_value": "high" },
    ]);
  });

  it("exports and round-trips the ArtificialFlow namespace and prefix", () => {
    const imported = parseBpmnXml(
      wrapProcess(
        `<bpmn:userTask id="task" artificialflow:assignee="modeler-user">
          <bpmn:extensionElements>
            <artificialflow:properties>
              <artificialflow:property name="priority" value="high"/>
            </artificialflow:properties>
          </bpmn:extensionElements>
        </bpmn:userTask>`,
        'xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"',
      ),
    );
    const exported = generateBpmnXml(
      imported.nodes,
      imported.edges,
      imported.processId,
      imported.processName,
    );

    expect(exported).toContain('xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"');
    expect(exported).toContain('artificialflow:assignee="modeler-user"');
    expect(exported).toContain("<artificialflow:properties>");
    expect(exported).toContain('<artificialflow:property name="priority" value="high"');

    const reparsed = parseBpmnXml(exported);
    const task = reparsed.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe("modeler-user");
  });
});

describe("BPMN modeler completeness export", () => {
  it("exports escalation/error root catalogs and event definition refs", () => {
    const nodes: Node[] = [
      {
        id: "start",
        type: "startEvent",
        position: { x: 0, y: 0 },
        data: { label: "Start", originalType: "bpmn:startEvent" },
      },
      {
        id: "escEnd",
        type: "endEvent",
        position: { x: 200, y: 0 },
        data: withEscalationCode(
          { label: "Escalation End", originalType: "bpmn:endEvent" },
          "ESC_TIMEOUT",
        ),
      },
      {
        id: "errEnd",
        type: "endEvent",
        position: { x: 200, y: 100 },
        data: withErrorCode(
          { label: "Error End", originalType: "bpmn:endEvent" },
          "ERR_VALIDATION",
        ),
      },
    ];
    const edges: Edge[] = [
      { id: "f1", source: "start", target: "escEnd" },
      { id: "f2", source: "start", target: "errEnd" },
    ];

    const exported = generateBpmnXml(nodes, edges, "Process_1", "Catalog Test");
    expect(exported).toContain('escalationRef="Escalation_ESC_TIMEOUT"');
    expect(exported).toContain('errorRef="Error_ERR_VALIDATION"');
    expect(exported).toMatch(/<bpmn:escalation[^>]*id="Escalation_ESC_TIMEOUT"/);
    expect(exported).toMatch(/<bpmn:error[^>]*id="Error_ERR_VALIDATION"/);
    expect(exported).toContain('escalationCode="ESC_TIMEOUT"');
    expect(exported).toContain('errorCode="ERR_VALIDATION"');

    const reparsed = parseBpmnXml(exported);
    const esc = reparsed.nodes.find((n) => n.id === "escEnd");
    const err = reparsed.nodes.find((n) => n.id === "errEnd");
    const escDef = esc?.data["bpmn:escalationEventDefinition"] as Record<string, unknown>;
    const errDef = err?.data["bpmn:errorEventDefinition"] as Record<string, unknown>;
    expect(escDef?.["@_escalationRef"]).toBe("Escalation_ESC_TIMEOUT");
    expect(errDef?.["@_errorRef"]).toBe("Error_ERR_VALIDATION");
  });

  it("exports script task body as bpmn:script", () => {
    const nodes: Node[] = [
      {
        id: "start",
        type: "startEvent",
        position: { x: 0, y: 0 },
        data: { label: "Start", originalType: "bpmn:startEvent" },
      },
      {
        id: "script",
        type: "scriptTask",
        position: { x: 160, y: 0 },
        data: {
          label: "Run Script",
          originalType: "bpmn:scriptTask",
          "bpmn:script": "return 1 + 1;",
          "@_artificialflow:scriptFormat": "javascript",
          "@_artificialflow:resultVariable": "sum",
          "@_artificialflow:timeout": "PT10S",
        },
      },
      {
        id: "end",
        type: "endEvent",
        position: { x: 320, y: 0 },
        data: { label: "End", originalType: "bpmn:endEvent" },
      },
    ];
    const edges: Edge[] = [
      { id: "f1", source: "start", target: "script" },
      { id: "f2", source: "script", target: "end" },
    ];

    const exported = generateBpmnXml(nodes, edges);
    expect(exported).toContain("<bpmn:script>return 1 + 1;</bpmn:script>");
    expect(exported).toContain('artificialflow:scriptFormat="javascript"');
    expect(exported).toContain('artificialflow:timeout="PT10S"');

    const reparsed = parseBpmnXml(exported);
    const script = reparsed.nodes.find((n) => n.id === "script");
    expect(script?.data["bpmn:script"]).toBe("return 1 + 1;");
  });

  it("exports multi-instance loop characteristics and AF collection attrs", () => {
    const nodes: Node[] = [
      {
        id: "start",
        type: "startEvent",
        position: { x: 0, y: 0 },
        data: { label: "Start", originalType: "bpmn:startEvent" },
      },
      {
        id: "task",
        type: "userTask",
        position: { x: 160, y: 0 },
        data: {
          label: "Review Items",
          originalType: "bpmn:userTask",
          "@_artificialflow:collection": "items",
          "@_artificialflow:elementVariable": "item",
          "bpmn:multiInstanceLoopCharacteristics": { "@_isSequential": "true" },
        },
      },
      {
        id: "end",
        type: "endEvent",
        position: { x: 320, y: 0 },
        data: { label: "End", originalType: "bpmn:endEvent" },
      },
    ];
    const edges: Edge[] = [
      { id: "f1", source: "start", target: "task" },
      { id: "f2", source: "task", target: "end" },
    ];

    const exported = generateBpmnXml(nodes, edges);
    expect(exported).toContain("bpmn:multiInstanceLoopCharacteristics");
    expect(exported).toContain('isSequential="true"');
    expect(exported).toContain('artificialflow:collection="items"');
    expect(exported).toContain('artificialflow:elementVariable="item"');

    const reparsed = parseBpmnXml(exported);
    const task = reparsed.nodes.find((n) => n.id === "task");
    const mi = task?.data["bpmn:multiInstanceLoopCharacteristics"] as Record<string, unknown>;
    expect(mi?.["@_isSequential"]).toBe("true");
    expect(task?.data["@_artificialflow:collection"]).toBe("items");
    expect(task?.data["@_artificialflow:elementVariable"]).toBe("item");
  });

  it("exports collaboration/participant, laneSet/flowNodeRefs, and annotation text", () => {
    const nodes: Node[] = [
      {
        id: "Pool_1",
        type: "visualArtifact",
        position: { x: 0, y: 0 },
        data: {
          label: "Main Pool",
          originalType: "bpmn:participant",
          visualKind: "participant",
          visualOnly: true,
          width: 600,
          height: 300,
        },
        style: { width: 600, height: 300 },
      },
      {
        id: "Lane_1",
        type: "visualArtifact",
        position: { x: 20, y: 40 },
        data: {
          label: "Ops",
          originalType: "bpmn:lane",
          visualKind: "lane",
          visualOnly: true,
          width: 560,
          height: 240,
        },
        style: { width: 560, height: 240 },
      },
      {
        id: "start",
        type: "startEvent",
        position: { x: 80, y: 100 },
        parentId: "Lane_1",
        data: { label: "Start", originalType: "bpmn:startEvent", width: 36, height: 36 },
        style: { width: 36, height: 36 },
      },
      {
        id: "task",
        type: "userTask",
        position: { x: 180, y: 90 },
        parentId: "Lane_1",
        data: { label: "Review", originalType: "bpmn:userTask", width: 100, height: 80 },
        style: { width: 100, height: 80 },
      },
      {
        id: "note",
        type: "visualArtifact",
        position: { x: 320, y: 40 },
        data: {
          label: "Note",
          text: "Review carefully",
          originalType: "bpmn:textAnnotation",
          visualKind: "textAnnotation",
          visualOnly: true,
          width: 120,
          height: 60,
        },
        style: { width: 120, height: 60 },
      },
      {
        id: "Group_1",
        type: "visualArtifact",
        position: { x: 400, y: 160 },
        data: {
          label: "Bundle",
          originalType: "bpmn:group",
          visualKind: "group",
          visualOnly: true,
          width: 140,
          height: 80,
        },
        style: { width: 140, height: 80 },
      },
      {
        id: "MF_1",
        type: "visualArtifact",
        position: { x: 40, y: 280 },
        data: {
          label: "Ping",
          originalType: "bpmn:messageFlow",
          visualKind: "messageFlow",
          visualOnly: true,
          "@_sourceRef": "Pool_1",
          "@_targetRef": "Pool_1",
          width: 100,
          height: 40,
        },
        style: { width: 100, height: 40 },
      },
    ];
    const edges: Edge[] = [{ id: "f1", source: "start", target: "task" }];

    const exported = generateBpmnXml(nodes, edges, "Process_1", "Collab Test");
    expect(exported).toContain("<bpmn:collaboration");
    expect(exported).toContain('processRef="Process_1"');
    expect(exported).toContain('id="Pool_1"');
    expect(exported).toContain("<bpmn:laneSet");
    expect(exported).toContain("<bpmn:flowNodeRef>start</bpmn:flowNodeRef>");
    expect(exported).toContain("<bpmn:flowNodeRef>task</bpmn:flowNodeRef>");
    expect(exported).toContain("<bpmn:text>Review carefully</bpmn:text>");
    expect(exported).toContain("<bpmn:group");
    expect(exported).toContain("<bpmn:messageFlow");
    expect(exported).not.toMatch(/<bpmn:process[^>]*>[\s\S]*<bpmn:participant/);

    const reparsed = parseBpmnXml(exported);
    expect(reparsed.nodes.find((n) => n.id === "Pool_1")?.data.visualKind).toBe("participant");
    expect(reparsed.nodes.find((n) => n.id === "Lane_1")?.data.visualKind).toBe("lane");
    expect(reparsed.nodes.find((n) => n.id === "note")?.data.text).toBe("Review carefully");
    expect(reparsed.nodes.find((n) => n.id === "Group_1")?.data.visualKind).toBe("group");
    expect(reparsed.nodes.find((n) => n.id === "MF_1")?.data.visualKind).toBe("messageFlow");
  });
});
