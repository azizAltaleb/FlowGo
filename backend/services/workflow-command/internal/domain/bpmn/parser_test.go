package bpmn

import (
	"github.com/artificialflow/artificialflow/backend/libs/model"
	"strings"
	"testing"
)

func TestNormalizeConditionExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "xml escaped operator", input: "amount &lt; 100", want: "amount < 100"},
		{name: "cdata and dollar wrapper", input: "<![CDATA[${amount >= 100}]]>", want: "amount >= 100"},
		{name: "hash wrapper", input: "#{isApproved}", want: "isApproved"},
		{name: "trim whitespace", input: "  amount == 10  ", want: "amount == 10"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeConditionExpression(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeConditionExpression(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParse_NormalizesSequenceFlowConditionExpression(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Cond" name="Condition Process" isExecutable="true">
    <bpmn:startEvent id="start" name="Start"/>
    <bpmn:exclusiveGateway id="xor" name="XOR"/>
    <bpmn:userTask id="taskA" name="Task A"/>
    <bpmn:userTask id="taskB" name="Task B"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="xor"/>
    <bpmn:sequenceFlow id="f2" sourceRef="xor" targetRef="taskA">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">amount &lt; 100</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f3" sourceRef="xor" targetRef="taskB">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression"><![CDATA[${amount >= 100}]]></bpmn:conditionExpression>
    </bpmn:sequenceFlow>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var xorStepFound bool
	var condA, condB string
	for _, step := range wf.Steps {
		if step.ID != "xor" {
			continue
		}
		xorStepFound = true
		for _, out := range step.Outgoing {
			switch out.TargetRef {
			case "taskA":
				condA = out.Condition
			case "taskB":
				condB = out.Condition
			}
		}
	}

	if !xorStepFound {
		t.Fatalf("expected xor step to be parsed")
	}
	if condA != "amount < 100" {
		t.Fatalf("unexpected condition for taskA: got %q", condA)
	}
	if condB != "amount >= 100" {
		t.Fatalf("unexpected condition for taskB: got %q", condB)
	}
}

func TestParse_MapsExtendedElementsAndProperties(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_OrderPlaced" name="OrderPlaced"/>
  <bpmn:error id="Error_Business" errorCode="BUSINESS_ERROR"/>
  <bpmn:process id="Process_Extended" name="Extended Process" isExecutable="true">
    <bpmn:startEvent id="start" name="Start"/>
    <bpmn:eventBasedGateway id="eventGateway" name="Event Gateway"/>
    <bpmn:receiveTask id="receive" name="Receive" messageRef="Message_OrderPlaced" artificialflow:correlationKey="orderId"/>
    <bpmn:intermediateCatchEvent id="timerCatch" name="Wait Timer">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:manualTask id="manual" name="Manual Review"/>
    <bpmn:businessRuleTask id="rule" name="Decision" artificialflow:decisionRef="decision_table" artificialflow:resultVariable="decisionResult"/>
    <bpmn:callActivity id="call" name="Call Child" calledElement="ChildProcess"/>
    <bpmn:userTask id="user" name="User Task" artificialflow:assignee="alice" artificialflow:candidateUsers="bob,charlie" artificialflow:candidateGroups="ops" artificialflow:dueDate="2026-01-01T00:00:00Z"/>
    <bpmn:boundaryEvent id="boundaryTimer" attachedToRef="user" cancelActivity="false">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT5M</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:intermediateThrowEvent id="throwMessage" name="Throw Message">
      <bpmn:messageEventDefinition messageRef="Message_OrderPlaced"/>
    </bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="endError" name="Error End">
      <bpmn:errorEventDefinition errorRef="Error_Business"/>
    </bpmn:endEvent>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="eventGateway"/>
    <bpmn:sequenceFlow id="f2" sourceRef="eventGateway" targetRef="receive"/>
    <bpmn:sequenceFlow id="f3" sourceRef="eventGateway" targetRef="timerCatch"/>
    <bpmn:sequenceFlow id="f4" sourceRef="receive" targetRef="manual"/>
    <bpmn:sequenceFlow id="f5" sourceRef="timerCatch" targetRef="manual"/>
    <bpmn:sequenceFlow id="f6" sourceRef="manual" targetRef="rule"/>
    <bpmn:sequenceFlow id="f7" sourceRef="rule" targetRef="call"/>
    <bpmn:sequenceFlow id="f8" sourceRef="call" targetRef="user"/>
    <bpmn:sequenceFlow id="f9" sourceRef="user" targetRef="endError"/>
    <bpmn:sequenceFlow id="f10" sourceRef="boundaryTimer" targetRef="throwMessage"/>
    <bpmn:sequenceFlow id="f11" sourceRef="throwMessage" targetRef="endError"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	steps := make(map[string]model.StepDefinition)
	for _, step := range wf.Steps {
		steps[step.ID] = step
	}

	if steps["eventGateway"].Type != model.StepTypeGatewayEventBased {
		t.Fatalf("eventGateway type mismatch: got %s", steps["eventGateway"].Type)
	}

	receive := steps["receive"]
	if receive.Type != model.StepTypeReceiveTask {
		t.Fatalf("receive type mismatch: got %s", receive.Type)
	}
	if receive.Properties["message_ref"] != "OrderPlaced" {
		t.Fatalf("receive.message_ref mismatch: got %v", receive.Properties["message_ref"])
	}
	if receive.Properties["correlation_key"] != "orderId" {
		t.Fatalf("receive.correlation_key mismatch: got %v", receive.Properties["correlation_key"])
	}

	timerCatch := steps["timerCatch"]
	if timerCatch.Type != model.StepTypeIntermediateTimerCatchEvent {
		t.Fatalf("timerCatch type mismatch: got %s", timerCatch.Type)
	}
	if timerCatch.Properties["timer_duration"] != "PT1S" {
		t.Fatalf("timerCatch.timer_duration mismatch: got %v", timerCatch.Properties["timer_duration"])
	}

	if steps["manual"].Type != model.StepTypeManualTask {
		t.Fatalf("manual type mismatch: got %s", steps["manual"].Type)
	}

	rule := steps["rule"]
	if rule.Type != model.StepTypeBusinessRuleTask {
		t.Fatalf("rule type mismatch: got %s", rule.Type)
	}
	if rule.Properties["decision_ref"] != "decision_table" {
		t.Fatalf("rule.decision_ref mismatch: got %v", rule.Properties["decision_ref"])
	}

	call := steps["call"]
	if call.Type != model.StepTypeCallActivity {
		t.Fatalf("call type mismatch: got %s", call.Type)
	}
	if call.Properties["called_element"] != "ChildProcess" {
		t.Fatalf("call.called_element mismatch: got %v", call.Properties["called_element"])
	}

	user := steps["user"]
	if user.Type != model.StepTypeUserTask {
		t.Fatalf("user type mismatch: got %s", user.Type)
	}
	if user.Properties["assignee"] != "alice" {
		t.Fatalf("user.assignee mismatch: got %v", user.Properties["assignee"])
	}
	if len(user.BoundaryEventRefs) != 1 || user.BoundaryEventRefs[0] != "boundaryTimer" {
		t.Fatalf("user.BoundaryEventRefs mismatch: got %v", user.BoundaryEventRefs)
	}

	boundary := steps["boundaryTimer"]
	if boundary.Type != model.StepTypeBoundaryEvent {
		t.Fatalf("boundaryTimer type mismatch: got %s", boundary.Type)
	}
	if boundary.Properties["attached_to"] != "user" {
		t.Fatalf("boundaryTimer.attached_to mismatch: got %v", boundary.Properties["attached_to"])
	}
	if boundary.Properties["cancel_activity"] != false {
		t.Fatalf("boundaryTimer.cancel_activity mismatch: got %v", boundary.Properties["cancel_activity"])
	}
	if boundary.Properties["timer_duration"] != "PT5M" {
		t.Fatalf("boundaryTimer.timer_duration mismatch: got %v", boundary.Properties["timer_duration"])
	}

	throwMessage := steps["throwMessage"]
	if throwMessage.Type != model.StepTypeIntermediateThrowEvent {
		t.Fatalf("throwMessage type mismatch: got %s", throwMessage.Type)
	}
	if throwMessage.Properties["message_ref"] != "OrderPlaced" {
		t.Fatalf("throwMessage.message_ref mismatch: got %v", throwMessage.Properties["message_ref"])
	}

	endError := steps["endError"]
	if endError.Type != model.StepTypeEnd {
		t.Fatalf("endError type mismatch: got %s", endError.Type)
	}
	if endError.Properties["error_code"] != "BUSINESS_ERROR" {
		t.Fatalf("endError.error_code mismatch: got %v", endError.Properties["error_code"])
	}

	manual := steps["manual"]
	if len(manual.Incoming) != 2 || !containsString(manual.Incoming, "receive") || !containsString(manual.Incoming, "timerCatch") {
		t.Fatalf("manual incoming mismatch: got %v", manual.Incoming)
	}
}

func TestParse_ArtificialFlowNamespaceCompatibilityAndPrecedence(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions
  xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn"
  id="Definitions_ArtificialFlow_Attributes"
  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ArtificialFlow_Attributes" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="canonical" artificialflow:taskType="canonical-worker" taskType="plain-worker">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="source" value="canonical"/>
        </artificialflow:properties>
        <properties>
          <property name="source" value="plain"/>
        </properties>
      </bpmn:extensionElements>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="legacy" taskType="legacy-worker"/>
    <bpmn:serviceTask id="plain" taskType="plain-worker"/>
    <bpmn:userTask id="user" artificialflow:assignee="canonical-user" assignee="plain-user"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="canonical"/>
    <bpmn:sequenceFlow id="f2" sourceRef="canonical" targetRef="legacy"/>
    <bpmn:sequenceFlow id="f3" sourceRef="legacy" targetRef="plain"/>
    <bpmn:sequenceFlow id="f4" sourceRef="plain" targetRef="user"/>
    <bpmn:sequenceFlow id="f5" sourceRef="user" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	steps := make(map[string]model.StepDefinition)
	for _, step := range wf.Steps {
		steps[step.ID] = step
	}

	if steps["canonical"].Implementation != "canonical-worker" {
		t.Fatalf("expected artificialflow task type precedence, got %q", steps["canonical"].Implementation)
	}
	if steps["canonical"].Properties["source"] != "canonical" {
		t.Fatalf("expected artificialflow extension precedence, got %v", steps["canonical"].Properties["source"])
	}
	if steps["legacy"].Implementation != "legacy-worker" {
		t.Fatalf("expected plain taskType attribute, got %q", steps["legacy"].Implementation)
	}
	if steps["plain"].Implementation != "plain-worker" {
		t.Fatalf("expected unqualified task type, got %q", steps["plain"].Implementation)
	}
	if steps["user"].Properties["assignee"] != "canonical-user" {
		t.Fatalf("expected artificialflow assignee precedence, got %v", steps["user"].Properties["assignee"])
	}
}

func TestParse_PopulatesIncomingForGatewayJoin(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Join" name="Join Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:parallelGateway id="split"/>
    <bpmn:userTask id="taskA"/>
    <bpmn:userTask id="taskB"/>
    <bpmn:parallelGateway id="join"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="split"/>
    <bpmn:sequenceFlow id="f2" sourceRef="split" targetRef="taskA"/>
    <bpmn:sequenceFlow id="f3" sourceRef="split" targetRef="taskB"/>
    <bpmn:sequenceFlow id="f4" sourceRef="taskA" targetRef="join"/>
    <bpmn:sequenceFlow id="f5" sourceRef="taskB" targetRef="join"/>
    <bpmn:sequenceFlow id="f6" sourceRef="join" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var join *model.StepDefinition
	for i := range wf.Steps {
		if wf.Steps[i].ID == "join" {
			join = &wf.Steps[i]
			break
		}
	}
	if join == nil {
		t.Fatalf("expected join step to be parsed")
	}

	if len(join.Incoming) != 2 {
		t.Fatalf("expected 2 incoming sources for join, got %d (%v)", len(join.Incoming), join.Incoming)
	}
	if !containsString(join.Incoming, "taskA") || !containsString(join.Incoming, "taskB") {
		t.Fatalf("join incoming mismatch: %v", join.Incoming)
	}
}

func TestParse_FailsForUnsupportedElementReferences(t *testing.T) {
	cases := []struct {
		name   string
		xml    string
		flowID string
		target string
	}{
		{
			name: "adHocSubProcess",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Unsupported" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Unsupported" name="Unsupported Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:adHocSubProcess id="adhoc"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="adhoc"/>
    <bpmn:sequenceFlow id="f2" sourceRef="adhoc" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`,
			flowID: "f1",
			target: "adhoc",
		},
		{
			name: "complexGateway",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_ComplexGW" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ComplexGW" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:complexGateway id="cgw"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="cgw"/>
    <bpmn:sequenceFlow id="f2" sourceRef="cgw" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`,
			flowID: "f1",
			target: "cgw",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.xml))
			if err == nil {
				t.Fatalf("expected Parse to fail for unsupported %s reference", tc.name)
			}
			if !strings.Contains(err.Error(), "sequenceFlow "+tc.flowID) {
				t.Fatalf("expected sequenceFlow id in error, got: %v", err)
			}
			if !strings.Contains(err.Error(), "targetRef="+tc.target) {
				t.Fatalf("expected missing targetRef=%s in error, got: %v", tc.target, err)
			}
		})
	}
}

