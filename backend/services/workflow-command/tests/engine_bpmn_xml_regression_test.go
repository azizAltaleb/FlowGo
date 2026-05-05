package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
	"workflow-engine/backend/libs/model"
)

func TestDeployWorkflowFromBPMN_ThrowSignalTriggersCatch(t *testing.T) {
	e := setupTestEngine(t)

	receiverXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Receiver" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:signal id="Signal_Global" name="GlobalSignal"/>
  <bpmn:process id="ReceiverProcess" name="Receiver" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="catchSignal">
      <bpmn:signalEventDefinition signalRef="Signal_Global"/>
    </bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="r1" sourceRef="start" targetRef="catchSignal"/>
    <bpmn:sequenceFlow id="r2" sourceRef="catchSignal" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	receiverWF, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(receiverXML))
	if err != nil {
		t.Fatalf("failed to deploy receiver BPMN: %v", err)
	}

	receiverInstance, err := e.StartInstance(context.Background(), strconv.FormatInt(receiverWF.ID, 10), nil)
	if err != nil {
		t.Fatalf("failed to start receiver instance: %v", err)
	}
	if current := getCurrentSteps(receiverInstance); len(current) != 1 || !contains(current, "catchSignal") {
		t.Fatalf("expected receiver waiting at catchSignal, got %v", current)
	}

	senderXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Sender" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:signal id="Signal_Global" name="GlobalSignal"/>
  <bpmn:process id="SenderProcess" name="Sender" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateThrowEvent id="throwSignal">
      <bpmn:signalEventDefinition signalRef="Signal_Global"/>
    </bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="s1" sourceRef="start" targetRef="throwSignal"/>
    <bpmn:sequenceFlow id="s2" sourceRef="throwSignal" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	senderWF, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(senderXML))
	if err != nil {
		t.Fatalf("failed to deploy sender BPMN: %v", err)
	}

	senderInstance, err := e.StartInstance(context.Background(), strconv.FormatInt(senderWF.ID, 10), nil)
	if err != nil {
		t.Fatalf("failed to start sender instance: %v", err)
	}

	senderInstance, err = e.GetInstance(context.Background(), senderInstance.ID)
	if err != nil {
		t.Fatalf("failed to reload sender instance: %v", err)
	}
	if senderInstance.Status != model.StatusCompleted {
		t.Fatalf("expected sender to complete, got %s", senderInstance.Status)
	}

	receiverInstance, err = e.GetInstance(context.Background(), receiverInstance.ID)
	if err != nil {
		t.Fatalf("failed to reload receiver instance: %v", err)
	}
	if receiverInstance.Status != model.StatusCompleted {
		t.Fatalf("expected receiver to complete after signal throw, got %s", receiverInstance.Status)
	}
}

func TestDeployWorkflowFromBPMN_ThrowMessageUsesCorrelationKey(t *testing.T) {
	e := setupTestEngine(t)

	receiverXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:workflowsa="http://workflowsa.com/schema/1.0/bpmn" id="Definitions_ReceiverMsg" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_Order" name="OrderCreated"/>
  <bpmn:process id="ReceiverProcessMsg" name="ReceiverMsg" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:receiveTask id="receiveOrder" messageRef="Message_Order" workflowsa:correlationKey="orderId"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="r1" sourceRef="start" targetRef="receiveOrder"/>
    <bpmn:sequenceFlow id="r2" sourceRef="receiveOrder" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	receiverWF, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(receiverXML))
	if err != nil {
		t.Fatalf("failed to deploy receiver BPMN: %v", err)
	}

	matchedReceiver, err := e.StartInstance(context.Background(), strconv.FormatInt(receiverWF.ID, 10), map[string]any{"orderId": "order-1"})
	if err != nil {
		t.Fatalf("failed to start matched receiver: %v", err)
	}
	otherReceiver, err := e.StartInstance(context.Background(), strconv.FormatInt(receiverWF.ID, 10), map[string]any{"orderId": "order-2"})
	if err != nil {
		t.Fatalf("failed to start other receiver: %v", err)
	}

	senderXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:workflowsa="http://workflowsa.com/schema/1.0/bpmn" id="Definitions_SenderMsg" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_Order" name="OrderCreated"/>
  <bpmn:process id="SenderProcessMsg" name="SenderMsg" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateThrowEvent id="throwOrder" workflowsa:correlationKey="orderId">
      <bpmn:messageEventDefinition messageRef="Message_Order"/>
    </bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="s1" sourceRef="start" targetRef="throwOrder"/>
    <bpmn:sequenceFlow id="s2" sourceRef="throwOrder" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	senderWF, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(senderXML))
	if err != nil {
		t.Fatalf("failed to deploy sender BPMN: %v", err)
	}

	senderInstance, err := e.StartInstance(context.Background(), strconv.FormatInt(senderWF.ID, 10), map[string]any{"orderId": "order-1"})
	if err != nil {
		t.Fatalf("failed to start sender instance: %v", err)
	}

	senderInstance, err = e.GetInstance(context.Background(), senderInstance.ID)
	if err != nil {
		t.Fatalf("failed to reload sender instance: %v", err)
	}
	if senderInstance.Status != model.StatusCompleted {
		t.Fatalf("expected sender to complete, got %s", senderInstance.Status)
	}

	matchedReceiver, err = e.GetInstance(context.Background(), matchedReceiver.ID)
	if err != nil {
		t.Fatalf("failed to reload matched receiver: %v", err)
	}
	if matchedReceiver.Status != model.StatusCompleted {
		t.Fatalf("expected matched receiver to complete, got %s", matchedReceiver.Status)
	}

	otherReceiver, err = e.GetInstance(context.Background(), otherReceiver.ID)
	if err != nil {
		t.Fatalf("failed to reload other receiver: %v", err)
	}
	if otherReceiver.Status != model.StatusRunning {
		t.Fatalf("expected other receiver to keep running, got %s", otherReceiver.Status)
	}
	if current := getCurrentSteps(otherReceiver); len(current) != 1 || !contains(current, "receiveOrder") {
		t.Fatalf("expected other receiver still waiting at receiveOrder, got %v", current)
	}
}

