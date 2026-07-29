package bpmn

import (
	"encoding/xml"
	"fmt"
	"github.com/artificialflow/artificialflow/backend/libs/model"
	"html"
	"io"
	"strconv"
	"strings"
)

const (
	ArtificialFlowNamespace = "http://artificialflow.io/schema/1.0/bpmn"
)

// Definitions represents the top-level element in a BPMN 2.0 XML file.
// It's simplified to focus on the process definition.
type Definitions struct {
	XMLName     xml.Name             `xml:"definitions"`
	Process     Process              `xml:"process"`
	Messages    []MessageElement     `xml:"message"`
	Signals     []SignalElement      `xml:"signal"`
	Errors      []ErrorElementDef    `xml:"error"`
	Escalations []EscalationElement  `xml:"escalation"`
}

type EscalationElement struct {
	ID             string `xml:"id,attr"`
	Name           string `xml:"name,attr"`
	EscalationCode string `xml:"escalationCode,attr"`
}

type MessageElement struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type SignalElement struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type ErrorElementDef struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	ErrorCode string `xml:"errorCode,attr"`
}

// Process represents the <process> element, containing all the flow elements.
type Process struct {
	XMLName                 xml.Name                 `xml:"process"`
	ID                      string                   `xml:"id,attr"`
	Name                    string                   `xml:"name,attr"`
	IsExecutable            bool                     `xml:"isExecutable,attr"`
	StartEvents             []StartEvent             `xml:"startEvent"`
	EndEvents               []EndEvent               `xml:"endEvent"`
	ServiceTasks            []ServiceTask            `xml:"serviceTask"`
	SendTasks               []SendTask               `xml:"sendTask"`
	UserTasks               []UserTask               `xml:"userTask"`
	ScriptTasks             []ScriptTask             `xml:"scriptTask"`
	ReceiveTasks            []ReceiveTask            `xml:"receiveTask"`
	ManualTasks             []ManualTask             `xml:"manualTask"`
	BusinessRuleTasks       []BusinessRuleTask       `xml:"businessRuleTask"`
	CallActivities          []CallActivity           `xml:"callActivity"`
	IntermediateCatchEvents []IntermediateCatchEvent `xml:"intermediateCatchEvent"`
	IntermediateThrowEvents []IntermediateThrowEvent `xml:"intermediateThrowEvent"`
	BoundaryEvents          []BoundaryEvent          `xml:"boundaryEvent"`
	SequenceFlows           []SequenceFlow           `xml:"sequenceFlow"`
	ExclusiveGateways       []ExclusiveGateway       `xml:"exclusiveGateway"`
	ParallelGateways        []ParallelGateway        `xml:"parallelGateway"`
	InclusiveGateways       []InclusiveGateway       `xml:"inclusiveGateway"`
	EventBasedGateways      []EventBasedGateway      `xml:"eventBasedGateway"`
	SubProcesses            []SubProcess             `xml:"subProcess"`
	Transactions            []SubProcess             `xml:"transaction"`
}

type SubProcess struct {
	ID                      string                   `xml:"id,attr"`
	Name                    string                   `xml:"name,attr"`
	TriggeredByEvent        string                   `xml:"triggeredByEvent,attr"`
	fromTransaction         bool                     // set when unmarshaled from <transaction>
	ExtensionElements       *ExtensionElements       `xml:"extensionElements"`
	StartEvents             []StartEvent             `xml:"startEvent"`
	EndEvents               []EndEvent               `xml:"endEvent"`
	ServiceTasks            []ServiceTask            `xml:"serviceTask"`
	SendTasks               []SendTask               `xml:"sendTask"`
	UserTasks               []UserTask               `xml:"userTask"`
	ScriptTasks             []ScriptTask             `xml:"scriptTask"`
	ReceiveTasks            []ReceiveTask            `xml:"receiveTask"`
	ManualTasks             []ManualTask             `xml:"manualTask"`
	BusinessRuleTasks       []BusinessRuleTask       `xml:"businessRuleTask"`
	CallActivities          []CallActivity           `xml:"callActivity"`
	IntermediateCatchEvents []IntermediateCatchEvent `xml:"intermediateCatchEvent"`
	IntermediateThrowEvents []IntermediateThrowEvent `xml:"intermediateThrowEvent"`
	BoundaryEvents          []BoundaryEvent          `xml:"boundaryEvent"`
	SequenceFlows           []SequenceFlow           `xml:"sequenceFlow"`
	ExclusiveGateways       []ExclusiveGateway       `xml:"exclusiveGateway"`
	ParallelGateways        []ParallelGateway        `xml:"parallelGateway"`
	InclusiveGateways       []InclusiveGateway       `xml:"inclusiveGateway"`
	EventBasedGateways      []EventBasedGateway      `xml:"eventBasedGateway"`
	SubProcesses            []SubProcess             `xml:"subProcess"`
	Transactions            []SubProcess             `xml:"transaction"`
}

// --- BPMN Elements ---

type StartEvent struct {
	ID                     string                   `xml:"id,attr"`
	Name                   string                   `xml:"name,attr"`
	ExtensionElements      *ExtensionElements       `xml:"extensionElements"`
	TimerEventDefinition   *TimerEventDefinition    `xml:"timerEventDefinition"`
	MessageEventDefinition *MessageEventDefinition  `xml:"messageEventDefinition"`
	SignalEventDefinition  *SignalEventDefinition   `xml:"signalEventDefinition"`
	// Unsupported on start until Tier-2/3: detect and reject below.
	EscalationEventDefinition *EscalationEventDefinition `xml:"escalationEventDefinition"`
	ConditionalEventDefinition *ConditionalEventDefinition `xml:"conditionalEventDefinition"`
}