func TestParse_SendTaskDefaultsToHTTPConnector(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_SendDefault" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_SendDefault" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:sendTask id="send" name="Notify"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="send"/>
    <bpmn:sequenceFlow id="f2" sourceRef="send" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, step := range wf.Steps {
		if step.ID == "send" {
			if step.Implementation != "io.artificialflow.connector.http" {
				t.Fatalf("expected default HTTP connector, got %q", step.Implementation)
			}
			return
		}
	}
	t.Fatalf("send task not found")
}

func TestParse_SupportsSendTask(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="Definitions_Send" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Send" name="Send Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:sendTask id="send" name="Notify" artificialflow:taskType="io.artificialflow.connector.send"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="send"/>
    <bpmn:sequenceFlow id="f2" sourceRef="send" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("expected Parse to succeed for sendTask, got: %v", err)
	}
	found := false
	for _, step := range wf.Steps {
		if step.ID == "send" {
			found = true
			if step.Type != model.StepTypeSendTask {
				t.Fatalf("expected SEND_TASK, got %s", step.Type)
			}
			if step.Implementation != "io.artificialflow.connector.send" {
				t.Fatalf("unexpected implementation %q", step.Implementation)
			}
		}
	}
	if !found {
		t.Fatalf("send step not found")
	}
}

func TestParse_ElementTypeMatrix(t *testing.T) {
	wrap := func(extraDefs, processBody string) string {
		return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="Definitions_Matrix" targetNamespace="http://bpmn.io/schema/bpmn">` + extraDefs + `
  <bpmn:process id="Process_Matrix" name="Matrix" isExecutable="true">` + processBody + `
  </bpmn:process>
</bpmn:definitions>`
	}

	tests := []struct {
		name      string
		extraDefs string
		body      string
		stepID    string
		stepType  model.StepType
	}{
		{
			name: "service task",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="target" artificialflow:taskType="svc"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeServiceTask,
		},
		{
			name: "user task",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:userTask id="target" artificialflow:assignee="alice"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeUserTask,
		},
		{
			name: "script task",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:scriptTask id="target" scriptFormat="javascript"><bpmn:script>1+1</bpmn:script></bpmn:scriptTask>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeScriptTask,
		},
		{
			name:      "receive task",
			extraDefs: `<bpmn:message id="Message_1" name="Message1"/>`,
			body: `<bpmn:startEvent id="start"/>
    <bpmn:receiveTask id="target" messageRef="Message_1"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeReceiveTask,
		},
		{
			name: "manual task",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:manualTask id="target"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeManualTask,
		},
		{
			name: "business rule task",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:businessRuleTask id="target" artificialflow:decisionRef="decision_1"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeBusinessRuleTask,
		},
		{
			name: "call activity",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:callActivity id="target" calledElement="ChildProcess"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeCallActivity,
		},
		{
			name: "intermediate catch event",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="target"><bpmn:signalEventDefinition signalRef="Signal_1"/></bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeIntermediateCatchEvent,
		},
		{
			name: "intermediate timer catch event",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="target"><bpmn:timerEventDefinition><bpmn:timeDuration>PT1S</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeIntermediateTimerCatchEvent,
		},
		{
			name: "intermediate throw event",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:intermediateThrowEvent id="target"><bpmn:messageEventDefinition messageRef="Message_1"/></bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			extraDefs: `<bpmn:message id="Message_1" name="Message1"/>`,
			stepID:    "target",
			stepType:  model.StepTypeIntermediateThrowEvent,
		},
		{
			name: "boundary event",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:userTask id="task"/>
    <bpmn:boundaryEvent id="target" attachedToRef="task"><bpmn:timerEventDefinition><bpmn:timeDuration>PT1S</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:boundaryEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="task"/>
    <bpmn:sequenceFlow id="f2" sourceRef="task" targetRef="end"/>
    <bpmn:sequenceFlow id="f3" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeBoundaryEvent,
		},
		{
			name: "exclusive gateway",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:exclusiveGateway id="target" default="f2"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeGatewayExclusive,
		},
		{
			name: "parallel gateway",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:parallelGateway id="target"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeGatewayParallel,
		},
		{
			name: "inclusive gateway",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:inclusiveGateway id="target" default="f2"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeGatewayInclusive,
		},
		{
			name: "event based gateway",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:eventBasedGateway id="target"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeGatewayEventBased,
		},
		{
			name: "sub process",
			body: `<bpmn:startEvent id="start"/>
    <bpmn:subProcess id="target"><bpmn:startEvent id="subStart"/><bpmn:endEvent id="subEnd"/><bpmn:sequenceFlow id="sf1" sourceRef="subStart" targetRef="subEnd"/></bpmn:subProcess>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="target"/>
    <bpmn:sequenceFlow id="f2" sourceRef="target" targetRef="end"/>`,
			stepID:   "target",
			stepType: model.StepTypeSubProcess,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wf, err := Parse(strings.NewReader(wrap(tc.extraDefs, tc.body)))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			var found *model.StepDefinition
			for i := range wf.Steps {
				if wf.Steps[i].ID == tc.stepID {
					found = &wf.Steps[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("expected step %s to be parsed", tc.stepID)
			}
			if found.Type != tc.stepType {
				t.Fatalf("step %s type mismatch: got %s, want %s", tc.stepID, found.Type, tc.stepType)
			}
		})
	}
}