func TestDeployWorkflowFromBPMN_BoundaryTimerNonInterruptingKeepsTaskActive(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_BoundaryNonInterrupting" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_BoundaryNonInterrupting" name="Boundary Non-Interrupting" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="userTask"/>
    <bpmn:boundaryEvent id="timerBoundary" attachedToRef="userTask" cancelActivity="false">
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
		t.Fatalf("expected userTask active before timer, got %v", current)
	}

	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	if instance.Status != model.StatusRunning {
		t.Fatalf("expected instance to keep running due to non-interrupting boundary, got %s", instance.Status)
	}

	if current := getCurrentSteps(instance); !contains(current, "userTask") {
		t.Fatalf("expected userTask to remain active after non-interrupting boundary timer, got %v", current)
	}

	var timeoutReached bool
	for _, ex := range instance.Executions {
		if ex.StepID == "timeoutEnd" && ex.Status == "COMPLETED" {
			timeoutReached = true
			break
		}
	}
	if !timeoutReached {
		t.Fatalf("expected timeout branch to reach timeoutEnd, executions: %+v", instance.Executions)
	}
}

func TestDeployWorkflowFromBPMN_BoundaryCancelActivityExtensionAliasKeepsTaskActive(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:workflowsa="http://workflowsa.com/schema/1.0/bpmn" id="Definitions_BoundaryExtensionAlias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_BoundaryExtensionAlias" name="Boundary Extension Alias" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="userTask"/>
    <bpmn:boundaryEvent id="timerBoundary" attachedToRef="userTask">
      <bpmn:extensionElements>
        <workflowsa:properties>
          <workflowsa:property name="cancelActivity" value="false"/>
        </workflowsa:properties>
      </bpmn:extensionElements>
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
		t.Fatalf("expected userTask active before timer, got %v", current)
	}

	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	if instance.Status != model.StatusRunning {
		t.Fatalf("expected instance to keep running due to non-interrupting boundary extension alias, got %s", instance.Status)
	}

	if current := getCurrentSteps(instance); !contains(current, "userTask") {
		t.Fatalf("expected userTask to remain active after boundary extension alias timer, got %v", current)
	}

	var timeoutReached bool
	for _, ex := range instance.Executions {
		if ex.StepID == "timeoutEnd" && ex.Status == "COMPLETED" {
			timeoutReached = true
			break
		}
	}
	if !timeoutReached {
		t.Fatalf("expected timeout branch to reach timeoutEnd, executions: %+v", instance.Executions)
	}
}

func TestDeployWorkflowFromBPMN_BoundaryCancelActivityAttributeTakesPrecedenceOverExtensionAlias(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:workflowsa="http://workflowsa.com/schema/1.0/bpmn" id="Definitions_BoundaryPrecedence" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_BoundaryPrecedence" name="Boundary Attribute Precedence" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="userTask"/>
    <bpmn:boundaryEvent id="timerBoundary" attachedToRef="userTask" cancelActivity="true">
      <bpmn:extensionElements>
        <workflowsa:properties>
          <workflowsa:property name="cancelActivity" value="false"/>
        </workflowsa:properties>
      </bpmn:extensionElements>
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
		t.Fatalf("expected userTask active before timer, got %v", current)
	}

	if err := e.CheckTimers(context.Background()); err != nil {
		t.Fatalf("CheckTimers failed: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	if contains(getCurrentSteps(instance), "userTask") {
		t.Fatalf("expected userTask to be interrupted by mapped cancelActivity=true, current=%v", getCurrentSteps(instance))
	}

	var timeoutReached bool
	for _, ex := range instance.Executions {
		if ex.StepID == "timeoutEnd" && ex.Status == "COMPLETED" {
			timeoutReached = true
			break
		}
	}
	if !timeoutReached {
		t.Fatalf("expected timeout branch to reach timeoutEnd, executions: %+v", instance.Executions)
	}
}