type EndEvent struct {
	ID                        string                     `xml:"id,attr"`
	Name                      string                     `xml:"name,attr"`
	ExtensionElements         *ExtensionElements         `xml:"extensionElements"`
	SignalEventDefinition     *SignalEventDefinition     `xml:"signalEventDefinition"`
	ErrorEventDefinition      *ErrorEventDefinition      `xml:"errorEventDefinition"`
	CompensateEventDefinition *CompensateEventDefinition `xml:"compensateEventDefinition"`
	TerminateEventDefinition  *TerminateEventDefinition  `xml:"terminateEventDefinition"`
	EscalationEventDefinition *EscalationEventDefinition `xml:"escalationEventDefinition"`
	CancelEventDefinition     *CancelEventDefinition     `xml:"cancelEventDefinition"`
}

type ServiceTask struct {
	ID                     string             `xml:"id,attr"`
	Name                   string             `xml:"name,attr"`
	JobType                string             `xml:"jobType,attr"`
	ExtensionElements      *ExtensionElements `xml:"extensionElements"`
	ArtificialFlowTopic    string             `xml:"http://artificialflow.io/schema/1.0/bpmn topic,attr"`
	PlainTopic             string             `xml:"topic,attr"`
	ArtificialFlowTaskType string             `xml:"http://artificialflow.io/schema/1.0/bpmn taskType,attr"`
	PlainTaskType          string             `xml:"taskType,attr"`
}

type SendTask struct {
	ID                     string             `xml:"id,attr"`
	Name                   string             `xml:"name,attr"`
	JobType                string             `xml:"jobType,attr"`
	ExtensionElements      *ExtensionElements `xml:"extensionElements"`
	ArtificialFlowTopic    string             `xml:"http://artificialflow.io/schema/1.0/bpmn topic,attr"`
	PlainTopic             string             `xml:"topic,attr"`
	ArtificialFlowTaskType string             `xml:"http://artificialflow.io/schema/1.0/bpmn taskType,attr"`
	PlainTaskType          string             `xml:"taskType,attr"`
	MessageRef             string             `xml:"messageRef,attr"`
}

type ScriptTask struct {
	ID                           string             `xml:"id,attr"`
	Name                         string             `xml:"name,attr"`
	ExtensionElements            *ExtensionElements `xml:"extensionElements"`
	ArtificialFlowScriptFormat   string             `xml:"http://artificialflow.io/schema/1.0/bpmn scriptFormat,attr"`
	PlainScriptFormat            string             `xml:"scriptFormat,attr"`
	ArtificialFlowResultVariable string             `xml:"http://artificialflow.io/schema/1.0/bpmn resultVariable,attr"`
	PlainResultVariable          string             `xml:"resultVariable,attr"`
	ArtificialFlowTimeout        string             `xml:"http://artificialflow.io/schema/1.0/bpmn timeout,attr"`
	PlainTimeout                 string             `xml:"timeout,attr"`
	Script                       string             `xml:"script"`
}

type UserTask struct {
	ID                            string             `xml:"id,attr"`
	Name                          string             `xml:"name,attr"`
	ExtensionElements             *ExtensionElements `xml:"extensionElements"`
	ArtificialFlowAssignee        string             `xml:"http://artificialflow.io/schema/1.0/bpmn assignee,attr"`
	PlainAssignee                 string             `xml:"assignee,attr"`
	ArtificialFlowCandidateUsers  string             `xml:"http://artificialflow.io/schema/1.0/bpmn candidateUsers,attr"`
	PlainCandidateUsers           string             `xml:"candidateUsers,attr"`
	ArtificialFlowCandidateGroups string             `xml:"http://artificialflow.io/schema/1.0/bpmn candidateGroups,attr"`
	PlainCandidateGroups          string             `xml:"candidateGroups,attr"`
	ArtificialFlowDueDate         string             `xml:"http://artificialflow.io/schema/1.0/bpmn dueDate,attr"`
	PlainDueDate                  string             `xml:"dueDate,attr"`
}

type ReceiveTask struct {
	ID                           string             `xml:"id,attr"`
	Name                         string             `xml:"name,attr"`
	ExtensionElements            *ExtensionElements `xml:"extensionElements"`
	MessageRef                   string             `xml:"messageRef,attr"`
	ArtificialFlowCorrelationKey string             `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey          string             `xml:"correlationKey,attr"`
}

type ManualTask struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
}

type BusinessRuleTask struct {
	ID                           string             `xml:"id,attr"`
	Name                         string             `xml:"name,attr"`
	ExtensionElements            *ExtensionElements `xml:"extensionElements"`
	ArtificialFlowDecisionRef    string             `xml:"http://artificialflow.io/schema/1.0/bpmn decisionRef,attr"`
	PlainDecisionRef             string             `xml:"decisionRef,attr"`
	ArtificialFlowResultVariable string             `xml:"http://artificialflow.io/schema/1.0/bpmn resultVariable,attr"`
	PlainResultVariable          string             `xml:"resultVariable,attr"`
}

type CallActivity struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
	CalledElement     string             `xml:"calledElement,attr"`
}

type TimerEventDefinition struct {
	TimeDuration string `xml:"timeDuration"`
	TimeDate     string `xml:"timeDate"`
}

type MessageEventDefinition struct {
	MessageRef string `xml:"messageRef,attr"`
}

type SignalEventDefinition struct {
	SignalRef string `xml:"signalRef,attr"`
}

type ErrorEventDefinition struct {
	ErrorRef string `xml:"errorRef,attr"`
}

type CompensateEventDefinition struct {
	ActivityRef string `xml:"activityRef,attr"`
}

type TerminateEventDefinition struct{}

type LinkEventDefinition struct {
	Name string `xml:"name,attr"`
}

type EscalationEventDefinition struct {
	EscalationRef string `xml:"escalationRef,attr"`
}

type ConditionalEventDefinition struct {
	Condition *ConditionExpression `xml:"condition"`
}

type CancelEventDefinition struct{}

type IntermediateCatchEvent struct {
	ID                           string                  `xml:"id,attr"`
	Name                         string                  `xml:"name,attr"`
	ExtensionElements            *ExtensionElements      `xml:"extensionElements"`
	ArtificialFlowTimerDuration  string                  `xml:"http://artificialflow.io/schema/1.0/bpmn timerDuration,attr"`
	PlainTimerDuration           string                  `xml:"timerDuration,attr"`
	ArtificialFlowCorrelationKey string                  `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey          string                  `xml:"correlationKey,attr"`
	TimerEventDefinition         *TimerEventDefinition   `xml:"timerEventDefinition"`
	MessageEventDefinition       *MessageEventDefinition `xml:"messageEventDefinition"`
	SignalEventDefinition        *SignalEventDefinition  `xml:"signalEventDefinition"`
	LinkEventDefinition          *LinkEventDefinition    `xml:"linkEventDefinition"`
	EscalationEventDefinition    *EscalationEventDefinition `xml:"escalationEventDefinition"`
	ConditionalEventDefinition   *ConditionalEventDefinition `xml:"conditionalEventDefinition"`
	CancelEventDefinition        *CancelEventDefinition  `xml:"cancelEventDefinition"`
}

