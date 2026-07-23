export type BpmnFixtureName =
  | "roleBasedComplexProcess"
  | "userTaskApproval"
  | "parserElementMatrix"
  | "extensionProperties"
  | "gatewayJoins"
  | "callBusinessManual"
  | "eventGateway"
  | "boundaryTimer"
  | "messageSignal"
  | "serviceUserAssignment"
  | "unsupportedSendTask"
  | "unsupportedElement";

export interface BpmnDocument {
  processId: string;
  xml: string;
}

export interface BpmnBundle {
  primary: BpmnDocument;
  dependencies?: BpmnDocument[];
}

const ns = `xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:flowgo="http://flowgo.com/schema/1.0/bpmn"`;

function doc(id: string, body: string, extraDefs = ""): string {
  return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions ${ns} id="Definitions_${id}" targetNamespace="https://artificialflow.io/uat">
${extraDefs}
${body}
</bpmn:definitions>`;
}

function process(id: string, name: string, body: string): string {
  return `<bpmn:process id="${id}" name="${name}" isExecutable="true">
${body}
  </bpmn:process>`;
}

export function buildBpmnFixture(name: BpmnFixtureName, runId: string): BpmnBundle {
  const id = (suffix: string) => `UAT_${runId}_${suffix}`;

  switch (name) {
    case "roleBasedComplexProcess": {
      const childId = id("RoleChildProcess");
      const processId = id("RoleComplexProcess");
      return {
        dependencies: [childProcess(childId)],
        primary: {
          processId,
          xml: doc(
            processId,
            process(processId, "UAT Role Based Complex Process", `
    <bpmn:startEvent id="start" name="Request submitted"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:serviceTask id="validateInvoice" name="Validate invoice" flowgo:taskType="uat-complex-worker"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:serviceTask>
    <bpmn:scriptTask id="calculateRisk" name="Calculate risk"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f3</bpmn:outgoing></bpmn:scriptTask>
    <bpmn:businessRuleTask id="classifyInvoice" name="Classify invoice" flowgo:decisionRef="invoice_decision" flowgo:resultVariable="decisionResult"><bpmn:incoming>f3</bpmn:incoming><bpmn:outgoing>f4</bpmn:outgoing></bpmn:businessRuleTask>
    <bpmn:exclusiveGateway id="exclusiveApproval" name="Exclusive amount route" default="f5_low"><bpmn:incoming>f4</bpmn:incoming><bpmn:outgoing>f5</bpmn:outgoing><bpmn:outgoing>f5_low</bpmn:outgoing></bpmn:exclusiveGateway>
    <bpmn:parallelGateway id="parallelPrep" name="Parallel preparation"><bpmn:incoming>f5</bpmn:incoming><bpmn:outgoing>f6a</bpmn:outgoing><bpmn:outgoing>f6b</bpmn:outgoing></bpmn:parallelGateway>
    <bpmn:scriptTask id="parallelRiskStamp" name="Parallel risk stamp"><bpmn:incoming>f6a</bpmn:incoming><bpmn:outgoing>f7a</bpmn:outgoing></bpmn:scriptTask>
    <bpmn:scriptTask id="parallelOpsAudit" name="Parallel operations audit"><bpmn:incoming>f6b</bpmn:incoming><bpmn:outgoing>f7b</bpmn:outgoing></bpmn:scriptTask>
    <bpmn:endEvent id="parallelChildEnd" name="Parallel child branch done"><bpmn:incoming>f7b</bpmn:incoming></bpmn:endEvent>
    <bpmn:inclusiveGateway id="inclusivePolicy" name="Inclusive policy checks"><bpmn:incoming>f7a</bpmn:incoming><bpmn:outgoing>f9a</bpmn:outgoing><bpmn:outgoing>f9b</bpmn:outgoing></bpmn:inclusiveGateway>
    <bpmn:scriptTask id="archivePrep" name="Archive preparation"><bpmn:incoming>f9a</bpmn:incoming><bpmn:outgoing>f10a</bpmn:outgoing></bpmn:scriptTask>
    <bpmn:businessRuleTask id="complianceRule" name="Compliance rule" flowgo:decisionRef="compliance_decision" flowgo:resultVariable="complianceDecision"><bpmn:incoming>f9b</bpmn:incoming><bpmn:outgoing>f10b</bpmn:outgoing></bpmn:businessRuleTask>
    <bpmn:endEvent id="complianceEnd" name="Compliance branch done"><bpmn:incoming>f10b</bpmn:incoming></bpmn:endEvent>
    <bpmn:subProcess id="embeddedAudit" name="Embedded audit"><bpmn:incoming>f10a</bpmn:incoming><bpmn:outgoing>f11</bpmn:outgoing></bpmn:subProcess>
    <bpmn:callActivity id="callChild" name="Parallel child validation" calledElement="${childId}"><bpmn:incoming>f11</bpmn:incoming><bpmn:outgoing>f12</bpmn:outgoing></bpmn:callActivity>
    <bpmn:userTask id="accountantReview" name="Accountant review" flowgo:assignee="accountant" flowgo:candidateUsers="accountant,accounts.lead@artificialflow.io" flowgo:candidateGroups="accountant,finance" flowgo:dueDate="PT1H">
      <bpmn:extensionElements>
        <flowgo:properties>
          <flowgo:property name="uat_role" value="accountant"/>
          <flowgo:property name="candidateUsers" value="accountant,accounts.lead@artificialflow.io"/>
          <flowgo:property name="candidateGroups" value="accountant,finance"/>
        </flowgo:properties>
      </bpmn:extensionElements>
      <bpmn:incoming>f12</bpmn:incoming><bpmn:outgoing>f13</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:boundaryEvent id="accountantReminder" name="Accountant reminder" attachedToRef="timerManualFollowup" cancelActivity="false">
      <bpmn:timerEventDefinition><bpmn:timeDuration>PT1H</bpmn:timeDuration></bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:eventBasedGateway id="waitForDocuments" name="Event-based document wait"><bpmn:incoming>f13</bpmn:incoming><bpmn:outgoing>f14a</bpmn:outgoing><bpmn:outgoing>f14b</bpmn:outgoing></bpmn:eventBasedGateway>
    <bpmn:receiveTask id="receiveDocuments" name="Receive supporting documents" messageRef="Message_Documents"><bpmn:incoming>f14a</bpmn:incoming><bpmn:outgoing>f15a</bpmn:outgoing></bpmn:receiveTask>
    <bpmn:receiveTask id="receiveBudgetConfirmation" name="Receive budget confirmation" messageRef="Message_Budget"><bpmn:incoming>f14b</bpmn:incoming><bpmn:outgoing>f15b</bpmn:outgoing></bpmn:receiveTask>
    <bpmn:endEvent id="budgetEnd" name="Budget branch done"><bpmn:incoming>f15b</bpmn:incoming></bpmn:endEvent>
    <bpmn:intermediateCatchEvent id="documentTimer" name="Document timer"><bpmn:outgoing>f18</bpmn:outgoing><bpmn:timerEventDefinition><bpmn:timeDuration>PT1H</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:intermediateCatchEvent>
    <bpmn:manualTask id="timerManualFollowup" name="Timer manual follow-up"><bpmn:incoming>f18</bpmn:incoming><bpmn:outgoing>f19</bpmn:outgoing></bpmn:manualTask>
    <bpmn:userTask id="reviewerApproval" name="Reviewer approval" flowgo:assignee="reviewer" flowgo:candidateUsers="reviewer,reviewer.lead@artificialflow.io" flowgo:candidateGroups="reviewer,approvers" flowgo:dueDate="PT2H">
      <bpmn:extensionElements>
        <flowgo:properties>
          <flowgo:property name="uat_role" value="reviewer"/>
          <flowgo:property name="candidateUsers" value="reviewer,reviewer.lead@artificialflow.io"/>
          <flowgo:property name="candidateGroups" value="reviewer,approvers"/>
        </flowgo:properties>
      </bpmn:extensionElements>
      <bpmn:incoming>f15a</bpmn:incoming><bpmn:outgoing>f17</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:intermediateThrowEvent id="notifyApproval" name="Notify approval"><bpmn:incoming>f19</bpmn:incoming><bpmn:outgoing>f20</bpmn:outgoing><bpmn:signalEventDefinition signalRef="Signal_Approved"/></bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="approvedEnd" name="Approved"><bpmn:incoming>f17</bpmn:incoming></bpmn:endEvent>
    <bpmn:endEvent id="timeoutEnd" name="Timed out"><bpmn:incoming>f20</bpmn:incoming></bpmn:endEvent>
    <bpmn:endEvent id="lowAmountEnd" name="Low amount"><bpmn:incoming>f5_low</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="validateInvoice"/>
    <bpmn:sequenceFlow id="f2" sourceRef="validateInvoice" targetRef="calculateRisk"/>
    <bpmn:sequenceFlow id="f3" sourceRef="calculateRisk" targetRef="classifyInvoice"/>
    <bpmn:sequenceFlow id="f4" sourceRef="classifyInvoice" targetRef="exclusiveApproval"/>
    <bpmn:sequenceFlow id="f5" sourceRef="exclusiveApproval" targetRef="parallelPrep"><bpmn:conditionExpression><![CDATA[amount >= 1000]]></bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f5_low" sourceRef="exclusiveApproval" targetRef="lowAmountEnd"><bpmn:conditionExpression><![CDATA[amount < 1000]]></bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f6a" sourceRef="parallelPrep" targetRef="parallelRiskStamp"/>
    <bpmn:sequenceFlow id="f6b" sourceRef="parallelPrep" targetRef="parallelOpsAudit"/>
    <bpmn:sequenceFlow id="f7a" sourceRef="parallelRiskStamp" targetRef="inclusivePolicy"/>
    <bpmn:sequenceFlow id="f7b" sourceRef="parallelOpsAudit" targetRef="parallelChildEnd"/>
    <bpmn:sequenceFlow id="f9a" sourceRef="inclusivePolicy" targetRef="archivePrep"><bpmn:conditionExpression><![CDATA[needsArchive == true]]></bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f9b" sourceRef="inclusivePolicy" targetRef="complianceRule"><bpmn:conditionExpression><![CDATA[needsCompliance == true]]></bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f10a" sourceRef="archivePrep" targetRef="embeddedAudit"/>
    <bpmn:sequenceFlow id="f10b" sourceRef="complianceRule" targetRef="complianceEnd"/>
    <bpmn:sequenceFlow id="f11" sourceRef="embeddedAudit" targetRef="callChild"/>
    <bpmn:sequenceFlow id="f12" sourceRef="callChild" targetRef="accountantReview"/>
    <bpmn:sequenceFlow id="f13" sourceRef="accountantReview" targetRef="waitForDocuments"/>
    <bpmn:sequenceFlow id="f14a" sourceRef="waitForDocuments" targetRef="receiveDocuments"/>
    <bpmn:sequenceFlow id="f14b" sourceRef="waitForDocuments" targetRef="receiveBudgetConfirmation"/>
    <bpmn:sequenceFlow id="f15a" sourceRef="receiveDocuments" targetRef="reviewerApproval"/>
    <bpmn:sequenceFlow id="f15b" sourceRef="receiveBudgetConfirmation" targetRef="budgetEnd"/>
    <bpmn:sequenceFlow id="f17" sourceRef="reviewerApproval" targetRef="approvedEnd"/>
    <bpmn:sequenceFlow id="f18" sourceRef="documentTimer" targetRef="timerManualFollowup"/>
    <bpmn:sequenceFlow id="f19" sourceRef="timerManualFollowup" targetRef="notifyApproval"/>
    <bpmn:sequenceFlow id="f20" sourceRef="notifyApproval" targetRef="timeoutEnd"/>`),
            `<bpmn:message id="Message_Documents" name="DocumentsReceived"/><bpmn:message id="Message_Budget" name="BudgetConfirmed"/><bpmn:signal id="Signal_Approved" name="InvoiceApproved"/>`,
          ),
        },
      };
    }
    case "userTaskApproval": {
      const processId = id("UserTaskApproval");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT User Task Approval", `
    <bpmn:startEvent id="start" name="Start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:userTask id="reviewTask" name="Review request" flowgo:assignee="admin"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:userTask>
    <bpmn:endEvent id="end" name="Done"><bpmn:incoming>f2</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="reviewTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="reviewTask" targetRef="end"/>`)),
        },
      };
    }
    case "parserElementMatrix": {
      const processId = id("ParserMatrix");
      return {
        primary: {
          processId,
          xml: doc(
            processId,
            process(processId, "UAT Parser Element Matrix", `
    <bpmn:startEvent id="start" name="Start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:serviceTask id="service" name="Service" flowgo:taskType="uat-parser-service"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:serviceTask>
    <bpmn:scriptTask id="script" name="Script"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f3</bpmn:outgoing></bpmn:scriptTask>
    <bpmn:userTask id="user" name="User"><bpmn:incoming>f3</bpmn:incoming><bpmn:outgoing>f4</bpmn:outgoing></bpmn:userTask>
    <bpmn:manualTask id="manual" name="Manual"><bpmn:incoming>f4</bpmn:incoming><bpmn:outgoing>f5</bpmn:outgoing></bpmn:manualTask>
    <bpmn:businessRuleTask id="rule" name="Rule" flowgo:decisionRef="decision_1"><bpmn:incoming>f5</bpmn:incoming><bpmn:outgoing>f6</bpmn:outgoing></bpmn:businessRuleTask>
    <bpmn:exclusiveGateway id="exclusive" name="Exclusive"><bpmn:incoming>f6</bpmn:incoming><bpmn:outgoing>f7</bpmn:outgoing></bpmn:exclusiveGateway>
    <bpmn:parallelGateway id="parallel" name="Parallel"><bpmn:incoming>f7</bpmn:incoming><bpmn:outgoing>f8</bpmn:outgoing></bpmn:parallelGateway>
    <bpmn:inclusiveGateway id="inclusive" name="Inclusive"><bpmn:incoming>f8</bpmn:incoming><bpmn:outgoing>f9</bpmn:outgoing></bpmn:inclusiveGateway>
    <bpmn:subProcess id="sub" name="Sub Process"><bpmn:incoming>f9</bpmn:incoming><bpmn:outgoing>f10</bpmn:outgoing></bpmn:subProcess>
    <bpmn:callActivity id="call" name="Call" calledElement="${id("Child")}"><bpmn:incoming>f10</bpmn:incoming><bpmn:outgoing>f11</bpmn:outgoing></bpmn:callActivity>
    <bpmn:receiveTask id="receive" name="Receive" messageRef="Message_UAT"><bpmn:incoming>f11</bpmn:incoming><bpmn:outgoing>f12</bpmn:outgoing></bpmn:receiveTask>
    <bpmn:eventBasedGateway id="eventGateway" name="Event Gateway"><bpmn:incoming>f12</bpmn:incoming><bpmn:outgoing>f13</bpmn:outgoing></bpmn:eventBasedGateway>
    <bpmn:intermediateCatchEvent id="catchSignal" name="Catch Signal"><bpmn:incoming>f13</bpmn:incoming><bpmn:outgoing>f14</bpmn:outgoing><bpmn:signalEventDefinition signalRef="Signal_UAT"/></bpmn:intermediateCatchEvent>
    <bpmn:intermediateThrowEvent id="throwMessage" name="Throw Message"><bpmn:incoming>f14</bpmn:incoming><bpmn:outgoing>f15</bpmn:outgoing><bpmn:messageEventDefinition messageRef="Message_UAT"/></bpmn:intermediateThrowEvent>
    <bpmn:boundaryEvent id="boundaryTimer" attachedToRef="user" cancelActivity="false"><bpmn:timerEventDefinition><bpmn:timeDuration>PT1H</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:boundaryEvent>
    <bpmn:endEvent id="end" name="End"><bpmn:incoming>f15</bpmn:incoming></bpmn:endEvent>
    ${Array.from({ length: 15 }, (_, i) => `<bpmn:sequenceFlow id="f${i + 1}" sourceRef="${[
      "start",
      "service",
      "script",
      "user",
      "manual",
      "rule",
      "exclusive",
      "parallel",
      "inclusive",
      "sub",
      "call",
      "receive",
      "eventGateway",
      "catchSignal",
      "throwMessage",
    ][i]}" targetRef="${[
      "service",
      "script",
      "user",
      "manual",
      "rule",
      "exclusive",
      "parallel",
      "inclusive",
      "sub",
      "call",
      "receive",
      "eventGateway",
      "catchSignal",
      "throwMessage",
      "end",
    ][i]}"/>`).join("\n    ")}`),
            `<bpmn:message id="Message_UAT" name="UATMessage"/><bpmn:signal id="Signal_UAT" name="UATSignal"/>`,
          ),
        },
        dependencies: [childProcess(id("Child"))],
      };
    }
    case "extensionProperties": {
      const processId = id("ExtensionProperties");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Extension Properties", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:userTask id="assignedUser" name="Assigned User" flowgo:assignee="admin" flowgo:candidateGroups="ops"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:userTask>
    <bpmn:serviceTask id="externalService" name="External Service" flowgo:taskType="uat-extension-service"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f3</bpmn:outgoing></bpmn:serviceTask>
    <bpmn:endEvent id="end"><bpmn:incoming>f3</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="assignedUser"/>
    <bpmn:sequenceFlow id="f2" sourceRef="assignedUser" targetRef="externalService"/>
    <bpmn:sequenceFlow id="f3" sourceRef="externalService" targetRef="end"/>`)),
        },
      };
    }
    case "gatewayJoins": {
      const processId = id("GatewayJoins");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Gateway Joins", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:parallelGateway id="split"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing><bpmn:outgoing>f3</bpmn:outgoing></bpmn:parallelGateway>
    <bpmn:userTask id="taskA"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f4</bpmn:outgoing></bpmn:userTask>
    <bpmn:userTask id="taskB"><bpmn:incoming>f3</bpmn:incoming><bpmn:outgoing>f5</bpmn:outgoing></bpmn:userTask>
    <bpmn:inclusiveGateway id="join"><bpmn:incoming>f4</bpmn:incoming><bpmn:incoming>f5</bpmn:incoming><bpmn:outgoing>f6</bpmn:outgoing></bpmn:inclusiveGateway>
    <bpmn:endEvent id="end"><bpmn:incoming>f6</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="split"/>
    <bpmn:sequenceFlow id="f2" sourceRef="split" targetRef="taskA"/>
    <bpmn:sequenceFlow id="f3" sourceRef="split" targetRef="taskB"/>
    <bpmn:sequenceFlow id="f4" sourceRef="taskA" targetRef="join"/>
    <bpmn:sequenceFlow id="f5" sourceRef="taskB" targetRef="join"/>
    <bpmn:sequenceFlow id="f6" sourceRef="join" targetRef="end"/>`)),
        },
      };
    }
    case "callBusinessManual": {
      const childId = id("ChildProcess");
      const parentId = id("CallBusinessManual");
      return {
        dependencies: [childProcess(childId)],
        primary: {
          processId: parentId,
          xml: doc(parentId, process(parentId, "UAT Call Business Manual", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:businessRuleTask id="rule" name="Rule" flowgo:decisionRef="decision_1"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:businessRuleTask>
    <bpmn:callActivity id="callChild" name="Call Child" calledElement="${childId}"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f3</bpmn:outgoing></bpmn:callActivity>
    <bpmn:manualTask id="manualReview" name="Manual Review"><bpmn:incoming>f3</bpmn:incoming><bpmn:outgoing>f4</bpmn:outgoing></bpmn:manualTask>
    <bpmn:endEvent id="end"><bpmn:incoming>f4</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="rule"/>
    <bpmn:sequenceFlow id="f2" sourceRef="rule" targetRef="callChild"/>
    <bpmn:sequenceFlow id="f3" sourceRef="callChild" targetRef="manualReview"/>
    <bpmn:sequenceFlow id="f4" sourceRef="manualReview" targetRef="end"/>`)),
        },
      };
    }
    case "eventGateway": {
      const processId = id("EventGateway");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Event Gateway", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:eventBasedGateway id="gateway"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing><bpmn:outgoing>f3</bpmn:outgoing></bpmn:eventBasedGateway>
    <bpmn:receiveTask id="receive" messageRef="Message_Payment"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f4</bpmn:outgoing></bpmn:receiveTask>
    <bpmn:intermediateCatchEvent id="timerCatch"><bpmn:incoming>f3</bpmn:incoming><bpmn:outgoing>f5</bpmn:outgoing><bpmn:timerEventDefinition><bpmn:timeDuration>PT1H</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:intermediateCatchEvent>
    <bpmn:manualTask id="afterReceive"><bpmn:incoming>f4</bpmn:incoming><bpmn:outgoing>f6</bpmn:outgoing></bpmn:manualTask>
    <bpmn:manualTask id="afterTimer"><bpmn:incoming>f5</bpmn:incoming><bpmn:outgoing>f7</bpmn:outgoing></bpmn:manualTask>
    <bpmn:endEvent id="end"><bpmn:incoming>f6</bpmn:incoming><bpmn:incoming>f7</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="gateway"/>
    <bpmn:sequenceFlow id="f2" sourceRef="gateway" targetRef="receive"/>
    <bpmn:sequenceFlow id="f3" sourceRef="gateway" targetRef="timerCatch"/>
    <bpmn:sequenceFlow id="f4" sourceRef="receive" targetRef="afterReceive"/>
    <bpmn:sequenceFlow id="f5" sourceRef="timerCatch" targetRef="afterTimer"/>
    <bpmn:sequenceFlow id="f6" sourceRef="afterReceive" targetRef="end"/>
    <bpmn:sequenceFlow id="f7" sourceRef="afterTimer" targetRef="end"/>`), `<bpmn:message id="Message_Payment" name="MsgPaymentReceived"/>`),
        },
      };
    }
    case "boundaryTimer": {
      const processId = id("BoundaryTimer");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Boundary Timer", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:userTask id="userTask"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:userTask>
    <bpmn:boundaryEvent id="timerBoundary" attachedToRef="userTask" cancelActivity="true"><bpmn:outgoing>f3</bpmn:outgoing><bpmn:timerEventDefinition><bpmn:timeDuration>PT0S</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:boundaryEvent>
    <bpmn:scriptTask id="timeoutTask"><bpmn:incoming>f3</bpmn:incoming><bpmn:outgoing>f4</bpmn:outgoing></bpmn:scriptTask>
    <bpmn:endEvent id="normalEnd"><bpmn:incoming>f2</bpmn:incoming></bpmn:endEvent>
    <bpmn:endEvent id="timeoutEnd"><bpmn:incoming>f4</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="userTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="userTask" targetRef="normalEnd"/>
    <bpmn:sequenceFlow id="f3" sourceRef="timerBoundary" targetRef="timeoutTask"/>
    <bpmn:sequenceFlow id="f4" sourceRef="timeoutTask" targetRef="timeoutEnd"/>`)),
        },
      };
    }
    case "messageSignal": {
      const receiverId = id("SignalReceiver");
      const senderId = id("SignalSender");
      return {
        dependencies: [{
          processId: receiverId,
          xml: doc(receiverId, process(receiverId, "UAT Signal Receiver", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:intermediateCatchEvent id="catchSignal"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing><bpmn:signalEventDefinition signalRef="Signal_Global"/></bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"><bpmn:incoming>f2</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="catchSignal"/>
    <bpmn:sequenceFlow id="f2" sourceRef="catchSignal" targetRef="end"/>`), `<bpmn:signal id="Signal_Global" name="GlobalSignal"/>`),
        }],
        primary: {
          processId: senderId,
          xml: doc(senderId, process(senderId, "UAT Signal Sender", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:intermediateThrowEvent id="throwSignal"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing><bpmn:signalEventDefinition signalRef="Signal_Global"/></bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"><bpmn:incoming>f2</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="throwSignal"/>
    <bpmn:sequenceFlow id="f2" sourceRef="throwSignal" targetRef="end"/>`), `<bpmn:signal id="Signal_Global" name="GlobalSignal"/>`),
        },
      };
    }
    case "serviceUserAssignment": {
      const processId = id("ServiceUserAssignment");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Service User Assignment", `
    <bpmn:startEvent id="start"><bpmn:outgoing>f1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:serviceTask id="externalService" name="External Service" flowgo:taskType="uat-worker"><bpmn:incoming>f1</bpmn:incoming><bpmn:outgoing>f2</bpmn:outgoing></bpmn:serviceTask>
    <bpmn:userTask id="assignedUser" name="Assigned User" flowgo:assignee="admin" flowgo:candidateGroups="ops"><bpmn:incoming>f2</bpmn:incoming><bpmn:outgoing>f3</bpmn:outgoing></bpmn:userTask>
    <bpmn:endEvent id="end"><bpmn:incoming>f3</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="externalService"/>
    <bpmn:sequenceFlow id="f2" sourceRef="externalService" targetRef="assignedUser"/>
    <bpmn:sequenceFlow id="f3" sourceRef="assignedUser" targetRef="end"/>`)),
        },
      };
    }
    case "unsupportedSendTask": {
      const processId = id("UnsupportedSend");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Unsupported Send", `
    <bpmn:startEvent id="start"/>
    <bpmn:sendTask id="send"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="send"/>
    <bpmn:sequenceFlow id="f2" sourceRef="send" targetRef="end"/>`)),
        },
      };
    }
    case "unsupportedElement": {
      const processId = id("UnsupportedElement");
      return {
        primary: {
          processId,
          xml: doc(processId, process(processId, "UAT Unsupported Element", `
    <bpmn:startEvent id="start"/>
    <bpmn:adHocSubProcess id="adhoc"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="adhoc"/>
    <bpmn:sequenceFlow id="f2" sourceRef="adhoc" targetRef="end"/>`)),
        },
      };
    }
  }
}

function childProcess(processId: string): BpmnDocument {
  return {
    processId,
    xml: doc(processId, process(processId, "UAT Child Process", `
    <bpmn:startEvent id="childStart"><bpmn:outgoing>cf1</bpmn:outgoing></bpmn:startEvent>
    <bpmn:endEvent id="childEnd"><bpmn:incoming>cf1</bpmn:incoming></bpmn:endEvent>
    <bpmn:sequenceFlow id="cf1" sourceRef="childStart" targetRef="childEnd"/>`)),
  };
}
