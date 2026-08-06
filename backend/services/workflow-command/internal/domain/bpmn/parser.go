package bpmn

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/artificialflow/artificialflow/backend/libs/model"
)

const (
	ArtificialFlowNamespace = "http://artificialflow.io/schema/1.0/bpmn"
)

// Definitions represents the top-level element in a BPMN 2.0 XML file.
// It's simplified to focus on the process definition.
type Definitions struct {
	XMLName     xml.Name            `xml:"definitions"`
	Process     Process             `xml:"process"`
	Messages    []MessageElement    `xml:"message"`
	Signals     []SignalElement     `xml:"signal"`
	Errors      []ErrorElementDef   `xml:"error"`
	Escalations []EscalationElement `xml:"escalation"`
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
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	TriggeredByEvent                 string                            `xml:"triggeredByEvent,attr"`
	fromTransaction                  bool                              // set when unmarshaled from <transaction>
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
	StartEvents                      []StartEvent                      `xml:"startEvent"`
	EndEvents                        []EndEvent                        `xml:"endEvent"`
	ServiceTasks                     []ServiceTask                     `xml:"serviceTask"`
	SendTasks                        []SendTask                        `xml:"sendTask"`
	UserTasks                        []UserTask                        `xml:"userTask"`
	ScriptTasks                      []ScriptTask                      `xml:"scriptTask"`
	ReceiveTasks                     []ReceiveTask                     `xml:"receiveTask"`
	ManualTasks                      []ManualTask                      `xml:"manualTask"`
	BusinessRuleTasks                []BusinessRuleTask                `xml:"businessRuleTask"`
	CallActivities                   []CallActivity                    `xml:"callActivity"`
	IntermediateCatchEvents          []IntermediateCatchEvent          `xml:"intermediateCatchEvent"`
	IntermediateThrowEvents          []IntermediateThrowEvent          `xml:"intermediateThrowEvent"`
	BoundaryEvents                   []BoundaryEvent                   `xml:"boundaryEvent"`
	SequenceFlows                    []SequenceFlow                    `xml:"sequenceFlow"`
	ExclusiveGateways                []ExclusiveGateway                `xml:"exclusiveGateway"`
	ParallelGateways                 []ParallelGateway                 `xml:"parallelGateway"`
	InclusiveGateways                []InclusiveGateway                `xml:"inclusiveGateway"`
	EventBasedGateways               []EventBasedGateway               `xml:"eventBasedGateway"`
	SubProcesses                     []SubProcess                      `xml:"subProcess"`
	Transactions                     []SubProcess                      `xml:"transaction"`
}

// --- BPMN Elements ---

type StartEvent struct {
	ID                     string                  `xml:"id,attr"`
	Name                   string                  `xml:"name,attr"`
	IsInterrupting         string                  `xml:"isInterrupting,attr"`
	ExtensionElements      *ExtensionElements      `xml:"extensionElements"`
	TimerEventDefinition   *TimerEventDefinition   `xml:"timerEventDefinition"`
	MessageEventDefinition *MessageEventDefinition `xml:"messageEventDefinition"`
	SignalEventDefinition  *SignalEventDefinition  `xml:"signalEventDefinition"`
	// Unsupported on start until Tier-2/3: detect and reject below.
	EscalationEventDefinition  *EscalationEventDefinition  `xml:"escalationEventDefinition"`
	ConditionalEventDefinition *ConditionalEventDefinition `xml:"conditionalEventDefinition"`
}