type IntermediateThrowEvent struct {
	ID                           string                     `xml:"id,attr"`
	Name                         string                     `xml:"name,attr"`
	ExtensionElements            *ExtensionElements         `xml:"extensionElements"`
	ArtificialFlowCorrelationKey string                     `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey          string                     `xml:"correlationKey,attr"`
	ArtificialFlowErrorCode      string                     `xml:"http://artificialflow.io/schema/1.0/bpmn errorCode,attr"`
	PlainErrorCode               string                     `xml:"errorCode,attr"`
	ArtificialFlowErrorMessage   string                     `xml:"http://artificialflow.io/schema/1.0/bpmn errorMessage,attr"`
	PlainErrorMessage            string                     `xml:"errorMessage,attr"`
	MessageEventDefinition       *MessageEventDefinition    `xml:"messageEventDefinition"`
	SignalEventDefinition        *SignalEventDefinition     `xml:"signalEventDefinition"`
	ErrorEventDefinition         *ErrorEventDefinition      `xml:"errorEventDefinition"`
	CompensateEventDefinition    *CompensateEventDefinition `xml:"compensateEventDefinition"`
	LinkEventDefinition          *LinkEventDefinition       `xml:"linkEventDefinition"`
	EscalationEventDefinition    *EscalationEventDefinition `xml:"escalationEventDefinition"`
}

type BoundaryEvent struct {
	ID                           string                     `xml:"id,attr"`
	Name                         string                     `xml:"name,attr"`
	ExtensionElements            *ExtensionElements         `xml:"extensionElements"`
	AttachedToRef                string                     `xml:"attachedToRef,attr"`
	CancelActivity               string                     `xml:"cancelActivity,attr"`
	ArtificialFlowTimerDuration  string                     `xml:"http://artificialflow.io/schema/1.0/bpmn timerDuration,attr"`
	PlainTimerDuration           string                     `xml:"timerDuration,attr"`
	ArtificialFlowCorrelationKey string                     `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey          string                     `xml:"correlationKey,attr"`
	ArtificialFlowErrorCode      string                     `xml:"http://artificialflow.io/schema/1.0/bpmn errorCode,attr"`
	PlainErrorCode               string                     `xml:"errorCode,attr"`
	ArtificialFlowErrorMessage   string                     `xml:"http://artificialflow.io/schema/1.0/bpmn errorMessage,attr"`
	PlainErrorMessage            string                     `xml:"errorMessage,attr"`
	TimerEventDefinition          *TimerEventDefinition          `xml:"timerEventDefinition"`
	MessageEventDefinition        *MessageEventDefinition        `xml:"messageEventDefinition"`
	SignalEventDefinition         *SignalEventDefinition         `xml:"signalEventDefinition"`
	ErrorEventDefinition          *ErrorEventDefinition          `xml:"errorEventDefinition"`
	CompensateEventDefinition     *CompensateEventDefinition     `xml:"compensateEventDefinition"`
	EscalationEventDefinition     *EscalationEventDefinition     `xml:"escalationEventDefinition"`
	ConditionalEventDefinition    *ConditionalEventDefinition    `xml:"conditionalEventDefinition"`
	CancelEventDefinition         *CancelEventDefinition         `xml:"cancelEventDefinition"`
}

type ExclusiveGateway struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
	DefaultFlowID     string             `xml:"default,attr"`
}

type ParallelGateway struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
}

type InclusiveGateway struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
	DefaultFlowID     string             `xml:"default,attr"`
}

type EventBasedGateway struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
}

// SequenceFlow represents a connection between two elements.
type SequenceFlow struct {
	ID                      string               `xml:"id,attr"`
	SourceRef               string               `xml:"sourceRef,attr"`
	TargetRef               string               `xml:"targetRef,attr"`
	ConditionExpression     *ConditionExpression `xml:"conditionExpression"`
	ArtificialFlowCondition string               `xml:"http://artificialflow.io/schema/1.0/bpmn condition,attr"`
	PlainCondition          string               `xml:"condition,attr"`
}

// ConditionExpression holds the logic for a conditional flow.
type ConditionExpression struct {
	Content string `xml:",innerxml"`
}

type ExtensionElements struct {
	ArtificialFlowProperties *WorkflowProperties `xml:"http://artificialflow.io/schema/1.0/bpmn properties"`
	PlainWorkflowProperties  *WorkflowProperties `xml:"properties"`
}

type WorkflowProperties struct {
	ArtificialFlowProperties []WorkflowProperty `xml:"http://artificialflow.io/schema/1.0/bpmn property"`
	PlainProperties          []WorkflowProperty `xml:"property"`
}

type WorkflowProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type elementRefs struct {
	messageByID        map[string]string
	signalByID         map[string]string
	errorCodeByID      map[string]string
	escalationCodeByID map[string]string
}

// Parse reads BPMN 2.0 XML from an io.Reader and transforms it into a simplified
// model.WorkflowDefinition.
func Parse(r io.Reader) (*model.WorkflowDefinition, error) {
	var defs Definitions
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&defs); err != nil {
		return nil, err
	}

	refs := buildElementRefs(defs)

	process := defs.Process
	// Avoid len(a)+len(b) capacity: CodeQL flags possible int overflow on allocation size.
	subProcesses := append([]SubProcess(nil), process.SubProcesses...)
	for _, tx := range process.Transactions {
		tx.fromTransaction = true
		subProcesses = append(subProcesses, tx)
	}
	steps, err := parseFlowElements(
		process.StartEvents,
		process.EndEvents,
		process.ServiceTasks,
		process.SendTasks,
		process.UserTasks,
		process.ScriptTasks,
		process.ReceiveTasks,
		process.ManualTasks,
		process.BusinessRuleTasks,
		process.CallActivities,
		process.IntermediateCatchEvents,
		process.IntermediateThrowEvents,
		process.BoundaryEvents,
		process.ExclusiveGateways,
		process.ParallelGateways,
		process.InclusiveGateways,
		process.EventBasedGateways,
		subProcesses,
		process.SequenceFlows,
		refs,
	)
	if err != nil {
		return nil, err
	}

	wf := &model.WorkflowDefinition{
		ProcessDefinitionID: process.ID,
		Name:                process.Name,
		Steps:               steps,
	}

	return wf, nil
}

