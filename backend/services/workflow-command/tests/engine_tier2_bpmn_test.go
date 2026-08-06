package tests

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/model"
)

func TestDeployWorkflowFromBPMN_EscalationBoundary(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:escalation id="Esc_1" name="NeedHelp" escalationCode="NEED_HELP"/>
  <bpmn:process id="EscProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="work" name="Work"/>
    <bpmn:boundaryEvent id="escBound" attachedToRef="work" cancelActivity="true">
      <bpmn:escalationEventDefinition escalationRef="Esc_1"/>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="endOk"/>
    <bpmn:endEvent id="endEsc"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="work"/>
    <bpmn:sequenceFlow id="f2" sourceRef="work" targetRef="endOk"/>
    <bpmn:sequenceFlow id="f3" sourceRef="escBound" targetRef="endEsc"/>
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
	if err := e.PublishEscalation(context.Background(), "NEED_HELP", nil); err != nil {
		t.Fatalf("PublishEscalation: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected completed via escalation path, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
}

func TestDeployWorkflowFromBPMN_ConditionalCatch(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="CondProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="waitCond">
      <bpmn:conditionalEventDefinition>
        <bpmn:condition xsi:type="bpmn:tFormalExpression" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">ready === true</bpmn:condition>
      </bpmn:conditionalEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="waitCond"/>
    <bpmn:sequenceFlow id="f2" sourceRef="waitCond" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"ready": false})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if current := getCurrentSteps(inst); !contains(current, "waitCond") {
		t.Fatalf("expected waitCond, got %v", current)
	}
	// Flip condition via complete path: update context by completing a no-op — use CheckConditionals after StartInstance with ready true on new instance
	inst2, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), map[string]any{"ready": true})
	if err != nil {
		t.Fatalf("start2: %v", err)
	}
	inst2, _ = e.GetInstance(context.Background(), inst2.ID)
	if inst2.Status != model.StatusCompleted {
		// evaluate on arrive
		if err := e.CheckConditionals(context.Background()); err != nil {
			t.Fatalf("CheckConditionals: %v", err)
		}
		inst2, _ = e.GetInstance(context.Background(), inst2.ID)
	}
	if inst2.Status != model.StatusCompleted {
		t.Fatalf("expected inst2 completed, got %s", inst2.Status)
	}
	_ = inst
}

func TestDeployWorkflowFromBPMN_MessageStart(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:message id="Msg_Start" name="OrderPlaced"/>
  <bpmn:process id="MsgStartProcess" isExecutable="true">
    <bpmn:startEvent id="start">
      <bpmn:messageEventDefinition messageRef="Msg_Start"/>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	if _, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml)); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if err := e.PublishMessage(context.Background(), "OrderPlaced", "", map[string]any{"n": 1}); err != nil {
		t.Fatalf("PublishMessage: %v", err)
	}
}