func TestDeployWorkflowFromBPMN_ThrowMessageUsesPlainCorrelationKeyAlias(t *testing.T) {
	e := setupTestEngine(t)

	receiverXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_ReceiverPlainAlias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_Order" name="OrderCreated"/>
  <bpmn:process id="ReceiverProcessPlainAlias" name="ReceiverPlainAlias" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:receiveTask id="receiveOrder" messageRef="Message_Order" correlationKey="orderId"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="r1" sourceRef="start" targetRef="receiveOrder"/>
    <bpmn:sequenceFlow id="r2" sourceRef="receiveOrder" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	receiverWF, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(receiverXML))
	if err != nil {
		t.Fatalf("failed to deploy receiver BPMN: %v", err)
	}

	matchedReceiver, err := e.StartInstance(context.Background(), strconv.FormatInt(receiverWF.ID, 10), map[string]any{"orderId": "order-plain-1"})
	if err != nil {
		t.Fatalf("failed to start matched receiver: %v", err)
	}
	otherReceiver, err := e.StartInstance(context.Background(), strconv.FormatInt(receiverWF.ID, 10), map[string]any{"orderId": "order-plain-2"})
	if err != nil {
		t.Fatalf("failed to start other receiver: %v", err)
	}

	senderXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_SenderPlainAlias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:message id="Message_Order" name="OrderCreated"/>
  <bpmn:process id="SenderProcessPlainAlias" name="SenderPlainAlias" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateThrowEvent id="throwOrder" correlationKey="orderId">
      <bpmn:messageEventDefinition messageRef="Message_Order"/>
    </bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="s1" sourceRef="start" targetRef="throwOrder"/>
    <bpmn:sequenceFlow id="s2" sourceRef="throwOrder" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	senderWF, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(senderXML))
	if err != nil {
		t.Fatalf("failed to deploy sender BPMN: %v", err)
	}

	senderInstance, err := e.StartInstance(context.Background(), strconv.FormatInt(senderWF.ID, 10), map[string]any{"orderId": "order-plain-1"})
	if err != nil {
		t.Fatalf("failed to start sender instance: %v", err)
	}

	senderInstance, err = e.GetInstance(context.Background(), senderInstance.ID)
	if err != nil {
		t.Fatalf("failed to reload sender instance: %v", err)
	}
	if senderInstance.Status != model.StatusCompleted {
		t.Fatalf("expected sender to complete, got %s", senderInstance.Status)
	}

	matchedReceiver, err = e.GetInstance(context.Background(), matchedReceiver.ID)
	if err != nil {
		t.Fatalf("failed to reload matched receiver: %v", err)
	}
	if matchedReceiver.Status != model.StatusCompleted {
		t.Fatalf("expected matched receiver to complete, got %s", matchedReceiver.Status)
	}

	otherReceiver, err = e.GetInstance(context.Background(), otherReceiver.ID)
	if err != nil {
		t.Fatalf("failed to reload other receiver: %v", err)
	}
	if otherReceiver.Status != model.StatusRunning {
		t.Fatalf("expected other receiver to keep running, got %s", otherReceiver.Status)
	}
	if current := getCurrentSteps(otherReceiver); len(current) != 1 || !contains(current, "receiveOrder") {
		t.Fatalf("expected other receiver still waiting at receiveOrder, got %v", current)
	}
}

func TestDeployWorkflowFromBPMN_ServiceTaskPlainTaskTypeAliasMapsImplementation(t *testing.T) {
	e := setupTestEngine(t)

	e.RegisterHandler("legacy_plain_handler", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		if instance.Context == nil {
			instance.Context = map[string]any{}
		}
		instance.Context["plain_handler_executed"] = true
		return nil
	})

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_ServiceAlias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ServiceAlias" name="Service Alias" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="serviceTask" taskType="legacy_plain_handler"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="serviceTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="serviceTask" targetRef="end"/>
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

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance, got %s", instance.Status)
	}
	if executed, ok := instance.Context["plain_handler_executed"].(bool); !ok || !executed {
		t.Fatalf("expected service handler side effect in context, got %v", instance.Context["plain_handler_executed"])
	}
}