func buildElementRefs(defs Definitions) elementRefs {
	refs := elementRefs{
		messageByID:        make(map[string]string),
		signalByID:         make(map[string]string),
		errorCodeByID:      make(map[string]string),
		escalationCodeByID: make(map[string]string),
	}

	for _, msg := range defs.Messages {
		id := strings.TrimSpace(msg.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(msg.Name)
		if name == "" {
			name = id
		}
		refs.messageByID[id] = name
	}

	for _, sig := range defs.Signals {
		id := strings.TrimSpace(sig.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(sig.Name)
		if name == "" {
			name = id
		}
		refs.signalByID[id] = name
	}

	for _, errDef := range defs.Errors {
		id := strings.TrimSpace(errDef.ID)
		if id == "" {
			continue
		}
		code := strings.TrimSpace(errDef.ErrorCode)
		if code == "" {
			code = strings.TrimSpace(errDef.Name)
		}
		if code == "" {
			code = id
		}
		refs.errorCodeByID[id] = code
	}

	for _, esc := range defs.Escalations {
		id := strings.TrimSpace(esc.ID)
		if id == "" {
			continue
		}
		code := strings.TrimSpace(esc.EscalationCode)
		if code == "" {
			code = strings.TrimSpace(esc.Name)
		}
		if code == "" {
			code = id
		}
		refs.escalationCodeByID[id] = code
	}

	return refs
}

func extractConditionExpression(def *ConditionalEventDefinition) string {
	if def == nil || def.Condition == nil {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(def.Condition.Content))
}

func parseFlowElements(
	startEvents []StartEvent,
	endEvents []EndEvent,
	serviceTasks []ServiceTask,
	sendTasks []SendTask,
	userTasks []UserTask,
	scriptTasks []ScriptTask,
	receiveTasks []ReceiveTask,
	manualTasks []ManualTask,
	businessRuleTasks []BusinessRuleTask,
	callActivities []CallActivity,
	intermediateCatchEvents []IntermediateCatchEvent,
	intermediateThrowEvents []IntermediateThrowEvent,
	boundaryEvents []BoundaryEvent,
	exclusiveGateways []ExclusiveGateway,
	parallelGateways []ParallelGateway,
	inclusiveGateways []InclusiveGateway,
	eventBasedGateways []EventBasedGateway,
	subProcesses []SubProcess,
	sequenceFlows []SequenceFlow,
	refs elementRefs,
) ([]model.StepDefinition, error) {
	steps := make([]model.StepDefinition, 0)
	boundaryToAttached := make(map[string]string)

	for _, se := range startEvents {
		props := make(map[string]any)
		if timerDuration := extractTimerDuration(se.TimerEventDefinition); timerDuration != "" {
			setStringProperty(props, "timer_duration", timerDuration)
			setStringProperty(props, "event_definition_type", "timer")
		}
		if se.MessageEventDefinition != nil {
			setStringProperty(props, "message_ref", resolveRef(se.MessageEventDefinition.MessageRef, refs.messageByID))
			setStringProperty(props, "event_definition_type", "message")
		}
		if se.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(se.SignalEventDefinition.SignalRef, refs.signalByID))
			setStringProperty(props, "event_definition_type", "signal")
		}
		if se.EscalationEventDefinition != nil {
			code := resolveRef(se.EscalationEventDefinition.EscalationRef, refs.escalationCodeByID)
			setStringProperty(props, "escalation_ref", code)
			setStringProperty(props, "escalation_code", code)
			setStringProperty(props, "event_definition_type", "escalation")
		}
		if cond := extractConditionExpression(se.ConditionalEventDefinition); cond != "" {
			setStringProperty(props, "condition", cond)
			setStringProperty(props, "event_definition_type", "conditional")
		}
		props = mergeExtensionProperties(props, se.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: se.ID, Name: se.Name, Type: model.StepTypeStart, Properties: nilIfEmpty(props)})
	}

	for _, ee := range endEvents {
		props := make(map[string]any)
		if ee.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(ee.SignalEventDefinition.SignalRef, refs.signalByID))
		}
		if ee.ErrorEventDefinition != nil {
			errorCode := resolveRef(ee.ErrorEventDefinition.ErrorRef, refs.errorCodeByID)
			setStringProperty(props, "error_ref", errorCode)
			setStringProperty(props, "error_code", errorCode)
		}
		if ee.CompensateEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "compensate")
			setStringProperty(props, "activity_ref", ee.CompensateEventDefinition.ActivityRef)
		}
		if ee.TerminateEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "terminate")
		}
		if ee.EscalationEventDefinition != nil {
			code := resolveRef(ee.EscalationEventDefinition.EscalationRef, refs.escalationCodeByID)
			setStringProperty(props, "escalation_ref", code)
			setStringProperty(props, "escalation_code", code)
			setStringProperty(props, "event_definition_type", "escalation")
		}
		if ee.CancelEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "cancel")
		}
		props = mergeExtensionProperties(props, ee.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: ee.ID, Name: ee.Name, Type: model.StepTypeEnd, Properties: nilIfEmpty(props)})
	}

	for _, st := range serviceTasks {
		props := make(map[string]any)
		setStringProperty(props, "topic", firstNonEmpty(st.ArtificialFlowTopic, st.PlainTopic))
		setStringProperty(props, "task_type", firstNonEmpty(st.ArtificialFlowTaskType, st.PlainTaskType))
		props = mergeExtensionProperties(props, st.ExtensionElements)

		impl := strings.TrimSpace(st.JobType)
		if impl == "" {
			impl = firstStringProperty(props,
				"task_type",
				"taskType",
				"job_type",
				"jobType",
				"topic",
				"implementation",
				"handler",
			)
		}

		steps = append(steps, model.StepDefinition{
			ID:             st.ID,
			Name:           st.Name,
			Type:           model.StepTypeServiceTask,
			Implementation: impl,
			Properties:     nilIfEmpty(props),
		})
	}

	for _, st := range sendTasks {
		props := make(map[string]any)
		setStringProperty(props, "topic", firstNonEmpty(st.ArtificialFlowTopic, st.PlainTopic))
		setStringProperty(props, "task_type", firstNonEmpty(st.ArtificialFlowTaskType, st.PlainTaskType))
		setStringProperty(props, "message_ref", strings.TrimSpace(st.MessageRef))
		props = mergeExtensionProperties(props, st.ExtensionElements)

		impl := strings.TrimSpace(st.JobType)
		if impl == "" {
			impl = firstStringProperty(props,
				"task_type",
				"taskType",
				"job_type",
				"jobType",
				"topic",
				"implementation",
				"handler",
			)
		}
		if impl == "" {
			impl = "io.artificialflow.connector.send"
		}

		steps = append(steps, model.StepDefinition{
			ID:             st.ID,
			Name:           st.Name,
			Type:           model.StepTypeSendTask,
			Implementation: impl,
			Properties:     nilIfEmpty(props),
		})
	}

	for _, ut := range userTasks {
		props := make(map[string]any)
		setStringProperty(props, "assignee", firstNonEmpty(ut.ArtificialFlowAssignee, ut.PlainAssignee))
		setStringProperty(props, "candidate_users", firstNonEmpty(ut.ArtificialFlowCandidateUsers, ut.PlainCandidateUsers))
		setStringProperty(props, "candidate_groups", firstNonEmpty(ut.ArtificialFlowCandidateGroups, ut.PlainCandidateGroups))
		setStringProperty(props, "due_date", firstNonEmpty(ut.ArtificialFlowDueDate, ut.PlainDueDate))
		props = mergeExtensionProperties(props, ut.ExtensionElements)
		steps = append(steps, model.StepDefinition{
			ID:         ut.ID,
			Name:       ut.Name,
			Type:       model.StepTypeUserTask,
			Properties: nilIfEmpty(props),
		})
	}

	for _, st := range scriptTasks {
		props := make(map[string]any)
		setStringProperty(props, "script_format", firstNonEmpty(st.ArtificialFlowScriptFormat, st.PlainScriptFormat))
		setStringProperty(props, "result_variable", firstNonEmpty(st.ArtificialFlowResultVariable, st.PlainResultVariable))
		setStringProperty(props, "timeout", firstNonEmpty(st.ArtificialFlowTimeout, st.PlainTimeout))
		setStringProperty(props, "script", strings.TrimSpace(st.Script))
		props = mergeExtensionProperties(props, st.ExtensionElements)
		steps = append(steps, model.StepDefinition{
			ID:         st.ID,
			Name:       st.Name,
			Type:       model.StepTypeScriptTask,
			Properties: nilIfEmpty(props),
		})
	}

	for _, rt := range receiveTasks {
		props := make(map[string]any)
		setStringProperty(props, "message_ref", resolveRef(rt.MessageRef, refs.messageByID))
		setStringProperty(props, "correlation_key", firstNonEmpty(rt.ArtificialFlowCorrelationKey, rt.PlainCorrelationKey))
		props = mergeExtensionProperties(props, rt.ExtensionElements)

		steps = append(steps, model.StepDefinition{
			ID:         rt.ID,
			Name:       rt.Name,
			Type:       model.StepTypeReceiveTask,
			Properties: nilIfEmpty(props),
		})
	}

	for _, mt := range manualTasks {
		props := mergeExtensionProperties(make(map[string]any), mt.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: mt.ID, Name: mt.Name, Type: model.StepTypeManualTask, Properties: nilIfEmpty(props)})
	}

	for _, br := range businessRuleTasks {
		props := make(map[string]any)
		setStringProperty(props, "decision_ref", firstNonEmpty(br.ArtificialFlowDecisionRef, br.PlainDecisionRef))
		setStringProperty(props, "result_variable", firstNonEmpty(br.ArtificialFlowResultVariable, br.PlainResultVariable))
		props = mergeExtensionProperties(props, br.ExtensionElements)

		steps = append(steps, model.StepDefinition{
			ID:         br.ID,
			Name:       br.Name,
			Type:       model.StepTypeBusinessRuleTask,
			Properties: nilIfEmpty(props),
		})
	}

	for _, ca := range callActivities {
		props := make(map[string]any)
		setStringProperty(props, "called_element", ca.CalledElement)
		props = mergeExtensionProperties(props, ca.ExtensionElements)

		steps = append(steps, model.StepDefinition{
			ID:         ca.ID,
			Name:       ca.Name,
			Type:       model.StepTypeCallActivity,
			Properties: nilIfEmpty(props),
		})
	}

	for _, catchEvent := range intermediateCatchEvents {
		props := make(map[string]any)
		stepType := model.StepTypeIntermediateCatchEvent

		timerDuration := firstNonEmpty(catchEvent.ArtificialFlowTimerDuration, catchEvent.PlainTimerDuration, extractTimerDuration(catchEvent.TimerEventDefinition))
		if timerDuration != "" {
			stepType = model.StepTypeIntermediateTimerCatchEvent
			setStringProperty(props, "timer_duration", timerDuration)
		}

		if catchEvent.MessageEventDefinition != nil {
			setStringProperty(props, "message_ref", resolveRef(catchEvent.MessageEventDefinition.MessageRef, refs.messageByID))
		}
		if catchEvent.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(catchEvent.SignalEventDefinition.SignalRef, refs.signalByID))
		}
		if catchEvent.LinkEventDefinition != nil {
			linkName := firstNonEmpty(catchEvent.LinkEventDefinition.Name, catchEvent.Name)
			setStringProperty(props, "link_name", linkName)
			setStringProperty(props, "event_definition_type", "link")
		}
		if catchEvent.EscalationEventDefinition != nil {
			code := resolveRef(catchEvent.EscalationEventDefinition.EscalationRef, refs.escalationCodeByID)
			setStringProperty(props, "escalation_ref", code)
			setStringProperty(props, "escalation_code", code)
			setStringProperty(props, "event_definition_type", "escalation")
		}
		if cond := extractConditionExpression(catchEvent.ConditionalEventDefinition); cond != "" {
			setStringProperty(props, "condition", cond)
			setStringProperty(props, "event_definition_type", "conditional")
		}
		if catchEvent.CancelEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "cancel")
		}
		setStringProperty(props, "correlation_key", firstNonEmpty(catchEvent.ArtificialFlowCorrelationKey, catchEvent.PlainCorrelationKey))
		props = mergeExtensionProperties(props, catchEvent.ExtensionElements)

		steps = append(steps, model.StepDefinition{
			ID:         catchEvent.ID,
			Name:       catchEvent.Name,
			Type:       stepType,
			Properties: nilIfEmpty(props),
		})
	}

	for _, throwEvent := range intermediateThrowEvents {
		props := make(map[string]any)

		if throwEvent.MessageEventDefinition != nil {
			setStringProperty(props, "message_ref", resolveRef(throwEvent.MessageEventDefinition.MessageRef, refs.messageByID))
		}
		if throwEvent.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(throwEvent.SignalEventDefinition.SignalRef, refs.signalByID))
		}
		if throwEvent.ErrorEventDefinition != nil {
			errorCode := resolveRef(throwEvent.ErrorEventDefinition.ErrorRef, refs.errorCodeByID)
			setStringProperty(props, "error_ref", errorCode)
			setStringProperty(props, "error_code", errorCode)
		}
		if throwEvent.CompensateEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "compensate")
			setStringProperty(props, "activity_ref", throwEvent.CompensateEventDefinition.ActivityRef)
		}
		if throwEvent.LinkEventDefinition != nil {
			linkName := firstNonEmpty(throwEvent.LinkEventDefinition.Name, throwEvent.Name)
			setStringProperty(props, "link_name", linkName)
			setStringProperty(props, "event_definition_type", "link")
		}
		if throwEvent.EscalationEventDefinition != nil {
			code := resolveRef(throwEvent.EscalationEventDefinition.EscalationRef, refs.escalationCodeByID)
			setStringProperty(props, "escalation_ref", code)
			setStringProperty(props, "escalation_code", code)
			setStringProperty(props, "event_definition_type", "escalation")
		}

		setStringProperty(props, "correlation_key", firstNonEmpty(throwEvent.ArtificialFlowCorrelationKey, throwEvent.PlainCorrelationKey))
		setStringProperty(props, "error_code", firstNonEmpty(throwEvent.ArtificialFlowErrorCode, throwEvent.PlainErrorCode))
		setStringProperty(props, "error_message", firstNonEmpty(throwEvent.ArtificialFlowErrorMessage, throwEvent.PlainErrorMessage))
		props = mergeExtensionProperties(props, throwEvent.ExtensionElements)

		steps = append(steps, model.StepDefinition{
			ID:         throwEvent.ID,
			Name:       throwEvent.Name,
			Type:       model.StepTypeIntermediateThrowEvent,
			Properties: nilIfEmpty(props),
		})
	}

	for _, boundaryEvent := range boundaryEvents {
		props := make(map[string]any)
		setStringProperty(props, "attached_to", boundaryEvent.AttachedToRef)

		if rawCancelActivity := strings.TrimSpace(boundaryEvent.CancelActivity); rawCancelActivity != "" {
			cancelActivity := true
			if parsed, err := strconv.ParseBool(rawCancelActivity); err == nil {
				cancelActivity = parsed
			}
			props["cancel_activity"] = cancelActivity
		}

		timerDuration := firstNonEmpty(boundaryEvent.ArtificialFlowTimerDuration, boundaryEvent.PlainTimerDuration, extractTimerDuration(boundaryEvent.TimerEventDefinition))
		setStringProperty(props, "timer_duration", timerDuration)

		if boundaryEvent.MessageEventDefinition != nil {
			setStringProperty(props, "message_ref", resolveRef(boundaryEvent.MessageEventDefinition.MessageRef, refs.messageByID))
		}
		if boundaryEvent.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(boundaryEvent.SignalEventDefinition.SignalRef, refs.signalByID))
		}
		if boundaryEvent.ErrorEventDefinition != nil {
			errorCode := resolveRef(boundaryEvent.ErrorEventDefinition.ErrorRef, refs.errorCodeByID)
			setStringProperty(props, "error_ref", errorCode)
			setStringProperty(props, "error_code", errorCode)
		}
		if boundaryEvent.CompensateEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "compensate")
			setStringProperty(props, "activity_ref", boundaryEvent.CompensateEventDefinition.ActivityRef)
		}
		if boundaryEvent.EscalationEventDefinition != nil {
			code := resolveRef(boundaryEvent.EscalationEventDefinition.EscalationRef, refs.escalationCodeByID)
			setStringProperty(props, "escalation_ref", code)
			setStringProperty(props, "escalation_code", code)
			setStringProperty(props, "event_definition_type", "escalation")
		}
		if cond := extractConditionExpression(boundaryEvent.ConditionalEventDefinition); cond != "" {
			setStringProperty(props, "condition", cond)
			setStringProperty(props, "event_definition_type", "conditional")
		}
		if boundaryEvent.CancelEventDefinition != nil {
			setStringProperty(props, "event_definition_type", "cancel")
		}

		setStringProperty(props, "correlation_key", firstNonEmpty(boundaryEvent.ArtificialFlowCorrelationKey, boundaryEvent.PlainCorrelationKey))
		setStringProperty(props, "error_code", firstNonEmpty(boundaryEvent.ArtificialFlowErrorCode, boundaryEvent.PlainErrorCode))
		setStringProperty(props, "error_message", firstNonEmpty(boundaryEvent.ArtificialFlowErrorMessage, boundaryEvent.PlainErrorMessage))
		props = mergeExtensionProperties(props, boundaryEvent.ExtensionElements)

		steps = append(steps, model.StepDefinition{
			ID:         boundaryEvent.ID,
			Name:       boundaryEvent.Name,
			Type:       model.StepTypeBoundaryEvent,
			Properties: nilIfEmpty(props),
		})

		if attached := strings.TrimSpace(boundaryEvent.AttachedToRef); attached != "" {
			boundaryToAttached[boundaryEvent.ID] = attached
		}
	}

	for _, gw := range exclusiveGateways {
		props := mergeExtensionProperties(make(map[string]any), gw.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: gw.ID, Name: gw.Name, Type: model.StepTypeGatewayExclusive, DefaultFlow: gw.DefaultFlowID, Properties: nilIfEmpty(props)})
	}
	for _, gw := range parallelGateways {
		props := mergeExtensionProperties(make(map[string]any), gw.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: gw.ID, Name: gw.Name, Type: model.StepTypeGatewayParallel, Properties: nilIfEmpty(props)})
	}
	for _, gw := range inclusiveGateways {
		props := mergeExtensionProperties(make(map[string]any), gw.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: gw.ID, Name: gw.Name, Type: model.StepTypeGatewayInclusive, DefaultFlow: gw.DefaultFlowID, Properties: nilIfEmpty(props)})
	}
	for _, gw := range eventBasedGateways {
		props := mergeExtensionProperties(make(map[string]any), gw.ExtensionElements)
		steps = append(steps, model.StepDefinition{ID: gw.ID, Name: gw.Name, Type: model.StepTypeGatewayEventBased, Properties: nilIfEmpty(props)})
	}
	for _, sp := range subProcesses {
		// Avoid len(a)+len(b) capacity: CodeQL flags possible int overflow on allocation size.
		nested := append([]SubProcess(nil), sp.SubProcesses...)
		for _, tx := range sp.Transactions {
			tx.fromTransaction = true
			nested = append(nested, tx)
		}
		subSteps, err := parseFlowElements(
			sp.StartEvents,
			sp.EndEvents,
			sp.ServiceTasks,
			sp.SendTasks,
			sp.UserTasks,
			sp.ScriptTasks,
			sp.ReceiveTasks,
			sp.ManualTasks,
			sp.BusinessRuleTasks,
			sp.CallActivities,
			sp.IntermediateCatchEvents,
			sp.IntermediateThrowEvents,
			sp.BoundaryEvents,
			sp.ExclusiveGateways,
			sp.ParallelGateways,
			sp.InclusiveGateways,
			sp.EventBasedGateways,
			nested,
			sp.SequenceFlows,
			refs,
		)
		if err != nil {
			return nil, fmt.Errorf("subProcess %s: %w", sp.ID, err)
		}
		triggeredByEvent := false
		if raw := strings.TrimSpace(sp.TriggeredByEvent); raw != "" {
			if parsed, err := strconv.ParseBool(raw); err == nil {
				triggeredByEvent = parsed
			} else if strings.EqualFold(raw, "true") {
				triggeredByEvent = true
			}
		}
		props := mergeExtensionProperties(make(map[string]any), sp.ExtensionElements)
		stepType := model.StepTypeSubProcess
		if triggeredByEvent {
			stepType = model.StepTypeEventSubProcess
			props["triggered_by_event"] = true
		}
		if sp.fromTransaction {
			props["transaction"] = true
		}
		steps = append(steps, model.StepDefinition{
			ID:         sp.ID,
			Name:       sp.Name,
			Type:       stepType,
			SubSteps:   subSteps,
			Properties: nilIfEmpty(props),
		})
	}

	stepByID := make(map[string]*model.StepDefinition, len(steps))
	for i := range steps {
		stepByID[steps[i].ID] = &steps[i]
	}

	for boundaryID, attachedStepID := range boundaryToAttached {
		if attachedStep, ok := stepByID[attachedStepID]; ok {
			attachedStep.BoundaryEventRefs = appendIfMissing(attachedStep.BoundaryEventRefs, boundaryID)
		}
	}

	for _, flow := range sequenceFlows {
		sourceID := strings.TrimSpace(flow.SourceRef)
		targetID := strings.TrimSpace(flow.TargetRef)
		if sourceID == "" || targetID == "" {
			continue
		}

		sourceStep, sourceOK := stepByID[sourceID]
		targetStep, targetOK := stepByID[targetID]
		if !sourceOK || !targetOK {
			missing := make([]string, 0, 2)
			if !sourceOK {
				missing = append(missing, fmt.Sprintf("sourceRef=%s", sourceID))
			}
			if !targetOK {
				missing = append(missing, fmt.Sprintf("targetRef=%s", targetID))
			}
			flowID := strings.TrimSpace(flow.ID)
			if flowID == "" {
				flowID = "<unknown>"
			}
			return nil, fmt.Errorf("sequenceFlow %s references unsupported or undefined BPMN element(s): %s", flowID, strings.Join(missing, ", "))
		}

		transition := model.Transition{ID: flow.ID, TargetRef: targetID}
		if condition := firstNonEmpty(flow.ArtificialFlowCondition, flow.PlainCondition); condition != "" {
			transition.Condition = normalizeConditionExpression(condition)
		} else if flow.ConditionExpression != nil {
			transition.Condition = normalizeConditionExpression(flow.ConditionExpression.Content)
		}
		sourceStep.Outgoing = append(sourceStep.Outgoing, transition)
		targetStep.Incoming = appendIfMissing(targetStep.Incoming, sourceID)
	}

	return steps, nil
}