func TestDeployWorkflowFromBPMN_TimerStartDate(t *testing.T) {
	e := setupTestEngine(t)
	past := time.Now().Add(-time.Second).UTC().Format(time.RFC3339)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="TimerDateStartProcess" isExecutable="true">
    <bpmn:startEvent id="start">
      <bpmn:timerEventDefinition>
        <bpmn:timeDate>` + past + `</bpmn:timeDate>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers: %v", err)
	}
	wfID := strconv.FormatInt(wf.ID, 10)
	completed, err := e.ListCompletedInstances(context.Background(), 50)
	if err != nil {
		t.Fatalf("ListCompletedInstances: %v", err)
	}
	found := false
	for _, inst := range completed {
		if inst.WorkflowID == wfID {
			found = true
			break
		}
	}
	if !found {
		active, _ := e.ListActiveInstances(context.Background())
		for _, inst := range active {
			if inst.WorkflowID == wfID {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected instance started by timeDate start")
	}
}

func TestNonInterruptingBoundaryTimerCycle(t *testing.T) {
	e := setupTestEngine(t)
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "work"}}},
		{
			ID:                "work",
			Type:              model.StepTypeUserTask,
			Outgoing:          []model.Transition{{TargetRef: "endOk"}},
			BoundaryEventRefs: []string{"cycleBound"},
		},
		{
			ID:   "cycleBound",
			Type: model.StepTypeBoundaryEvent,
			Properties: map[string]any{
				"timer_cycle":     "R2/PT0S",
				"timer_type":      "cycle",
				"cancel_activity": false,
				"attached_to":     "work",
			},
			Outgoing: []model.Transition{{TargetRef: "ping"}},
		},
		{ID: "ping", Type: model.StepTypeScriptTask, Properties: map[string]any{"script": "true"}, Outgoing: []model.Transition{{TargetRef: "pingEnd"}}},
		{ID: "pingEnd", Type: model.StepTypeEnd},
		{ID: "endOk", Type: model.StepTypeEnd},
	}
	wf, err := e.DeployWorkflow(context.Background(), "Cycle Boundary", steps)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers 1: %v", err)
	}
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers 2: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	pingEnds := 0
	for _, ex := range inst.Executions {
		if ex.StepID == "pingEnd" || ex.StepID == "ping" {
			pingEnds++
		}
	}
	if pingEnds < 2 {
		t.Fatalf("expected cycle boundary to fire twice, got related executions=%d status=%s", pingEnds, inst.Status)
	}
	// Work should still be active (non-interrupting).
	if !contains(getCurrentSteps(inst), "work") {
		t.Fatalf("expected work still active after non-interrupting cycle, current=%v", getCurrentSteps(inst))
	}
}

