package tests

import (
	"context"
	"strconv"
	"testing"

	"github.com/artificialflow/artificialflow/backend/libs/model"
	"github.com/artificialflow/artificialflow/backend/services/workflow-command/internal/application"
)

func TestDeployWorkflowFromBPMN_SignalBoundary(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:signal id="Sig_Abort" name="AbortWork"/>
  <bpmn:process id="SignalBoundaryProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="work" name="Work"/>
    <bpmn:boundaryEvent id="sigBound" attachedToRef="work" cancelActivity="true">
      <bpmn:signalEventDefinition signalRef="Sig_Abort"/>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="endOk"/>
    <bpmn:endEvent id="endAbort"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="work"/>
    <bpmn:sequenceFlow id="f2" sourceRef="work" targetRef="endOk"/>
    <bpmn:sequenceFlow id="f3" sourceRef="sigBound" targetRef="endAbort"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if current := getCurrentSteps(inst); !contains(current, "work") {
		t.Fatalf("expected waiting at work, got %v", current)
	}
	if err := e.PublishSignal(context.Background(), "AbortWork", map[string]any{"reason": "cancel"}); err != nil {
		t.Fatalf("PublishSignal: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected completed via signal boundary, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
	reachedAbort := false
	for _, ex := range inst.Executions {
		if ex.StepID == "endAbort" && ex.Status == "COMPLETED" {
			reachedAbort = true
		}
		if ex.StepID == "work" && (ex.Status == "ACTIVE" || ex.Status == "") {
			t.Fatalf("expected work interrupted, still active")
		}
	}
	if !reachedAbort {
		t.Fatalf("expected endAbort completed, executions=%+v", inst.Executions)
	}
}

func TestDeployWorkflowFromBPMN_EventSubProcessSignal(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:signal id="Sig_Interrupt" name="Abort"/>
  <bpmn:process id="EspSignalProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="work"/>
    <bpmn:endEvent id="end"/>
    <bpmn:subProcess id="esp" triggeredByEvent="true">
      <bpmn:startEvent id="espStart" isInterrupting="true">
        <bpmn:signalEventDefinition signalRef="Sig_Interrupt"/>
      </bpmn:startEvent>
      <bpmn:endEvent id="espEnd"/>
      <bpmn:sequenceFlow id="ef1" sourceRef="espStart" targetRef="espEnd"/>
    </bpmn:subProcess>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="work"/>
    <bpmn:sequenceFlow id="f2" sourceRef="work" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := e.PublishSignal(context.Background(), "Abort", nil); err != nil {
		t.Fatalf("PublishSignal: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	espReached := false
	for _, ex := range inst.Executions {
		if ex.StepID == "espEnd" && ex.Status == "COMPLETED" {
			espReached = true
		}
	}
	if !espReached {
		t.Fatalf("expected event sub-process signal path to complete espEnd, status=%s executions=%+v", inst.Status, inst.Executions)
	}
}

func TestDeployWorkflowFromBPMN_SignalStart(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:signal id="Sig_Start" name="GoSignal"/>
  <bpmn:process id="SignalStartProcess" isExecutable="true">
    <bpmn:startEvent id="start">
      <bpmn:signalEventDefinition signalRef="Sig_Start"/>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	if _, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml)); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	before, err := e.ListCompletedInstances(context.Background(), 50)
	if err != nil {
		t.Fatalf("list before: %v", err)
	}
	if err := e.PublishSignal(context.Background(), "GoSignal", map[string]any{"n": 1}); err != nil {
		t.Fatalf("PublishSignal: %v", err)
	}
	after, err := e.ListCompletedInstances(context.Background(), 50)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(after) <= len(before) {
		t.Fatalf("expected signal start to create an instance, before=%d after=%d", len(before), len(after))
	}
}

func TestDeployWorkflowFromBPMN_MultiInstance(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="D" targetNamespace="t">
  <bpmn:process id="MIProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="review" name="Review" artificialflow:collection="items" artificialflow:elementVariable="item">
      <bpmn:multiInstanceLoopCharacteristics isSequential="false"/>
    </bpmn:userTask>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="review"/>
    <bpmn:sequenceFlow id="f2" sourceRef="review" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	var review *model.StepDefinition
	for i := range wf.Steps {
		if wf.Steps[i].ID == "review" {
			review = &wf.Steps[i]
			break
		}
	}
	if review == nil {
		t.Fatal("review step missing")
	}
	if review.LoopType != "PARALLEL" {
		t.Fatalf("expected PARALLEL loop, got %q", review.LoopType)
	}
	if review.LoopCollection != "items" || review.LoopElement != "item" {
		t.Fatalf("loop fields: collection=%q element=%q", review.LoopCollection, review.LoopElement)
	}

	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{
		"items": []any{"a", "b"},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	reviewIDs := map[string]bool{}
	for _, ex := range inst.Executions {
		if ex.StepID == "review" {
			reviewIDs[ex.ID] = true
		}
	}
	var childExecIDs []string
	for _, ex := range inst.Executions {
		// Multi-instance children parent to the MI scope execution (also StepID review).
		if ex.StepID == "review" && ex.Status == "ACTIVE" && reviewIDs[ex.ParentID] {
			childExecIDs = append(childExecIDs, ex.ID)
		}
	}
	if len(childExecIDs) < 2 {
		t.Fatalf("expected at least 2 active multi-instance children, got %d executions=%+v", len(childExecIDs), inst.Executions)
	}
	for _, id := range childExecIDs {
		if err := e.CompleteExecution(context.Background(), inst.ID, id); err != nil {
			t.Fatalf("complete %s: %v", id, err)
		}
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected completed after MI join, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
}

func TestDeployWorkflowFromBPMN_BoundaryErrorFromXML(t *testing.T) {
	e := setupTestEngine(t)
	e.RegisterHandler("failingService", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		return &application.BpmnError{ErrorCode: "BUSINESS_ERROR", ErrorMessage: "boom"}
	})
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="D" targetNamespace="t">
  <bpmn:error id="Err_Biz" errorCode="BUSINESS_ERROR"/>
  <bpmn:process id="ErrorBoundaryXML" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="serviceTask" name="Fail" artificialflow:taskType="failingService"/>
    <bpmn:boundaryEvent id="boundaryError" attachedToRef="serviceTask" cancelActivity="true">
      <bpmn:errorEventDefinition errorRef="Err_Biz"/>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="normalEnd"/>
    <bpmn:endEvent id="errorEnd"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="serviceTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="serviceTask" targetRef="normalEnd"/>
    <bpmn:sequenceFlow id="f3" sourceRef="boundaryError" targetRef="errorEnd"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	inst, _ = e.GetInstance(context.Background(), inst.ID)
	errorEnd := false
	for _, ex := range inst.Executions {
		if ex.StepID == "errorEnd" && ex.Status == "COMPLETED" {
			errorEnd = true
		}
		if ex.StepID == "normalEnd" && ex.Status == "COMPLETED" {
			t.Fatalf("normal path should not complete")
		}
	}
	if !errorEnd {
		t.Fatalf("expected error boundary path, status=%s executions=%+v", inst.Status, inst.Executions)
	}
}

func TestDeployWorkflowFromBPMN_ExclusiveDefault(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" id="D" targetNamespace="t">
  <bpmn:process id="XorDefault" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:exclusiveGateway id="xor" default="fDefault"/>
    <bpmn:userTask id="taskA"/>
    <bpmn:userTask id="taskDefault"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="xor"/>
    <bpmn:sequenceFlow id="fA" sourceRef="xor" targetRef="taskA">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">amount &gt; 100</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="fDefault" sourceRef="xor" targetRef="taskDefault"/>
    <bpmn:sequenceFlow id="f2" sourceRef="taskA" targetRef="end"/>
    <bpmn:sequenceFlow id="f3" sourceRef="taskDefault" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"amount": 10})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if current := getCurrentSteps(inst); !contains(current, "taskDefault") {
		t.Fatalf("expected default path taskDefault, got %v", current)
	}
}

func TestDeployWorkflowFromBPMN_ParallelJoin(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="ParallelJoinXML" isExecutable="true">
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
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	current := getCurrentSteps(inst)
	if !contains(current, "taskA") || !contains(current, "taskB") {
		t.Fatalf("expected both branches, got %v", current)
	}
	var execA, execB string
	for _, ex := range inst.Executions {
		if ex.StepID == "taskA" && ex.Status == "ACTIVE" {
			execA = ex.ID
		}
		if ex.StepID == "taskB" && ex.Status == "ACTIVE" {
			execB = ex.ID
		}
	}
	if err := e.CompleteExecution(context.Background(), inst.ID, execA); err != nil {
		t.Fatalf("complete A: %v", err)
	}
	if err := e.CompleteExecution(context.Background(), inst.ID, execB); err != nil {
		t.Fatalf("complete B: %v", err)
	}
	inst, _ = e.GetInstance(context.Background(), inst.ID)
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected join to complete, got %s", inst.Status)
	}
}