func resolveRef(ref string, lookup map[string]string) string {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return ""
	}
	if resolved, ok := lookup[trimmed]; ok {
		return resolved
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractTimerDuration(def *TimerEventDefinition) string {
	if def == nil {
		return ""
	}
	return firstNonEmpty(def.TimeDuration, def.TimeDate)
}

func setStringProperty(props map[string]any, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		props[key] = trimmed
	}
}

func mergeExtensionProperties(props map[string]any, extensions *ExtensionElements) map[string]any {
	if extensions == nil {
		return props
	}
	if props == nil {
		props = make(map[string]any)
	}

	groups := []*WorkflowProperties{
		extensions.ArtificialFlowProperties,
		extensions.PlainWorkflowProperties,
	}
	for _, group := range groups {
		if group == nil {
			continue
		}

		allProps := make([]WorkflowProperty, 0, len(group.ArtificialFlowProperties)+len(group.PlainProperties))
		allProps = append(allProps, group.ArtificialFlowProperties...)
		allProps = append(allProps, group.PlainProperties...)

		for _, property := range allProps {
			key := strings.TrimSpace(property.Name)
			if key == "" {
				continue
			}
			value := strings.TrimSpace(property.Value)
			normalizedValue := normalizeExtensionPropertyValue(key, value)

			if _, exists := props[key]; !exists {
				props[key] = normalizedValue
			}

			canonicalKey := canonicalPropertyKey(key)
			if canonicalKey == "" || canonicalKey == key {
				continue
			}
			if _, exists := props[canonicalKey]; exists {
				continue
			}
			props[canonicalKey] = normalizeExtensionPropertyValue(canonicalKey, value)
		}
	}

	return props
}