// MultiInstanceLoopCharacteristics models bpmn:multiInstanceLoopCharacteristics.
type MultiInstanceLoopCharacteristics struct {
	IsSequential                  string `xml:"isSequential,attr"`
	ArtificialFlowCollection      string `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection               string `xml:"collection,attr"`
	ArtificialFlowLoopCollection  string `xml:"http://artificialflow.io/schema/1.0/bpmn loopCollection,attr"`
	PlainLoopCollection           string `xml:"loopCollection,attr"`
	ArtificialFlowElementVariable string `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable          string `xml:"elementVariable,attr"`
	ArtificialFlowLoopElement     string `xml:"http://artificialflow.io/schema/1.0/bpmn loopElement,attr"`
	PlainLoopElement              string `xml:"loopElement,attr"`
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
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	JobType                          string                            `xml:"jobType,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowTopic              string                            `xml:"http://artificialflow.io/schema/1.0/bpmn topic,attr"`
	PlainTopic                       string                            `xml:"topic,attr"`
	ArtificialFlowTaskType           string                            `xml:"http://artificialflow.io/schema/1.0/bpmn taskType,attr"`
	PlainTaskType                    string                            `xml:"taskType,attr"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type SendTask struct {
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	JobType                          string                            `xml:"jobType,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowTopic              string                            `xml:"http://artificialflow.io/schema/1.0/bpmn topic,attr"`
	PlainTopic                       string                            `xml:"topic,attr"`
	ArtificialFlowTaskType           string                            `xml:"http://artificialflow.io/schema/1.0/bpmn taskType,attr"`
	PlainTaskType                    string                            `xml:"taskType,attr"`
	MessageRef                       string                            `xml:"messageRef,attr"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type ScriptTask struct {
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowScriptFormat       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn scriptFormat,attr"`
	PlainScriptFormat                string                            `xml:"scriptFormat,attr"`
	ArtificialFlowResultVariable     string                            `xml:"http://artificialflow.io/schema/1.0/bpmn resultVariable,attr"`
	PlainResultVariable              string                            `xml:"resultVariable,attr"`
	ArtificialFlowTimeout            string                            `xml:"http://artificialflow.io/schema/1.0/bpmn timeout,attr"`
	PlainTimeout                     string                            `xml:"timeout,attr"`
	Script                           string                            `xml:"script"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type UserTask struct {
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowAssignee           string                            `xml:"http://artificialflow.io/schema/1.0/bpmn assignee,attr"`
	PlainAssignee                    string                            `xml:"assignee,attr"`
	ArtificialFlowCandidateUsers     string                            `xml:"http://artificialflow.io/schema/1.0/bpmn candidateUsers,attr"`
	PlainCandidateUsers              string                            `xml:"candidateUsers,attr"`
	ArtificialFlowCandidateGroups    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn candidateGroups,attr"`
	PlainCandidateGroups             string                            `xml:"candidateGroups,attr"`
	ArtificialFlowDueDate            string                            `xml:"http://artificialflow.io/schema/1.0/bpmn dueDate,attr"`
	PlainDueDate                     string                            `xml:"dueDate,attr"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type ReceiveTask struct {
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	MessageRef                       string                            `xml:"messageRef,attr"`
	ArtificialFlowCorrelationKey     string                            `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey              string                            `xml:"correlationKey,attr"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type ManualTask struct {
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type BusinessRuleTask struct {
	ID                               string                            `xml:"id,attr"`
	Name                             string                            `xml:"name,attr"`
	ExtensionElements                *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	ArtificialFlowDecisionRef        string                            `xml:"http://artificialflow.io/schema/1.0/bpmn decisionRef,attr"`
	PlainDecisionRef                 string                            `xml:"decisionRef,attr"`
	ArtificialFlowResultVariable     string                            `xml:"http://artificialflow.io/schema/1.0/bpmn resultVariable,attr"`
	PlainResultVariable              string                            `xml:"resultVariable,attr"`
	ArtificialFlowCollection         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                  string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable    string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable             string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential       string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                string                            `xml:"isSequential,attr"`
}

type CallActivity struct {
	ID                                 string                            `xml:"id,attr"`
	Name                               string                            `xml:"name,attr"`
	ExtensionElements                  *ExtensionElements                `xml:"extensionElements"`
	MultiInstanceLoopCharacteristics   *MultiInstanceLoopCharacteristics `xml:"multiInstanceLoopCharacteristics"`
	CalledElement                      string                            `xml:"calledElement,attr"`
	ArtificialFlowCalledElementVersion string                            `xml:"http://artificialflow.io/schema/1.0/bpmn calledElementVersion,attr"`
	PlainCalledElementVersion          string                            `xml:"calledElementVersion,attr"`
	PlainVersion                       string                            `xml:"version,attr"`
	ArtificialFlowCollection           string                            `xml:"http://artificialflow.io/schema/1.0/bpmn collection,attr"`
	PlainCollection                    string                            `xml:"collection,attr"`
	ArtificialFlowElementVariable      string                            `xml:"http://artificialflow.io/schema/1.0/bpmn elementVariable,attr"`
	PlainElementVariable               string                            `xml:"elementVariable,attr"`
	ArtificialFlowIsSequential         string                            `xml:"http://artificialflow.io/schema/1.0/bpmn isSequential,attr"`
	PlainIsSequential                  string                            `xml:"isSequential,attr"`
}

type TimerEventDefinition struct {
	TimeDuration string `xml:"timeDuration"`
	TimeDate     string `xml:"timeDate"`
	TimeCycle    string `xml:"timeCycle"`
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
	ID                           string                      `xml:"id,attr"`
	Name                         string                      `xml:"name,attr"`
	ExtensionElements            *ExtensionElements          `xml:"extensionElements"`
	ArtificialFlowTimerDuration  string                      `xml:"http://artificialflow.io/schema/1.0/bpmn timerDuration,attr"`
	PlainTimerDuration           string                      `xml:"timerDuration,attr"`
	ArtificialFlowCorrelationKey string                      `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey          string                      `xml:"correlationKey,attr"`
	TimerEventDefinition         *TimerEventDefinition       `xml:"timerEventDefinition"`
	MessageEventDefinition       *MessageEventDefinition     `xml:"messageEventDefinition"`
	SignalEventDefinition        *SignalEventDefinition      `xml:"signalEventDefinition"`
	LinkEventDefinition          *LinkEventDefinition        `xml:"linkEventDefinition"`
	EscalationEventDefinition    *EscalationEventDefinition  `xml:"escalationEventDefinition"`
	ConditionalEventDefinition   *ConditionalEventDefinition `xml:"conditionalEventDefinition"`
	CancelEventDefinition        *CancelEventDefinition      `xml:"cancelEventDefinition"`
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
	ID                           string                      `xml:"id,attr"`
	Name                         string                      `xml:"name,attr"`
	ExtensionElements            *ExtensionElements          `xml:"extensionElements"`
	AttachedToRef                string                      `xml:"attachedToRef,attr"`
	CancelActivity               string                      `xml:"cancelActivity,attr"`
	ArtificialFlowTimerDuration  string                      `xml:"http://artificialflow.io/schema/1.0/bpmn timerDuration,attr"`
	PlainTimerDuration           string                      `xml:"timerDuration,attr"`
	ArtificialFlowCorrelationKey string                      `xml:"http://artificialflow.io/schema/1.0/bpmn correlationKey,attr"`
	PlainCorrelationKey          string                      `xml:"correlationKey,attr"`
	ArtificialFlowErrorCode      string                      `xml:"http://artificialflow.io/schema/1.0/bpmn errorCode,attr"`
	PlainErrorCode               string                      `xml:"errorCode,attr"`
	ArtificialFlowErrorMessage   string                      `xml:"http://artificialflow.io/schema/1.0/bpmn errorMessage,attr"`
	PlainErrorMessage            string                      `xml:"errorMessage,attr"`
	TimerEventDefinition         *TimerEventDefinition       `xml:"timerEventDefinition"`
	MessageEventDefinition       *MessageEventDefinition     `xml:"messageEventDefinition"`
	SignalEventDefinition        *SignalEventDefinition      `xml:"signalEventDefinition"`
	ErrorEventDefinition         *ErrorEventDefinition       `xml:"errorEventDefinition"`
	CompensateEventDefinition    *CompensateEventDefinition  `xml:"compensateEventDefinition"`
	EscalationEventDefinition    *EscalationEventDefinition  `xml:"escalationEventDefinition"`
	ConditionalEventDefinition   *ConditionalEventDefinition `xml:"conditionalEventDefinition"`
	CancelEventDefinition        *CancelEventDefinition      `xml:"cancelEventDefinition"`
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
	ArtificialFlowIOMapping  *IOMapping          `xml:"http://artificialflow.io/schema/1.0/bpmn ioMapping"`
	PlainIOMapping           *IOMapping          `xml:"ioMapping"`
}

type WorkflowProperties struct {
	ArtificialFlowProperties []WorkflowProperty `xml:"http://artificialflow.io/schema/1.0/bpmn property"`
	PlainProperties          []WorkflowProperty `xml:"property"`
}

type WorkflowProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// IOMapping models artificialflow:ioMapping with input/output entries.
type IOMapping struct {
	ArtificialFlowInputs  []IOMappingEntry `xml:"http://artificialflow.io/schema/1.0/bpmn input"`
	PlainInputs           []IOMappingEntry `xml:"input"`
	ArtificialFlowOutputs []IOMappingEntry `xml:"http://artificialflow.io/schema/1.0/bpmn output"`
	PlainOutputs          []IOMappingEntry `xml:"output"`
}

type IOMappingEntry struct {
	Target string `xml:"target,attr"`
	Source string `xml:"source,attr"`
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
		if applyTimerEventDefinition(props, se.TimerEventDefinition, "") {
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
		// BPMN isInterrupting on startEvent (event sub-process); default interrupting=true at runtime.
		if raw := strings.TrimSpace(se.IsInterrupting); raw != "" {
			cancelActivity := true
			if parsed, err := strconv.ParseBool(raw); err == nil {
				cancelActivity = parsed
			}
			props["cancel_activity"] = cancelActivity
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

		step := model.StepDefinition{
			ID:             st.ID,
			Name:           st.Name,
			Type:           model.StepTypeServiceTask,
			Implementation: impl,
			Properties:     nilIfEmpty(props),
		}
		applyMultiInstance(&step, st.MultiInstanceLoopCharacteristics,
			firstNonEmpty(st.ArtificialFlowCollection, st.PlainCollection),
			firstNonEmpty(st.ArtificialFlowElementVariable, st.PlainElementVariable),
			firstNonEmpty(st.ArtificialFlowIsSequential, st.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, st.ExtensionElements)
		steps = append(steps, step)
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
			// Default to the shipped HTTP connector; there is no connector.send worker.
			impl = "io.artificialflow.connector.http"
		}

		step := model.StepDefinition{
			ID:             st.ID,
			Name:           st.Name,
			Type:           model.StepTypeSendTask,
			Implementation: impl,
			Properties:     nilIfEmpty(props),
		}
		applyMultiInstance(&step, st.MultiInstanceLoopCharacteristics,
			firstNonEmpty(st.ArtificialFlowCollection, st.PlainCollection),
			firstNonEmpty(st.ArtificialFlowElementVariable, st.PlainElementVariable),
			firstNonEmpty(st.ArtificialFlowIsSequential, st.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, st.ExtensionElements)
		steps = append(steps, step)
	}

	for _, ut := range userTasks {
		props := make(map[string]any)
		setStringProperty(props, "assignee", firstNonEmpty(ut.ArtificialFlowAssignee, ut.PlainAssignee))
		setStringProperty(props, "candidate_users", firstNonEmpty(ut.ArtificialFlowCandidateUsers, ut.PlainCandidateUsers))
		setStringProperty(props, "candidate_groups", firstNonEmpty(ut.ArtificialFlowCandidateGroups, ut.PlainCandidateGroups))
		setStringProperty(props, "due_date", firstNonEmpty(ut.ArtificialFlowDueDate, ut.PlainDueDate))
		props = mergeExtensionProperties(props, ut.ExtensionElements)
		step := model.StepDefinition{
			ID:         ut.ID,
			Name:       ut.Name,
			Type:       model.StepTypeUserTask,
			Properties: nilIfEmpty(props),
		}
		applyMultiInstance(&step, ut.MultiInstanceLoopCharacteristics,
			firstNonEmpty(ut.ArtificialFlowCollection, ut.PlainCollection),
			firstNonEmpty(ut.ArtificialFlowElementVariable, ut.PlainElementVariable),
			firstNonEmpty(ut.ArtificialFlowIsSequential, ut.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, ut.ExtensionElements)
		steps = append(steps, step)
	}

	for _, st := range scriptTasks {
		props := make(map[string]any)
		setStringProperty(props, "script_format", firstNonEmpty(st.ArtificialFlowScriptFormat, st.PlainScriptFormat))
		setStringProperty(props, "result_variable", firstNonEmpty(st.ArtificialFlowResultVariable, st.PlainResultVariable))
		setStringProperty(props, "timeout", firstNonEmpty(st.ArtificialFlowTimeout, st.PlainTimeout))
		setStringProperty(props, "script", strings.TrimSpace(st.Script))
		props = mergeExtensionProperties(props, st.ExtensionElements)
		step := model.StepDefinition{
			ID:         st.ID,
			Name:       st.Name,
			Type:       model.StepTypeScriptTask,
			Properties: nilIfEmpty(props),
		}
		applyMultiInstance(&step, st.MultiInstanceLoopCharacteristics,
			firstNonEmpty(st.ArtificialFlowCollection, st.PlainCollection),
			firstNonEmpty(st.ArtificialFlowElementVariable, st.PlainElementVariable),
			firstNonEmpty(st.ArtificialFlowIsSequential, st.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, st.ExtensionElements)
		steps = append(steps, step)
	}

	for _, rt := range receiveTasks {
		props := make(map[string]any)
		setStringProperty(props, "message_ref", resolveRef(rt.MessageRef, refs.messageByID))
		setStringProperty(props, "correlation_key", firstNonEmpty(rt.ArtificialFlowCorrelationKey, rt.PlainCorrelationKey))
		props = mergeExtensionProperties(props, rt.ExtensionElements)

		step := model.StepDefinition{
			ID:         rt.ID,
			Name:       rt.Name,
			Type:       model.StepTypeReceiveTask,
			Properties: nilIfEmpty(props),
		}
		applyMultiInstance(&step, rt.MultiInstanceLoopCharacteristics,
			firstNonEmpty(rt.ArtificialFlowCollection, rt.PlainCollection),
			firstNonEmpty(rt.ArtificialFlowElementVariable, rt.PlainElementVariable),
			firstNonEmpty(rt.ArtificialFlowIsSequential, rt.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, rt.ExtensionElements)
		steps = append(steps, step)
	}

	for _, mt := range manualTasks {
		props := mergeExtensionProperties(make(map[string]any), mt.ExtensionElements)
		step := model.StepDefinition{ID: mt.ID, Name: mt.Name, Type: model.StepTypeManualTask, Properties: nilIfEmpty(props)}
		applyMultiInstance(&step, mt.MultiInstanceLoopCharacteristics,
			firstNonEmpty(mt.ArtificialFlowCollection, mt.PlainCollection),
			firstNonEmpty(mt.ArtificialFlowElementVariable, mt.PlainElementVariable),
			firstNonEmpty(mt.ArtificialFlowIsSequential, mt.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, mt.ExtensionElements)
		steps = append(steps, step)
	}

	for _, br := range businessRuleTasks {
		props := make(map[string]any)
		setStringProperty(props, "decision_ref", firstNonEmpty(br.ArtificialFlowDecisionRef, br.PlainDecisionRef))
		setStringProperty(props, "result_variable", firstNonEmpty(br.ArtificialFlowResultVariable, br.PlainResultVariable))
		props = mergeExtensionProperties(props, br.ExtensionElements)

		step := model.StepDefinition{
			ID:         br.ID,
			Name:       br.Name,
			Type:       model.StepTypeBusinessRuleTask,
			Properties: nilIfEmpty(props),
		}
		applyMultiInstance(&step, br.MultiInstanceLoopCharacteristics,
			firstNonEmpty(br.ArtificialFlowCollection, br.PlainCollection),
			firstNonEmpty(br.ArtificialFlowElementVariable, br.PlainElementVariable),
			firstNonEmpty(br.ArtificialFlowIsSequential, br.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, br.ExtensionElements)
		steps = append(steps, step)
	}

	for _, ca := range callActivities {
		props := make(map[string]any)
		setStringProperty(props, "called_element", ca.CalledElement)
		setStringProperty(props, "called_element_version", firstNonEmpty(ca.ArtificialFlowCalledElementVersion, ca.PlainCalledElementVersion, ca.PlainVersion))
		props = mergeExtensionProperties(props, ca.ExtensionElements)

		step := model.StepDefinition{
			ID:         ca.ID,
			Name:       ca.Name,
			Type:       model.StepTypeCallActivity,
			Properties: nilIfEmpty(props),
		}
		applyMultiInstance(&step, ca.MultiInstanceLoopCharacteristics,
			firstNonEmpty(ca.ArtificialFlowCollection, ca.PlainCollection),
			firstNonEmpty(ca.ArtificialFlowElementVariable, ca.PlainElementVariable),
			firstNonEmpty(ca.ArtificialFlowIsSequential, ca.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, ca.ExtensionElements)
		steps = append(steps, step)
	}

	for _, catchEvent := range intermediateCatchEvents {
		props := make(map[string]any)
		stepType := model.StepTypeIntermediateCatchEvent

		attrDuration := firstNonEmpty(catchEvent.ArtificialFlowTimerDuration, catchEvent.PlainTimerDuration)
		if applyTimerEventDefinition(props, catchEvent.TimerEventDefinition, attrDuration) {
			stepType = model.StepTypeIntermediateTimerCatchEvent
		}

		if catchEvent.MessageEventDefinition != nil {
			setStringProperty(props, "message_ref", resolveRef(catchEvent.MessageEventDefinition.MessageRef, refs.messageByID))
		}
		if catchEvent.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(catchEvent.SignalEventDefinition.SignalRef, refs.signalByID))
			setStringProperty(props, "event_definition_type", "signal")
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

		attrDuration := firstNonEmpty(boundaryEvent.ArtificialFlowTimerDuration, boundaryEvent.PlainTimerDuration)
		_ = applyTimerEventDefinition(props, boundaryEvent.TimerEventDefinition, attrDuration)

		if boundaryEvent.MessageEventDefinition != nil {
			setStringProperty(props, "message_ref", resolveRef(boundaryEvent.MessageEventDefinition.MessageRef, refs.messageByID))
		}
		if boundaryEvent.SignalEventDefinition != nil {
			setStringProperty(props, "signal_ref", resolveRef(boundaryEvent.SignalEventDefinition.SignalRef, refs.signalByID))
			setStringProperty(props, "event_definition_type", "signal")
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
		step := model.StepDefinition{
			ID:         sp.ID,
			Name:       sp.Name,
			Type:       stepType,
			SubSteps:   subSteps,
			Properties: nilIfEmpty(props),
		}
		applyMultiInstance(&step, sp.MultiInstanceLoopCharacteristics,
			firstNonEmpty(sp.ArtificialFlowCollection, sp.PlainCollection),
			firstNonEmpty(sp.ArtificialFlowElementVariable, sp.PlainElementVariable),
			firstNonEmpty(sp.ArtificialFlowIsSequential, sp.PlainIsSequential),
		)
		applyIOMappingsAndListeners(&step, sp.ExtensionElements)
		steps = append(steps, step)
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

// applyIOMappingsAndListeners extracts InputParameters, OutputParameters, and TaskListeners
// from extension properties (input./output./listener.*) and optional artificialflow:ioMapping.
func applyIOMappingsAndListeners(step *model.StepDefinition, extensions *ExtensionElements) {
	if step == nil {
		return
	}

	inputs := make(map[string]string)
	outputs := make(map[string]string)
	var listeners []model.TaskListener

	mergeIOMap := func(dst map[string]string, src map[string]string) {
		for k, v := range src {
			if k == "" || v == "" {
				continue
			}
			if _, exists := dst[k]; !exists {
				dst[k] = v
			}
		}
	}

	if extensions != nil {
		for _, mapping := range []*IOMapping{extensions.ArtificialFlowIOMapping, extensions.PlainIOMapping} {
			if mapping == nil {
				continue
			}
			for _, in := range append(mapping.ArtificialFlowInputs, mapping.PlainInputs...) {
				target := strings.TrimSpace(in.Target)
				source := strings.TrimSpace(in.Source)
				if target != "" && source != "" {
					if _, exists := inputs[target]; !exists {
						inputs[target] = source
					}
				}
			}
			for _, out := range append(mapping.ArtificialFlowOutputs, mapping.PlainOutputs...) {
				target := strings.TrimSpace(out.Target)
				source := strings.TrimSpace(out.Source)
				if target != "" && source != "" {
					if _, exists := outputs[target]; !exists {
						outputs[target] = source
					}
				}
			}
		}
	}

	if step.Properties != nil {
		keysToDelete := make([]string, 0)
		for key, raw := range step.Properties {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			asString, ok := raw.(string)
			if !ok {
				continue
			}
			value := strings.TrimSpace(asString)
			if value == "" {
				continue
			}

			lower := strings.ToLower(trimmedKey)
			switch {
			case strings.HasPrefix(lower, "input."), strings.HasPrefix(lower, "input:"):
				local := strings.TrimSpace(trimmedKey[len("input."):])
				if strings.HasPrefix(lower, "input:") {
					local = strings.TrimSpace(trimmedKey[len("input:"):])
				}
				if local != "" {
					if _, exists := inputs[local]; !exists {
						inputs[local] = value
					}
					keysToDelete = append(keysToDelete, key)
				}
			case strings.HasPrefix(lower, "output."), strings.HasPrefix(lower, "output:"):
				global := strings.TrimSpace(trimmedKey[len("output."):])
				if strings.HasPrefix(lower, "output:") {
					global = strings.TrimSpace(trimmedKey[len("output:"):])
				}
				if global != "" {
					if _, exists := outputs[global]; !exists {
						outputs[global] = value
					}
					keysToDelete = append(keysToDelete, key)
				}
			case strings.HasPrefix(lower, "listener."):
				event := strings.TrimSpace(trimmedKey[len("listener."):])
				if event != "" {
					listeners = append(listeners, model.TaskListener{
						Event:          strings.ToLower(event),
						Implementation: value,
					})
					keysToDelete = append(keysToDelete, key)
				}
			case lower == "inputparameters" || lower == "input_parameters":
				parsed := parseParameterJSON(value)
				mergeIOMap(inputs, parsed)
				keysToDelete = append(keysToDelete, key)
			case lower == "outputparameters" || lower == "output_parameters":
				parsed := parseParameterJSON(value)
				mergeIOMap(outputs, parsed)
				keysToDelete = append(keysToDelete, key)
			}
		}
		for _, key := range keysToDelete {
			delete(step.Properties, key)
		}
		if len(step.Properties) == 0 {
			step.Properties = nil
		}
	}

	if len(inputs) > 0 {
		step.InputParameters = inputs
	}
	if len(outputs) > 0 {
		step.OutputParameters = outputs
	}
	if len(listeners) > 0 {
		step.TaskListeners = listeners
	}
}

func parseParameterJSON(value string) map[string]string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	var asMap map[string]string
	if err := json.Unmarshal([]byte(trimmed), &asMap); err == nil && len(asMap) > 0 {
		out := make(map[string]string, len(asMap))
		for k, v := range asMap {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k != "" && v != "" {
				out[k] = v
			}
		}
		return out
	}
	var asAny map[string]any
	if err := json.Unmarshal([]byte(trimmed), &asAny); err != nil {
		return nil
	}
	out := make(map[string]string)
	for k, v := range asAny {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		switch typed := v.(type) {
		case string:
			if s := strings.TrimSpace(typed); s != "" {
				out[k] = s
			}
		default:
			if s := strings.TrimSpace(fmt.Sprint(typed)); s != "" && s != "<nil>" {
				out[k] = s
			}
		}
	}
	return out
}

// applyMultiInstance sets LoopType/LoopCollection/LoopElement from
// multiInstanceLoopCharacteristics and/or ArtificialFlow/plain task attributes.
func applyMultiInstance(step *model.StepDefinition, mi *MultiInstanceLoopCharacteristics, collection, elementVariable, isSequential string) {
	if step == nil {
		return
	}
	hasMI := mi != nil
	collection = strings.TrimSpace(collection)
	elementVariable = strings.TrimSpace(elementVariable)
	isSequential = strings.TrimSpace(isSequential)

	if hasMI {
		if collection == "" {
			collection = firstNonEmpty(mi.ArtificialFlowCollection, mi.PlainCollection, mi.ArtificialFlowLoopCollection, mi.PlainLoopCollection)
		}
		if elementVariable == "" {
			elementVariable = firstNonEmpty(mi.ArtificialFlowElementVariable, mi.PlainElementVariable, mi.ArtificialFlowLoopElement, mi.PlainLoopElement)
		}
		if isSequential == "" {
			isSequential = strings.TrimSpace(mi.IsSequential)
		}
	}

	// Also accept extension-property aliases already merged onto the step.
	if step.Properties != nil {
		if collection == "" {
			collection = firstStringProperty(step.Properties, "collection", "loopCollection", "loop_collection")
		}
		if elementVariable == "" {
			elementVariable = firstStringProperty(step.Properties, "elementVariable", "element_variable", "loopElement", "loop_element")
		}
		if isSequential == "" {
			if v, ok := step.Properties["isSequential"].(string); ok {
				isSequential = v
			} else if v, ok := step.Properties["is_sequential"].(string); ok {
				isSequential = v
			} else if b, ok := step.Properties["isSequential"].(bool); ok {
				if b {
					isSequential = "true"
				} else {
					isSequential = "false"
				}
			}
		}
	}

	if !hasMI && collection == "" && elementVariable == "" && isSequential == "" {
		return
	}

	sequential := false
	if isSequential != "" {
		if parsed, err := strconv.ParseBool(isSequential); err == nil {
			sequential = parsed
		}
	}
	if sequential {
		step.LoopType = "SEQUENTIAL"
	} else {
		step.LoopType = "PARALLEL"
	}
	step.LoopCollection = collection
	step.LoopElement = elementVariable
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
	return firstNonEmpty(def.TimeDuration, def.TimeDate, def.TimeCycle)
}

// applyTimerEventDefinition sets timer_duration / timer_date / timer_cycle / timer_type.
// Returns true when any timer property was applied.
func applyTimerEventDefinition(props map[string]any, def *TimerEventDefinition, attrDuration string) bool {
	applied := false
	if def != nil {
		if d := strings.TrimSpace(def.TimeDuration); d != "" {
			setStringProperty(props, "timer_duration", d)
			setStringProperty(props, "timer_type", "duration")
			applied = true
		}
		if d := strings.TrimSpace(def.TimeDate); d != "" {
			setStringProperty(props, "timer_date", d)
			if _, ok := props["timer_type"]; !ok {
				setStringProperty(props, "timer_type", "date")
			}
			applied = true
		}
		if c := strings.TrimSpace(def.TimeCycle); c != "" {
			setStringProperty(props, "timer_cycle", c)
			setStringProperty(props, "timer_type", "cycle")
			applied = true
		}
	}
	if attr := strings.TrimSpace(attrDuration); attr != "" {
		if _, hasDur := props["timer_duration"]; !hasDur {
			if _, hasDate := props["timer_date"]; !hasDate {
				if _, hasCycle := props["timer_cycle"]; !hasCycle {
					// Attr-only: duration unless value looks like absolute date or cycle.
					if strings.HasPrefix(attr, "R") {
						setStringProperty(props, "timer_cycle", attr)
						setStringProperty(props, "timer_type", "cycle")
					} else if looksLikeAbsoluteTime(attr) {
						setStringProperty(props, "timer_date", attr)
						setStringProperty(props, "timer_type", "date")
					} else {
						setStringProperty(props, "timer_duration", attr)
						setStringProperty(props, "timer_type", "duration")
					}
					applied = true
				}
			}
		}
	}
	return applied
}

func looksLikeAbsoluteTime(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "P") || strings.HasPrefix(value, "R") {
		return false
	}
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return true
	}
	if _, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
		return true
	}
	return false
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
	case "calledelementversion", "called_element_version":
		return "called_element_version"
	case "collection", "loopcollection", "loop_collection":
		return "collection"
	case "elementvariable", "element_variable", "loopelement", "loop_element":
		return "element_variable"
	case "issequential", "is_sequential":
		return "is_sequential"
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