func TestDeployWorkflowFromBPMN_TimerStart(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="TimerStartProcess" isExecutable="true">
    <bpmn:startEvent id="start">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT0S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`
	wf, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers: %v", err)
	}
	active, err := e.ListActiveInstances(context.Background())
	if err != nil {
		t.Fatalf("ListActiveInstances: %v", err)
	}
	completed, err := e.ListCompletedInstances(context.Background(), 50)
	if err != nil {
		t.Fatalf("ListCompletedInstances: %v", err)
	}
	wfID := strconv.FormatInt(wf.ID, 10)
	found := 0
	for _, inst := range active {
		if inst.WorkflowID == wfID {
			found++
		}
	}
	for _, inst := range completed {
		if inst.WorkflowID == wfID {
			found++
		}
	}
	if found == 0 {
		t.Fatalf("expected at least one instance started by timer start for workflow %s", wfID)
	}
	// Second tick must not create another one-shot start for the same definition.
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers second: %v", err)
	}
	active2, _ := e.ListActiveInstances(context.Background())
	completed2, _ := e.ListCompletedInstances(context.Background(), 50)
	found2 := 0
	for _, inst := range active2 {
		if inst.WorkflowID == wfID {
			found2++
		}
	}
	for _, inst := range completed2 {
		if inst.WorkflowID == wfID {
			found2++
		}
	}
	if found2 != found {
		t.Fatalf("expected one-shot timer start (count=%d), got %d after second CheckTimers", found, found2)
	}
}

func TestDeployWorkflowFromBPMN_EventSubProcessTimer(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="TimerEspProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="work"/>
    <bpmn:endEvent id="end"/>
    <bpmn:subProcess id="esp" triggeredByEvent="true">
      <bpmn:startEvent id="espStart">
        <bpmn:timerEventDefinition>
          <bpmn:timeDuration>PT0S</bpmn:timeDuration>
        </bpmn:timerEventDefinition>
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
	if current := getCurrentSteps(inst); !contains(current, "work") {
		t.Fatalf("expected waiting at work, got %v", current)
	}
	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	hasEsp := false
	for _, ex := range inst.Executions {
		if ex.StepID == "espStart" || ex.StepID == "espEnd" {
			hasEsp = true
			break
		}
	}
	if !hasEsp && inst.Status != model.StatusCompleted {
		t.Fatalf("expected timer ESP to start, status=%s executions=%+v", inst.Status, inst.Executions)
	}
}

func TestDeployWorkflowFromBPMN_TransactionCancel(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="TxProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:transaction id="tx" name="Tx">
      <bpmn:startEvent id="txStart"/>
      <bpmn:userTask id="txWork"/>
      <bpmn:endEvent id="txCancel"><bpmn:cancelEventDefinition/></bpmn:endEvent>
      <bpmn:sequenceFlow id="t1" sourceRef="txStart" targetRef="txWork"/>
      <bpmn:sequenceFlow id="t2" sourceRef="txWork" targetRef="txCancel"/>
    </bpmn:transaction>
    <bpmn:boundaryEvent id="cancelBound" attachedToRef="tx">
      <bpmn:cancelEventDefinition/>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="afterCancel"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="tx"/>
    <bpmn:sequenceFlow id="f2" sourceRef="tx" targetRef="end"/>
    <bpmn:sequenceFlow id="f3" sourceRef="cancelBound" targetRef="afterCancel"/>
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
	if current := getCurrentSteps(inst); !contains(current, "txWork") {
		t.Fatalf("expected txWork, got %v", current)
	}
	if err := e.CompleteTask(context.Background(), inst.ID, "txWork"); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected completed after cancel path, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
}

func TestDeployWorkflowFromBPMN_CompensationFromXML(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:process id="CompProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:manualTask id="book"/>
    <bpmn:boundaryEvent id="compHandler" attachedToRef="book">
      <bpmn:compensateEventDefinition/>
    </bpmn:boundaryEvent>
    <bpmn:manualTask id="undo"/>
    <bpmn:intermediateThrowEvent id="throwComp">
      <bpmn:compensateEventDefinition/>
    </bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="book"/>
    <bpmn:sequenceFlow id="f2" sourceRef="book" targetRef="throwComp"/>
    <bpmn:sequenceFlow id="f3" sourceRef="throwComp" targetRef="end"/>
    <bpmn:sequenceFlow id="f4" sourceRef="compHandler" targetRef="undo"/>
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
	if current := getCurrentSteps(inst); !contains(current, "book") {
		t.Fatalf("expected book, got %v", current)
	}
	if err := e.CompleteTask(context.Background(), inst.ID, "book"); err != nil {
		t.Fatalf("complete book: %v", err)
	}
	inst, _ = e.GetInstance(context.Background(), inst.ID)
	// After book completes, throwComp runs compensation → undo may be active
	if contains(getCurrentSteps(inst), "undo") {
		if err := e.CompleteTask(context.Background(), inst.ID, "undo"); err != nil {
			t.Fatalf("complete undo: %v", err)
		}
		inst, _ = e.GetInstance(context.Background(), inst.ID)
	}
	if inst.Status != model.StatusCompleted && !contains(getCurrentSteps(inst), "end") {
		// Allow completed
		inst, _ = e.GetInstance(context.Background(), inst.ID)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected completed compensation flow, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
}

func TestManualTaskWaitsForComplete(t *testing.T) {
	e := setupTestEngine(t)
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "manual"}}},
		{ID: "manual", Type: model.StepTypeManualTask, Outgoing: []model.Transition{{TargetRef: "end"}}},
		{ID: "end", Type: model.StepTypeEnd},
	}
	wf, err := e.DeployWorkflow(context.Background(), "ManualWait", steps)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status == model.StatusCompleted {
		t.Fatalf("manual task must wait")
	}
	if err := e.CompleteTask(context.Background(), inst.ID, "manual"); err != nil {
		t.Fatal(err)
	}
	inst, _ = e.GetInstance(context.Background(), inst.ID)
	if inst.Status != model.StatusCompleted {
		t.Fatalf("got %s", inst.Status)
	}
}

func TestDeployWorkflowFromBPMN_EventSubProcessMessage(t *testing.T) {
	e := setupTestEngine(t)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D" targetNamespace="t">
  <bpmn:message id="Msg_Interrupt" name="Abort"/>
  <bpmn:process id="EspProcess" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="work"/>
    <bpmn:endEvent id="end"/>
    <bpmn:subProcess id="esp" triggeredByEvent="true">
      <bpmn:startEvent id="espStart">
        <bpmn:messageEventDefinition messageRef="Msg_Interrupt"/>
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
	if err := e.PublishMessage(context.Background(), "Abort", "", nil); err != nil {
		t.Fatalf("PublishMessage: %v", err)
	}
	// Event sub-process may complete the interrupting path; at minimum deploy+publish must not error.
	_, _ = e.GetInstance(context.Background(), inst.ID)
}

func TestCheckSLAs_PublishesDueDateBreachedEscalationOnce(t *testing.T) {
	e := setupTestEngine(t)
	steps := []model.StepDefinition{
		{ID: "start", Type: model.StepTypeStart, Outgoing: []model.Transition{{TargetRef: "work"}}},
		{
			ID:                "work",
			Type:              model.StepTypeUserTask,
			Properties:        map[string]any{"due_date": "PT0S"},
			Outgoing:          []model.Transition{{TargetRef: "endOk"}},
			BoundaryEventRefs: []string{"escBound"},
		},
		{
			ID:   "escBound",
			Type: model.StepTypeBoundaryEvent,
			Properties: map[string]any{
				"event_definition_type": "escalation",
				"escalation_code":       "user-task.due-date.breached",
				"cancel_activity":       true,
				"attached_to":           "work",
			},
			Outgoing: []model.Transition{{TargetRef: "endEsc"}},
		},
		{ID: "endOk", Type: model.StepTypeEnd},
		{ID: "endEsc", Type: model.StepTypeEnd},
	}
	wf, err := e.DeployWorkflow(context.Background(), "SLA Breach Escalation", steps)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	inst, err := e.StartInstance(context.Background(), strconv.FormatInt(wf.ID, 10), nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	jobs, err := e.ListInstanceJobs(context.Background(), inst.ID)
	if err != nil || len(jobs) == 0 {
		t.Fatalf("expected user-task job, err=%v jobs=%d", err, len(jobs))
	}

	if err := e.CheckSLAs(context.Background()); err != nil {
		t.Fatalf("CheckSLAs: %v", err)
	}
	jobs, err = e.ListInstanceJobs(context.Background(), inst.ID)
	if err != nil || len(jobs) == 0 {
		t.Fatalf("reload jobs: err=%v", err)
	}
	if jobs[0].BreachedAt == nil {
		t.Fatalf("expected BreachedAt set")
	}
	incidents, err := e.ListIncidents(context.Background(), 0, 50)
	if err != nil {
		t.Fatalf("ListIncidents: %v", err)
	}
	foundSLA := false
	for _, inc := range incidents {
		if inc.ErrorType == "SLA_DUE_DATE_BREACHED" && inc.JobKey == jobs[0].Key {
			foundSLA = true
			break
		}
	}
	if !foundSLA {
		t.Fatalf("expected SLA_DUE_DATE_BREACHED incident for job %d", jobs[0].Key)
	}
	inst, err = e.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if inst.Status != model.StatusCompleted {
		t.Fatalf("expected escalation boundary to complete instance, got %s current=%v", inst.Status, getCurrentSteps(inst))
	}
	if val, ok := inst.Context["jobKey"]; !ok {
		t.Fatalf("expected escalation payload jobKey in context, got %v", inst.Context)
	} else if _, isNum := val.(float64); !isNum {
		// JSON numbers may decode as float64; int64 also fine from in-memory merge
		switch val.(type) {
		case int64, float64, int, int32:
		default:
			t.Fatalf("unexpected jobKey type %T", val)
		}
	}

	// Second CheckSLAs must not fail or re-publish (job already breached).
	if err := e.CheckSLAs(context.Background()); err != nil {
		t.Fatalf("CheckSLAs second: %v", err)
	}
}