func TestDeployWorkflowFromBPMN_ServiceTaskExtensionPropertyMapsImplementation(t *testing.T) {
	e := setupTestEngine(t)

	e.RegisterHandler("ext_service_handler", func(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition) error {
		if instance.Context == nil {
			instance.Context = map[string]any{}
		}
		instance.Context["ext_handler_executed"] = true
		return nil
	})

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:workflowsa="http://workflowsa.com/schema/1.0/bpmn" id="Definitions_ServiceExtensionAlias" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ServiceExtensionAlias" name="Service Extension Alias" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:serviceTask id="serviceTask">
      <bpmn:extensionElements>
        <workflowsa:properties>
          <workflowsa:property name="taskType" value="ext_service_handler"/>
        </workflowsa:properties>
      </bpmn:extensionElements>
    </bpmn:serviceTask>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="serviceTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="serviceTask" targetRef="end"/>
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

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}

	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance, got %s", instance.Status)
	}
	if executed, ok := instance.Context["ext_handler_executed"].(bool); !ok || !executed {
		t.Fatalf("expected extension-based handler side effect in context, got %v", instance.Context["ext_handler_executed"])
	}
}

func TestDeployWorkflowFromBPMN_UserTaskAssignmentFromExtensionProperties(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:workflowsa="http://workflowsa.com/schema/1.0/bpmn" id="Definitions_UserTaskExtensions" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_UserTaskExtensions" name="User Task Extensions" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="reviewTask">
      <bpmn:extensionElements>
        <workflowsa:properties>
          <workflowsa:property name="assignee" value="alice"/>
          <workflowsa:property name="candidateUsers" value="bob,charlie"/>
          <workflowsa:property name="candidateGroups" value="ops"/>
          <workflowsa:property name="dueDate" value="PT1H"/>
        </workflowsa:properties>
      </bpmn:extensionElements>
    </bpmn:userTask>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="reviewTask"/>
    <bpmn:sequenceFlow id="f2" sourceRef="reviewTask" targetRef="end"/>
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

	jobs, err := e.ActivateJobs(context.Background(), "workflowsa:userTask", "worker-1", 1, 0, 30*time.Second)
	if err != nil {
		t.Fatalf("ActivateJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one activated user task job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.ElementID != "reviewTask" {
		t.Fatalf("expected reviewTask job, got %s", job.ElementID)
	}
	if job.Assignee != "alice" {
		t.Fatalf("expected assignee alice from extension properties, got %q", job.Assignee)
	}
	if job.CandidateUsers != "bob,charlie" {
		t.Fatalf("expected candidate users from extension properties, got %q", job.CandidateUsers)
	}
	if job.CandidateGroups != "ops" {
		t.Fatalf("expected candidate groups from extension properties, got %q", job.CandidateGroups)
	}
	if job.DueDate.IsZero() {
		t.Fatalf("expected due date from extension properties to be parsed")
	}

	if err := e.CompleteTask(context.Background(), instance.ID, "reviewTask"); err != nil {
		t.Fatalf("failed to complete reviewTask: %v", err)
	}

	instance, err = e.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("failed to reload instance: %v", err)
	}
	if instance.Status != model.StatusCompleted {
		t.Fatalf("expected completed instance after reviewTask completion, got %s", instance.Status)
	}
}

func TestDeployWorkflowFromBPMN_FailsForUnsupportedElementReferences(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_Unsupported" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Unsupported" name="Unsupported Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:adHocSubProcess id="adhoc"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="adhoc"/>
    <bpmn:sequenceFlow id="f2" sourceRef="adhoc" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	_, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xmlData))
	if err == nil {
		t.Fatalf("expected DeployWorkflowFromBPMN to fail for unsupported adHocSubProcess reference")
	}
	if !strings.Contains(err.Error(), "sequenceFlow f1") {
		t.Fatalf("expected sequenceFlow id in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "targetRef=adhoc") {
		t.Fatalf("expected missing targetRef=adhoc in error, got: %v", err)
	}
}

func TestDeployWorkflowFromBPMN_FailsForUnsupportedSendTaskReferences(t *testing.T) {
	e := setupTestEngine(t)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_UnsupportedSend" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_UnsupportedSend" name="Unsupported Send Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:sendTask id="send"/>
    <bpmn:endEvent id="end"/>

    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="send"/>
    <bpmn:sequenceFlow id="f2" sourceRef="send" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	_, err := e.DeployWorkflowFromBPMN(context.Background(), []byte(xmlData))
	if err == nil {
		t.Fatalf("expected DeployWorkflowFromBPMN to fail for unsupported sendTask reference")
	}
	if !strings.Contains(err.Error(), "sequenceFlow f1") {
		t.Fatalf("expected sequenceFlow id in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "targetRef=send") {
		t.Fatalf("expected missing targetRef=send in error, got: %v", err)
	}
}