func TestDeployWorkflowFromBPMN_InclusiveJoin(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" id="D" targetNamespace="t">
  <bpmn:process id="InclusiveJoinXML" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:inclusiveGateway id="split"/>
    <bpmn:userTask id="taskA"/>
    <bpmn:userTask id="taskB"/>
    <bpmn:inclusiveGateway id="join"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="split"/>
    <bpmn:sequenceFlow id="f2" sourceRef="split" targetRef="taskA">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">varA === true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f3" sourceRef="split" targetRef="taskB">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">varB === true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f4" sourceRef="taskA" targetRef="join"/>
    <bpmn:sequenceFlow id="f5" sourceRef="taskB" targetRef="join"/>
    <bpmn:sequenceFlow id="f6" sourceRef="join" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"varA": true, "varB": false})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if current := getCurrentSteps(inst); !contains(current, "taskA") || contains(current, "taskB") {
		t.Fatalf("expected only taskA, got %v", current)
	}
	var execA string
	for _, ex := range inst.Executions {
		if ex.StepID == "taskA" && ex.Status == "ACTIVE" {
			execA = ex.ID
			break
		}
	}
	if err := e.CompleteExecution(context.Background(), inst.ID, execA); err != nil {
		t.Fatalf("complete A: %v", err)
	}
	inst, _ = e.GetInstance(context.Background(), inst.ID)
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected inclusive join to complete, got %s", inst.Status)
	}
}

