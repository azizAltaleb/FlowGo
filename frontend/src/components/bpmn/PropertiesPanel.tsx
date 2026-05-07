import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { type Node, type Edge } from '@xyflow/react';
import ExtensionProps from "./ExtensionProps";
import { DebouncedInput } from "@/components/ui/debounced-input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface PropertiesPanelProps {
  element: Node | Edge | null;
  onUpdate: (id: string, newData: Record<string, unknown>) => void;
}

export default function PropertiesPanel({ element, onUpdate }: PropertiesPanelProps) {
  
  const updateField = (key: string, value: string) => {
    if (!element) return;

    const newData = { ...element.data };
    
    // Handle special mappings
    if (key === 'label') {
        newData.label = value;
        // Also update name attribute for BPMN
        newData['@_name'] = value;
    } else if (key.startsWith('@_')) {
        newData[key] = value;
    } else if (key === 'bpmn:conditionExpression') {
        // Condition is usually a child element
        if (value) {
            newData['bpmn:conditionExpression'] = {
                '#text': value,
                '@_xsi:type': 'bpmn:tFormalExpression'
            };
        } else {
            delete newData['bpmn:conditionExpression'];
        }
    } else {
        newData[key] = value;
    }

    onUpdate(element.id, newData);
  };

  if (!element) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center mt-10">
        <p>Select an element to edit properties.</p>
      </div>
    );
  }

  // Derived values from props
  const data = element.data || {};
  const elementId = element.id;
  const elementName = (data.label as string) || '';
  
  // User Task properties
  const assignee = (data['@_goflow:assignee'] as string) || '';
  const candidateUsers = (data['@_goflow:candidateUsers'] as string) || '';
  const candidateGroups = (data['@_goflow:candidateGroups'] as string) || '';
  
  // Business Rule Task properties
  const decisionRef = (data['@_goflow:decisionRef'] as string) || '';
  const brResultVariable = (data['@_goflow:resultVariable'] as string) || '';

  // Service Task properties
  const topic = (data['@_goflow:topic'] as string) || '';
  const taskType = (data['@_goflow:taskType'] as string) || '';

  // Script Task properties
  const scriptFormat = (data['@_scriptFormat'] as string) || '';
  const scriptResultVariable = (data['@_goflow:resultVariable'] as string) || '';

  // Sequence Flow properties
  let condition = '';
  const cond = data['bpmn:conditionExpression'];
  if (typeof cond === 'object' && cond !== null) {
      condition = (cond as Record<string, unknown>)['#text'] as string || '';
  } else {
      condition = (cond as string) || '';
  }

  // Call Activity properties
  const calledElement = (data['@_calledElement'] as string) || '';

  // Determine type
  // For nodes, we stored 'originalType' in data. For edges, typically no 'originalType' or it's sequenceFlow
  const originalType = (element.data?.originalType as string) || (element.type === 'floating' || element.type === 'smoothstep' ? 'bpmn:sequenceFlow' : '');
  
  const isUserTask = originalType === 'bpmn:userTask';
  const isBusinessRuleTask = originalType === 'bpmn:businessRuleTask';
  const isServiceTask = originalType === 'bpmn:serviceTask';
  const isScriptTask = originalType === 'bpmn:scriptTask';
  const isCallActivity = originalType === 'bpmn:callActivity';
  const isSequenceFlow = originalType === 'bpmn:sequenceFlow' || element.type === 'floating' || element.type === 'smoothstep'; // React Flow edges
  
  // Gateways
  const isGateway = originalType.includes('Gateway');
  const defaultFlow = (data['@_default'] as string) || '';

  return (
    <div className="h-full bg-gray-50 border-l overflow-y-auto">
      <Card className="rounded-none border-0 shadow-none bg-transparent h-full flex flex-col">
        <CardHeader className="py-3 px-4 border-b bg-white">
          <CardTitle className="text-sm font-semibold">Properties</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 p-0">
            <Tabs defaultValue="general" className="w-full h-full flex flex-col">
                <div className="px-4 py-2 border-b bg-white">
                    <TabsList className="w-full grid grid-cols-2">
                        <TabsTrigger value="general">General</TabsTrigger>
                        <TabsTrigger value="extensions">Extensions</TabsTrigger>
                    </TabsList>
                </div>

                <TabsContent value="general" className="flex-1 overflow-y-auto p-4 space-y-4 m-0">
                    {/* General Properties */}
                    <div className="space-y-3">
                        <div className="space-y-2">
                            <Label>Name</Label>
                            <DebouncedInput
                            value={elementName}
                            onValueChange={(val) => updateField('label', val)}
                            placeholder="e.g. Approve Request"
                            className="bg-white"
                            />
                        </div>
                        
                        <div className="space-y-1">
                            <Label className="text-xs text-muted-foreground">ID</Label>
                            <div className="text-xs font-mono text-muted-foreground bg-muted p-1.5 rounded select-all truncate border">
                                {elementId}
                            </div>
                        </div>
                    </div>

                    {/* User Task Properties */}
                    {isUserTask && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">User Task</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Assignee</Label>
                                    <DebouncedInput
                                    value={assignee}
                                    onValueChange={(val) => updateField('@_goflow:assignee', val)}
                                    placeholder="e.g. user@example.com"
                                    className="bg-white"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>Candidate Users</Label>
                                    <DebouncedInput
                                    value={candidateUsers}
                                    onValueChange={(val) => updateField('@_goflow:candidateUsers', val)}
                                    placeholder="e.g. user1,user2"
                                    className="bg-white"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>Candidate Groups</Label>
                                    <DebouncedInput
                                    value={candidateGroups}
                                    onValueChange={(val) => updateField('@_goflow:candidateGroups', val)}
                                    placeholder="e.g. managers,hr"
                                    className="bg-white"
                                    />
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Business Rule Task Properties */}
                    {isBusinessRuleTask && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Business Rule</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Decision Ref</Label>
                                    <DebouncedInput
                                    value={decisionRef}
                                    onValueChange={(val) => updateField('@_goflow:decisionRef', val)}
                                    placeholder="e.g. approve_decision"
                                    className="bg-white"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>Result Variable</Label>
                                    <DebouncedInput
                                    value={brResultVariable}
                                    onValueChange={(val) => updateField('@_goflow:resultVariable', val)}
                                    placeholder="e.g. decisionResult"
                                    className="bg-white"
                                    />
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Service Task Properties */}
                    {isServiceTask && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Service Task</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Topic</Label>
                                    <DebouncedInput
                                    value={topic}
                                    onValueChange={(val) => updateField('@_goflow:topic', val)}
                                    placeholder="e.g. payment_processing"
                                    className="bg-white"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>Task Type</Label>
                                    <DebouncedInput
                                    value={taskType}
                                    onValueChange={(val) => updateField('@_goflow:taskType', val)}
                                    placeholder="e.g. external"
                                    className="bg-white"
                                    />
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Script Task Properties */}
                    {isScriptTask && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Script Task</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Script Format</Label>
                                    <DebouncedInput
                                    value={scriptFormat}
                                    onValueChange={(val) => updateField('@_scriptFormat', val)}
                                    placeholder="e.g. javascript"
                                    className="bg-white"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>Result Variable</Label>
                                    <DebouncedInput
                                    value={scriptResultVariable}
                                    onValueChange={(val) => updateField('@_goflow:resultVariable', val)}
                                    placeholder="e.g. scriptResult"
                                    className="bg-white"
                                    />
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Call Activity Properties */}
                    {isCallActivity && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Call Activity</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Called Element</Label>
                                    <DebouncedInput
                                    value={calledElement}
                                    onValueChange={(val) => updateField('@_calledElement', val)}
                                    placeholder="e.g. process_id"
                                    className="bg-white"
                                    />
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Sequence Flow Properties */}
                    {isSequenceFlow && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Sequence Flow</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Condition Expression</Label>
                                    <DebouncedInput
                                    value={condition}
                                    onValueChange={(val) => updateField('bpmn:conditionExpression', val)}
                                    placeholder="e.g. amount > 1000"
                                    className="bg-white"
                                    />
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Gateway Properties */}
                    {isGateway && (originalType === 'bpmn:exclusiveGateway' || originalType === 'bpmn:inclusiveGateway') && (
                        <div className="space-y-3 pt-2 border-t">
                            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Gateway</h4>
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <Label>Default Flow ID</Label>
                                    <DebouncedInput
                                    value={defaultFlow}
                                    onValueChange={(val) => updateField('@_default', val)}
                                    placeholder="e.g. Flow_xyz"
                                    className="bg-white"
                                    />
                                    <p className="text-[10px] text-muted-foreground">The flow to take if no other conditions match.</p>
                                </div>
                            </div>
                        </div>
                    )}
                </TabsContent>

                <TabsContent value="extensions" className="flex-1 overflow-y-auto p-4 m-0">
                     {/* Extension Properties (Available for all elements) */}
                    <ExtensionProps data={element.data || null} onUpdate={(newData) => onUpdate(element.id, newData)} />
                </TabsContent>
            </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