func TestParse_MapsPlainAttributeAliasesForCompatibility(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Alias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_Alias" name="AliasMessage"/>
  <bpmn:error id="Error_Alias" errorCode="ALIAS_ERR"/>
  <bpmn:signal id="Signal_Alias" name="AliasSignal"/>
  <bpmn:process id="Process_Alias" name="Alias Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="service" topic="legacyTopic" taskType="legacyTaskType"/>
    <bpmn:userTask id="user" candidateUsers="u1,u2" candidateGroups="g1" dueDate="PT1H"/>
    <bpmn:receiveTask id="receive" messageRef="Message_Alias" correlationKey="orderId"/>
    <bpmn:intermediateCatchEvent id="timerCatch" timerDuration="PT2S"/>
    <bpmn:boundaryEvent id="boundary" attachedToRef="user" timerDuration="PT3S" correlationKey="customerId" errorCode="ALIAS_ERR" errorMessage="legacy boundary">
      <bpmn:signalEventDefinition signalRef="Signal_Alias"/>
    </bpmn:boundaryEvent>
    <bpmn:intermediateThrowEvent id="throwEvent" correlationKey="legacyKey" errorCode="ALIAS_ERR" errorMessage="legacy throw">
      <bpmn:errorEventDefinition errorRef="Error_Alias"/>
    </bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="service"/>
    <bpmn:sequenceFlow id="f2" sourceRef="service" targetRef="user"/>
    <bpmn:sequenceFlow id="f3" sourceRef="user" targetRef="receive"/>
    <bpmn:sequenceFlow id="f4" sourceRef="receive" targetRef="timerCatch"/>
    <bpmn:sequenceFlow id="f5" sourceRef="timerCatch" targetRef="throwEvent"/>
    <bpmn:sequenceFlow id="f6" sourceRef="throwEvent" targetRef="end"/>
    <bpmn:sequenceFlow id="f7" sourceRef="boundary" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	steps := make(map[string]model.StepDefinition)
	for _, step := range wf.Steps {
		steps[step.ID] = step
	}

	if steps["service"].Implementation != "legacyTaskType" {
		t.Fatalf("expected service implementation from plain taskType, got %q", steps["service"].Implementation)
	}
	if steps["service"].Properties["topic"] != "legacyTopic" {
		t.Fatalf("expected service.topic alias mapping, got %v", steps["service"].Properties["topic"])
	}

	if steps["user"].Properties["candidate_users"] != "u1,u2" {
		t.Fatalf("expected user candidate_users alias mapping, got %v", steps["user"].Properties["candidate_users"])
	}
	if steps["user"].Properties["candidate_groups"] != "g1" {
		t.Fatalf("expected user candidate_groups alias mapping, got %v", steps["user"].Properties["candidate_groups"])
	}
	if steps["user"].Properties["due_date"] != "PT1H" {
		t.Fatalf("expected user due_date alias mapping, got %v", steps["user"].Properties["due_date"])
	}

	if steps["receive"].Properties["correlation_key"] != "orderId" {
		t.Fatalf("expected receive correlation_key alias mapping, got %v", steps["receive"].Properties["correlation_key"])
	}

	if steps["timerCatch"].Type != model.StepTypeIntermediateTimerCatchEvent {
		t.Fatalf("expected timerCatch type %s, got %s", model.StepTypeIntermediateTimerCatchEvent, steps["timerCatch"].Type)
	}
	if steps["timerCatch"].Properties["timer_duration"] != "PT2S" {
		t.Fatalf("expected timerCatch timer_duration alias mapping, got %v", steps["timerCatch"].Properties["timer_duration"])
	}

	if steps["boundary"].Properties["timer_duration"] != "PT3S" {
		t.Fatalf("expected boundary timer_duration alias mapping, got %v", steps["boundary"].Properties["timer_duration"])
	}
	if steps["boundary"].Properties["correlation_key"] != "customerId" {
		t.Fatalf("expected boundary correlation_key alias mapping, got %v", steps["boundary"].Properties["correlation_key"])
	}
	if steps["boundary"].Properties["error_code"] != "ALIAS_ERR" {
		t.Fatalf("expected boundary error_code alias mapping, got %v", steps["boundary"].Properties["error_code"])
	}
	if steps["boundary"].Properties["error_message"] != "legacy boundary" {
		t.Fatalf("expected boundary error_message alias mapping, got %v", steps["boundary"].Properties["error_message"])
	}

	if steps["throwEvent"].Properties["correlation_key"] != "legacyKey" {
		t.Fatalf("expected throwEvent correlation_key alias mapping, got %v", steps["throwEvent"].Properties["correlation_key"])
	}
	if steps["throwEvent"].Properties["error_code"] != "ALIAS_ERR" {
		t.Fatalf("expected throwEvent error_code alias mapping, got %v", steps["throwEvent"].Properties["error_code"])
	}
	if steps["throwEvent"].Properties["error_message"] != "legacy throw" {
		t.Fatalf("expected throwEvent error_message alias mapping, got %v", steps["throwEvent"].Properties["error_message"])
	}
}

