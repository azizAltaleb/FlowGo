import { describe, expect, it } from "vitest";
import { generateBpmnXml, parseBpmnXml } from "./bpmn-parser";

const wrapProcess = (body: string, namespaces = "") => `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions
  xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  ${namespaces}
  id="Definitions_1"
  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" name="Namespace Migration" isExecutable="true">
    <bpmn:startEvent id="start"/>
    ${body}
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="task"/>
    <bpmn:sequenceFlow id="f2" sourceRef="task" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`;

describe("ArtificialFlow BPMN namespace migration", () => {
  it.each([
    {
      name: "canonical",
      namespaces: 'xmlns:af="http://artificialflow.io/schema/1.0/bpmn"',
      attribute: 'af:assignee="canonical-user"',
      expected: "canonical-user",
    },
    {
      name: "legacy",
      namespaces: 'xmlns:flowgo="http://flowgo.com/schema/1.0/bpmn"',
      attribute: 'flowgo:assignee="legacy-user"',
      expected: "legacy-user",
    },
    {
      name: "unqualified",
      namespaces: "",
      attribute: 'assignee="plain-user"',
      expected: "plain-user",
    },
  ])("imports $name attributes into canonical model keys", ({ namespaces, attribute, expected }) => {
    const result = parseBpmnXml(wrapProcess(`<bpmn:userTask id="task" ${attribute}/>`, namespaces));
    const task = result.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe(expected);
    expect(task?.data["@_flowgo:assignee"]).toBeUndefined();
    expect(task?.data["@_assignee"]).toBeUndefined();
  });

  it("prefers canonical values while preserving unique legacy extension properties", () => {
    const xml = wrapProcess(
      `<bpmn:userTask
        id="task"
        artificialflow:assignee="canonical-user"
        flowgo:assignee="legacy-user"
        assignee="plain-user">
        <bpmn:extensionElements>
          <artificialflow:properties>
            <artificialflow:property name="source" value="canonical"/>
          </artificialflow:properties>
          <flowgo:properties>
            <flowgo:property name="source" value="legacy"/>
            <flowgo:property name="legacy_only" value="preserved"/>
          </flowgo:properties>
        </bpmn:extensionElements>
      </bpmn:userTask>`,
      [
        'xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"',
        'xmlns:flowgo="http://flowgo.com/schema/1.0/bpmn"',
      ].join(" "),
    );

    const result = parseBpmnXml(xml);
    const task = result.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe("canonical-user");
    const extensionElements = task?.data["bpmn:extensionElements"] as Record<string, unknown>;
    const properties = extensionElements["artificialflow:properties"] as Record<string, unknown>;
    expect(properties["artificialflow:property"]).toEqual([
      { "@_name": "source", "@_value": "canonical" },
      { "@_name": "legacy_only", "@_value": "preserved" },
    ]);
  });

  it("exports and round-trips only the canonical namespace and prefix", () => {
    const imported = parseBpmnXml(wrapProcess(
      `<bpmn:userTask id="task" flowgo:assignee="legacy-user">
        <bpmn:extensionElements>
          <flowgo:properties>
            <flowgo:property name="priority" value="high"/>
          </flowgo:properties>
        </bpmn:extensionElements>
      </bpmn:userTask>`,
      'xmlns:flowgo="http://flowgo.com/schema/1.0/bpmn"',
    ));
    const exported = generateBpmnXml(
      imported.nodes,
      imported.edges,
      imported.processId,
      imported.processName,
    );

    expect(exported).toContain('xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"');
    expect(exported).toContain('artificialflow:assignee="legacy-user"');
    expect(exported).toContain("<artificialflow:properties>");
    expect(exported).toContain('<artificialflow:property name="priority" value="high">');
    expect(exported).not.toContain("flowgo");
    expect(exported).not.toContain("http://flowgo.com/schema/1.0/bpmn");

    const reparsed = parseBpmnXml(exported);
    const task = reparsed.nodes.find((node) => node.id === "task");
    expect(task?.data["@_artificialflow:assignee"]).toBe("legacy-user");
  });
});