func TestDeployWorkflowFromBPMN_MessageBoundary(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:artificialflow="http://artificialflow.io/schema/1.0/bpmn" id="D" targetNamespace="t">
  <bpmn:message id="Msg_Abort" name="AbortWork"/>
  <bpmn:process id="MessageBoundaryProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="work" name="Work"/>
    <bpmn:boundaryEvent id="msgBound" attachedToRef="work" cancelActivity="true" artificialflow:correlationKey="orderId">
      <bpmn:messageEventDefinition messageRef="Msg_Abort"/>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="endOk"/>
    <bpmn:endEvent id="endAbort"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="work"/>
    <bpmn:sequenceFlow id="f2" sourceRef="work" targetRef="endOk"/>
    <bpmn:sequenceFlow id="f3" sourceRef="msgBound" targetRef="endAbort"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"orderId": "ORD-1"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if current := getCurrentSteps(inst); !contains(current, "work") {
		t.Fatalf("expected waiting at work, got %v", current)
	}
	if err := e.PublishMessage(context.Background(), "AbortWork", "ORD-1", nil); err != nil {
		t.Fatalf("PublishMessage: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected completed via message boundary, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
	reachedAbort := false
	for _, ex := range inst.Executions {
		if ex.StepID == "endAbort" && ex.Status == "COMPLETED" {
			reachedAbort = true
		}
	}
	if !reachedAbort {
		t.Fatalf("expected endAbort completed, executions=%+v", inst.Executions)
	}
}