func TestParse_MergesExtensionPropertiesWithoutOverridingMappedKeys(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="Definitions_Extensions" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Extensions" name="Extensions Process" isExecutable="true">
    <bpmn:startEvent id="start">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="start_flag" value="true"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
    </bpmn:startEvent>

    <bpmn:serviceTask id="service">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="taskType" value="ext_service_handler"/>
          <artificialflow:property name="service_custom_key" value="service_custom_value"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
    </bpmn:serviceTask>

    <bpmn:userTask id="user" artificialflow:assignee="alice">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="custom_note" value="vip_review"/>
          <artificialflow:property name="assignee" value="bob"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
    </bpmn:userTask>

    <bpmn:subProcess id="sub" name="Sub With Extensions">
      <bpmn:extensionElements>
        <properties>
          <property name="sub_plain" value="ok"/>
        </properties>
      </bpmn:extensionElements>

      <bpmn:startEvent id="subStart"/>
      <bpmn:endEvent id="subEnd"/>
      <bpmn:sequenceFlow id="sf1" sourceRef="subStart" targetRef="subEnd"/>
    </bpmn:subProcess>

    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="service"/>
    <bpmn:sequenceFlow id="f2" sourceRef="service" targetRef="user"/>
    <bpmn:sequenceFlow id="f3" sourceRef="user" targetRef="sub"/>
    <bpmn:sequenceFlow id="f4" sourceRef="sub" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	steps := make(map[string]model.StepDefinition)
	for _, step := range wf.Steps {
		steps[step.ID] = step
	}

	if steps["start"].Properties["start_flag"] != "true" {
		t.Fatalf("expected start extension property, got %v", steps["start"].Properties["start_flag"])
	}

	if steps["service"].Properties["service_custom_key"] != "service_custom_value" {
		t.Fatalf("expected service custom extension property, got %v", steps["service"].Properties["service_custom_key"])
	}
	if steps["service"].Implementation != "ext_service_handler" {
		t.Fatalf("expected service implementation from extension task_type, got %q", steps["service"].Implementation)
	}

	if steps["user"].Properties["custom_note"] != "vip_review" {
		t.Fatalf("expected user custom extension property, got %v", steps["user"].Properties["custom_note"])
	}
	if steps["user"].Properties["assignee"] != "alice" {
		t.Fatalf("expected mapped assignee to take precedence over extension property, got %v", steps["user"].Properties["assignee"])
	}

	if steps["sub"].Properties["sub_plain"] != "ok" {
		t.Fatalf("expected subprocess plain extension property, got %v", steps["sub"].Properties["sub_plain"])
	}
}