func normalizeExtensionPropertyValue(key string, value string) any {
	if canonicalPropertyKey(key) == "cancel_activity" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return value
}

func canonicalPropertyKey(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "assignee":
		return "assignee"
	case "candidateusers", "candidate_users":
		return "candidate_users"
	case "candidategroups", "candidate_groups":
		return "candidate_groups"
	case "duedate", "due_date":
		return "due_date"
	case "calledelement", "called_element":
		return "called_element"
	case "decisionref", "decision_ref":
		return "decision_ref"
	case "resultvariable", "result_variable":
		return "result_variable"
	case "scriptformat", "script_format":
		return "script_format"
	case "messageref", "message_ref":
		return "message_ref"
	case "signalref", "signal_ref":
		return "signal_ref"
	case "correlationkey", "correlation_key":
		return "correlation_key"
	case "timerduration", "timer_duration":
		return "timer_duration"
	case "errorcode", "error_code":
		return "error_code"
	case "errormessage", "error_message":
		return "error_message"
	case "cancelactivity", "cancel_activity":
		return "cancel_activity"
	case "attachedto", "attached_to":
		return "attached_to"
	case "tasktype", "task_type":
		return "task_type"
	case "jobtype", "job_type":
		return "job_type"
	case "topic":
		return "topic"
	case "implementation":
		return "implementation"
	case "handler":
		return "handler"
	case "activityref", "activity_ref":
		return "activity_ref"
	case "eventdefinitiontype", "event_definition_type":
		return "event_definition_type"
	default:
		return ""
	}
}

func firstStringProperty(props map[string]any, keys ...string) string {
	if props == nil {
		return ""
	}

	for _, key := range keys {
		value, ok := props[key]
		if !ok {
			continue
		}

		asString, ok := value.(string)
		if !ok {
			continue
		}

		if trimmed := strings.TrimSpace(asString); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func nilIfEmpty(props map[string]any) map[string]any {
	if len(props) == 0 {
		return nil
	}
	return props
}

func appendIfMissing(values []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return values
	}
	for _, existing := range values {
		if existing == trimmed {
			return values
		}
	}
	return append(values, trimmed)
}

func normalizeConditionExpression(condition string) string {
	normalized := strings.TrimSpace(html.UnescapeString(condition))

	if strings.HasPrefix(normalized, "<![CDATA[") && strings.HasSuffix(normalized, "]]>") {
		normalized = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(normalized, "<![CDATA["), "]]>"))
	}

	if (strings.HasPrefix(normalized, "${") || strings.HasPrefix(normalized, "#{")) && strings.HasSuffix(normalized, "}") {
		normalized = strings.TrimSpace(normalized[2 : len(normalized)-1])
	}

	return normalized
}
