import { describe, expect, it } from "vitest";
import { generateBpmnXml, parseBpmnXml } from "./bpmn-parser";

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