func TestParse_CanonicalizesExtensionPropertyAliases(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="Definitions_ExtAlias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ExtAlias" name="Extension Alias Process" isExecutable="true">
    <bpmn:startEvent id="start"/>

    <bpmn:userTask id="user">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="candidateUsers" value="u1,u2"/>
          <artificialflow:property name="dueDate" value="PT1H"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
    </bpmn:userTask>

    <bpmn:boundaryEvent id="boundary" attachedToRef="user">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="cancelActivity" value="false"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>

    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="user"/>
    <bpmn:sequenceFlow id="f2" sourceRef="user" targetRef="end"/>
    <bpmn:sequenceFlow id="f3" sourceRef="boundary" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	steps := make(map[string]model.StepDefinition)
	for _, step := range wf.Steps {
		steps[step.ID] = step
	}

	if steps["user"].Properties["candidate_users"] != "u1,u2" {
		t.Fatalf("expected candidateUsers extension alias to map to candidate_users, got %v", steps["user"].Properties["candidate_users"])
	}
	if steps["user"].Properties["due_date"] != "PT1H" {
		t.Fatalf("expected dueDate extension alias to map to due_date, got %v", steps["user"].Properties["due_date"])
	}

	cancelActivity, ok := steps["boundary"].Properties["cancel_activity"].(bool)
	if !ok {
		t.Fatalf("expected cancelActivity extension alias to coerce to bool, got %T (%v)", steps["boundary"].Properties["cancel_activity"], steps["boundary"].Properties["cancel_activity"])
	}
	if cancelActivity {
		t.Fatalf("expected cancel_activity=false from extension alias, got true")
	}
}

