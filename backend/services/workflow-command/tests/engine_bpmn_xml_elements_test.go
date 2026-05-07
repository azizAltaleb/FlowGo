package tests

import (
	"context"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"strconv"
	"testing"
)

func TestDeployWorkflowFromBPMN_CallActivityBusinessRuleAndManualTask(t *testing.T) {
	e := setupTestEngine(t)

	childXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Child" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ChildProcess" name="Child Process" isExecutable="true">
    <bpmn:startEvent id="childStart"/>
    <bpmn:endEvent id="childEnd"/>
    <bpmn:sequenceFlow id="cf1" sourceRef="childStart" targetRef="childEnd"/>
  </bpmn:process>
</bpmn:definitions>`

	if _, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(childXML)); err != nil {
		t.Fatalf("failed to deploy child BPMN: %v", err)
	}

	parentXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:goflow="http://goflow.com/schema/1.0/bpmn" id="Definitions_Parent" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ParentProcess" name="Parent Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:businessRuleTask id="rule" name="Rule" goflow:decisionRef="decision_1"/>
    <bpmn:callActivity id="callChild" name="Call Child" calledElement="ChildProcess"/>
    <bpmn:manualTask id="manualReview" name="Manual Review"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="rule"/>
    <bpmn:sequenceFlow id="f2" sourceRef="rule" targetRef="callChild"/>
    <bpmn:sequenceFlow id="f3" sourceRef="callChild" targetRef="manualReview"/>
    <bpmn:sequenceFlow id="f4" sourceRef="manualReview" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(parentXML))
	if err != nil {
		t.Fatalf("failed to deploy parent BPMN: %v", err)
	}

	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("failed to start parent instance: %v", err)
	}

	current := getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "manualReview") {
		t.Fatalf("expected manualReview to be active, got %v", current)
	}

	if err := e.CompleteTask(context.Background(), instance.ID, "manualReview"); err != nil {
		t.Fatalf("failed to complete manualReview: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed parent instance, got %s", instance.Status)
	}
}

func TestDeployWorkflowFromBPMN_EventBasedGatewayReceiveAndTimer(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_EventGateway" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_Payment" name="MsgPaymentReceived"/>
  <bpmn:process id="Process_EventGateway" name="Event Gateway Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:eventBasedGateway id="gateway"/>
    <bpmn:receiveTask id="receive" messageRef="Message_Payment"/>
    <bpmn:intermediateCatchEvent id="timerCatch">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1H</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:manualTask id="afterReceive"/>
    <bpmn:manualTask id="afterTimer"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="gateway"/>
    <bpmn:sequenceFlow id="f2" sourceRef="gateway" targetRef="receive"/>
    <bpmn:sequenceFlow id="f3" sourceRef="gateway" targetRef="timerCatch"/>
    <bpmn:sequenceFlow id="f4" sourceRef="receive" targetRef="afterReceive"/>
    <bpmn:sequenceFlow id="f5" sourceRef="timerCatch" targetRef="afterTimer"/>
    <bpmn:sequenceFlow id="f6" sourceRef="afterReceive" targetRef="end"/>
    <bpmn:sequenceFlow id="f7" sourceRef="afterTimer" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xmlData))
	if err != nil {
		t.Fatalf("failed to deploy BPMN: %v", err)
	}

	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("failed to start instance: %v", err)
	}

	current := getCurrentSteps(instance)
	if len(current) != 2 || !contains(current, "receive") || !contains(current, "timerCatch") {
		t.Fatalf("expected receive and timerCatch to be active, got %v", current)
	}

	if err := e.PublishMessage(context.Background(), "MsgPaymentReceived", "", nil); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	current = getCurrentSteps(instance)
	if len(current) != 1 || !contains(current, "afterReceive") {
		t.Fatalf("expected only afterReceive active, got %v", current)
	}

	var timerTerminated bool
	for _, ex := range instance.Executions {
		if ex.StepID == "timerCatch" && ex.Status == "TERMINATED" {
			timerTerminated = true
			break
		}
	}
	if !timerTerminated {
		t.Fatalf("expected timerCatch execution to be terminated, executions: %+v", instance.Executions)
	}
}

func TestDeployWorkflowFromBPMN_BoundaryTimerInterruptsTask(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Boundary" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Boundary" name="Boundary Timer Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="userTask"/>
    <bpmn:boundaryEvent id="timerBoundary" attachedToRef="userTask" cancelActivity="true">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT0S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:scriptTask id="timeoutTask"/>
    <bpmn:endEvent id="normalEnd"/>
    <bpmn:endEvent id="timeoutEnd"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="userTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="userTask" targetRef="normalEnd"/>
    <bpmn:sequenceFlow id="f3" sourceRef="timerBoundary" targetRef="timeoutTask"/>
    <bpmn:sequenceFlow id="f4" sourceRef="timeoutTask" targetRef="timeoutEnd"/>
  </bpmn:process>
</bpmn:definitions>`

	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xmlData))
	if err != nil {
		t.Fatalf("failed to deploy BPMN: %v", err)
	}

	instance, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("failed to start instance: %v", err)
	}

	if current := getCurrentSteps(instance); len(current) != 1 || !contains(current, "userTask") {
		t.Fatalf("expected userTask to be active before timer, got %v", current)
	}

	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	for _, stepID := range getCurrentSteps(instance) {
		if stepID == "userTask" {
			t.Fatalf("expected userTask to be interrupted by boundary timer")
		}
	}

	var timeoutReached bool
	for _, ex := range instance.Executions {
		if ex.StepID == "timeoutEnd" && ex.Status == "COMPLETED" {
			timeoutReached = true
			break
		}
	}
	if !timeoutReached {
		t.Fatalf("expected timeout path to reach timeoutEnd, executions: %+v", instance.Executions)
	}
}