func TestParse_BoundaryCancelActivityAttributeTakesPrecedenceOverExtensionAlias(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="Definitions_BoundaryPrecedence" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_BoundaryPrecedence" name="Boundary Precedence" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="user"/>

    <bpmn:boundaryEvent id="boundary" attachedToRef="user" cancelActivity="true">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="cancelActivity" value="false"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>

    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="user"/>
    <bpmn:sequenceFlow id="f2" sourceRef="user" targetRef="end"/>
    <bpmn:sequenceFlow id="f3" sourceRef="boundary" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	steps := make(map[string]model.StepDefinition)
	for _, step := range wf.Steps {
		steps[step.ID] = step
	}

	cancelActivity, ok := steps["boundary"].Properties["cancel_activity"].(bool)
	if !ok {
		t.Fatalf("expected boundary cancel_activity bool, got %T (%v)", steps["boundary"].Properties["cancel_activity"], steps["boundary"].Properties["cancel_activity"])
	}
	if !cancelActivity {
		t.Fatalf("expected mapped cancelActivity attribute to take precedence over extension alias")
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestParse_MessageAndSignalStartProps(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Start" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_1" name="OrderPlaced"/>
  <bpmn:signal id="Signal_1" name="Go"/>
  <bpmn:process id="Process_Start" isExecutable="true">
    <bpmn:startEvent id="msgStart"><bpmn:messageEventDefinition messageRef="Message_1"/></bpmn:startEvent>
    <bpmn:startEvent id="sigStart"><bpmn:signalEventDefinition signalRef="Signal_1"/></bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="msgStart" targetRef="end"/>
    <bpmn:sequenceFlow id="f2" sourceRef="sigStart" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	byID := map[string]model.StepDefinition{}
	for _, s := range wf.Steps {
		byID[s.ID] = s
	}
	if byID["msgStart"].Properties["message_ref"] != "OrderPlaced" {
		t.Fatalf("message start props: %#v", byID["msgStart"].Properties)
	}
	if byID["sigStart"].Properties["signal_ref"] != "Go" {
		t.Fatalf("signal start props: %#v", byID["sigStart"].Properties)
	}
}

func TestParse_TerminateAndLink(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_TL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_TL" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateThrowEvent id="linkThrow"><bpmn:linkEventDefinition name="Jump"/></bpmn:intermediateThrowEvent>
    <bpmn:intermediateCatchEvent id="linkCatch"><bpmn:linkEventDefinition name="Jump"/></bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="term"><bpmn:terminateEventDefinition/></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="linkThrow"/>
    <bpmn:sequenceFlow id="f2" sourceRef="linkCatch" targetRef="term"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	byID := map[string]model.StepDefinition{}
	for _, s := range wf.Steps {
		byID[s.ID] = s
	}
	if byID["linkThrow"].Properties["link_name"] != "Jump" {
		t.Fatalf("link throw: %#v", byID["linkThrow"].Properties)
	}
	if byID["linkCatch"].Properties["link_name"] != "Jump" {
		t.Fatalf("link catch: %#v", byID["linkCatch"].Properties)
	}
	if byID["term"].Properties["event_definition_type"] != "terminate" {
		t.Fatalf("terminate: %#v", byID["term"].Properties)
	}
}

func TestParse_MultiInstance(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="D" targetNamespace="t">
  <bpmn:process id="MIParse" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="seqTask" artificialflow:collection="items" artificialflow:elementVariable="item">
      <bpmn:multiInstanceLoopCharacteristics isSequential="true"/>
    </bpmn:userTask>
    <bpmn:serviceTask id="parTask" artificialflow:taskType="work">
      <bpmn:multiInstanceLoopCharacteristics isSequential="false" collection="rows" elementVariable="row"/>
    </bpmn:serviceTask>
    <bpmn:callActivity id="call" calledElement="Child" artificialflow:calledElementVersion="2"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="seqTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="seqTask" targetRef="parTask"/>
    <bpmn:sequenceFlow id="f3" sourceRef="parTask" targetRef="call"/>
    <bpmn:sequenceFlow id="f4" sourceRef="call" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	byID := map[string]model.StepDefinition{}
	for _, s := range wf.Steps {
		byID[s.ID] = s
	}
	seq := byID["seqTask"]
	if seq.LoopType != "SEQUENTIAL" || seq.LoopCollection != "items" || seq.LoopElement != "item" {
		t.Fatalf("seqTask loop mismatch: type=%q collection=%q element=%q", seq.LoopType, seq.LoopCollection, seq.LoopElement)
	}
	par := byID["parTask"]
	if par.LoopType != "PARALLEL" || par.LoopCollection != "rows" || par.LoopElement != "row" {
		t.Fatalf("parTask loop mismatch: type=%q collection=%q element=%q", par.LoopType, par.LoopCollection, par.LoopElement)
	}
	call := byID["call"]
	if call.Properties["called_element_version"] != "2" {
		t.Fatalf("expected called_element_version=2, got %#v", call.Properties)
	}
}

func TestParse_IOMappingsAndListeners(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="D" targetNamespace="t">
  <bpmn:process id="IOParse" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="svc" artificialflow:taskType="work">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="input.localOrder" value="orderId"/>
          <artificialflow:property name="output.result" value="localResult"/>
          <artificialflow:property name="listener.create" value="onCreateHandler"/>
          <artificialflow:property name="listener.complete" value="onCompleteHandler"/>
        </artificialflow:properties>
        <artificialflow:ioMapping>
          <artificialflow:input target="fromXml" source="xmlSource"/>
          <artificialflow:output target="xmlOut" source="localOut"/>
        </artificialflow:ioMapping>
      </bpmn:extensionElements>
    </bpmn:serviceTask>
    <bpmn:userTask id="user">
      <bpmn:extensionElements>
        <artificialflow:properties>
          <artificialflow:property name="inputParameters" value='{"a":"b","c":"d"}'/>
          <artificialflow:property name="outputParameters" value='{"x":"y"}'/>
          <artificialflow:property name="input:colonVar" value="colonSource"/>
        </artificialflow:properties>
      </bpmn:extensionElements>
    </bpmn:userTask>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="svc"/>
    <bpmn:sequenceFlow id="f2" sourceRef="svc" targetRef="user"/>
    <bpmn:sequenceFlow id="f3" sourceRef="user" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	byID := map[string]model.StepDefinition{}
	for _, s := range wf.Steps {
		byID[s.ID] = s
	}

	svc := byID["svc"]
	if svc.InputParameters["localOrder"] != "orderId" {
		t.Fatalf("expected input.localOrder mapping, got %#v", svc.InputParameters)
	}
	if svc.InputParameters["fromXml"] != "xmlSource" {
		t.Fatalf("expected ioMapping input, got %#v", svc.InputParameters)
	}
	if svc.OutputParameters["result"] != "localResult" {
		t.Fatalf("expected output.result mapping, got %#v", svc.OutputParameters)
	}
	if svc.OutputParameters["xmlOut"] != "localOut" {
		t.Fatalf("expected ioMapping output, got %#v", svc.OutputParameters)
	}
	if len(svc.TaskListeners) != 2 {
		t.Fatalf("expected 2 listeners, got %#v", svc.TaskListeners)
	}
	listenerByEvent := map[string]string{}
	for _, l := range svc.TaskListeners {
		listenerByEvent[l.Event] = l.Implementation
	}
	if listenerByEvent["create"] != "onCreateHandler" || listenerByEvent["complete"] != "onCompleteHandler" {
		t.Fatalf("unexpected listeners: %#v", listenerByEvent)
	}
	if _, ok := svc.Properties["input.localOrder"]; ok {
		t.Fatalf("input.* property should be consumed from Properties")
	}

	user := byID["user"]
	if user.InputParameters["a"] != "b" || user.InputParameters["c"] != "d" {
		t.Fatalf("expected JSON inputParameters, got %#v", user.InputParameters)
	}
	if user.InputParameters["colonVar"] != "colonSource" {
		t.Fatalf("expected input: colon mapping, got %#v", user.InputParameters)
	}
	if user.OutputParameters["x"] != "y" {
		t.Fatalf("expected JSON outputParameters, got %#v", user.OutputParameters)
	}
}

func TestParse_IsInterruptingOnEventSubProcessStart(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:signal id="Sig_1" name="Ping"/>
  <bpmn:process id="P" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:subProcess id="esp" triggeredByEvent="true">
      <bpmn:startEvent id="espStart" isInterrupting="false">
        <bpmn:signalEventDefinition signalRef="Sig_1"/>
      </bpmn:startEvent>
      <bpmn:endEvent id="espEnd"/>
      <bpmn:sequenceFlow id="ef" sourceRef="espStart" targetRef="espEnd"/>
    </bpmn:subProcess>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	var espStart *model.StepDefinition
	for i := range wf.Steps {
		if wf.Steps[i].ID != "esp" {
			continue
		}
		for j := range wf.Steps[i].SubSteps {
			if wf.Steps[i].SubSteps[j].ID == "espStart" {
				espStart = &wf.Steps[i].SubSteps[j]
			}
		}
	}
	if espStart == nil {
		t.Fatal("espStart missing")
	}
	cancel, ok := espStart.Properties["cancel_activity"].(bool)
	if !ok || cancel {
		t.Fatalf("expected cancel_activity=false from isInterrupting=false, got %#v", espStart.Properties)
	}
	if espStart.Properties["event_definition_type"] != "signal" {
		t.Fatalf("expected signal start, got %#v", espStart.Properties)
	}
}

func TestParse_SupportsEscalationAndEventSubProcess(t *testing.T) {
	esc := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:escalation id="Esc_1" name="Esc" escalationCode="ESC_1"/>
  <bpmn:process id="P" isExecutable="true">
    <bpmn:startEvent id="start"><bpmn:escalationEventDefinition escalationRef="Esc_1"/></bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := Parse(strings.NewReader(esc))
	if err != nil {
		t.Fatalf("expected escalation parse ok, got %v", err)
	}
	if wf.Steps[0].Properties["event_definition_type"] != "escalation" {
		t.Fatalf("expected escalation start, got %#v", wf.Steps[0].Properties)
	}
	esp := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D2" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="P2" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:subProcess id="esp" triggeredByEvent="true">
      <bpmn:startEvent id="espStart"><bpmn:messageEventDefinition messageRef="M1"/></bpmn:startEvent>
      <bpmn:endEvent id="espEnd"/>
      <bpmn:sequenceFlow id="ef" sourceRef="espStart" targetRef="espEnd"/>
    </bpmn:subProcess>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
  <bpmn:message id="M1" name="M1"/>
</bpmn:definitions>`
	wf2, err := Parse(strings.NewReader(esp))
	if err != nil {
		t.Fatalf("expected event sub-process parse ok, got %v", err)
	}
	found := false
	for _, s := range wf2.Steps {
		if s.ID == "esp" {
			found = true
			if s.Type != model.StepTypeEventSubProcess {
				t.Fatalf("expected EVENT_SUB_PROCESS, got %s", s.Type)
			}
		}
	}
	if !found {
		t.Fatalf("event sub-process step missing")
	}
}
