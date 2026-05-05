import { useMemo } from 'react';
import { ReactFlow, Background, Controls, ReactFlowProvider, ConnectionMode, MarkerType, type Node, type Edge } from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { parseBpmnXml } from '@/lib/bpmn-parser';
import { type Execution } from '@/lib/api';
import { TaskNode } from './nodes/TaskNode';
import { EventNode } from './nodes/EventNode';
import { GatewayNode } from './nodes/GatewayNode';
import { AlertCircle } from 'lucide-react';

const nodeTypes = {
  startEvent: EventNode,
  endEvent: EventNode,
  intermediateCatchEvent: EventNode,
  intermediateThrowEvent: EventNode,
  boundaryEvent: EventNode,
  userTask: TaskNode,
  serviceTask: TaskNode,
  scriptTask: TaskNode,
  businessRuleTask: TaskNode,
  receiveTask: TaskNode,
  manualTask: TaskNode,
  callActivity: TaskNode,
  subProcess: TaskNode,
  exclusiveGateway: GatewayNode,
  parallelGateway: GatewayNode,
  inclusiveGateway: GatewayNode,
  eventBasedGateway: GatewayNode,
};

const defaultEdgeOptions = {
  type: 'smoothstep',
  markerEnd: {
    type: MarkerType.ArrowClosed,
    width: 20,
    height: 20,
    color: '#334155',
  },
  style: {
    strokeWidth: 2,
    stroke: '#334155',
  },
};

interface BpmnViewerProps {
  xml: string;
  executions?: Execution[];
  activeStepId?: string;
}

function BpmnViewerContent({ xml, executions, activeStepId }: BpmnViewerProps) {
  const { nodes, edges, error } = useMemo(() => {
    if (!xml) {
        return { nodes: [] as Node[], edges: [] as Edge[], error: null };
    }

    try {
      const result = parseBpmnXml(xml);
      
      // Map execution status to nodes
      const nodesWithStatus: Node[] = result.nodes.map(node => {
        let executionStatus = 'pending';
        
        if (executions && executions.length > 0) {
           const nodeExecs = executions.filter(e => e.step_id === node.id);
           if (nodeExecs.length > 0) {
               const hasActive = nodeExecs.some(e => e.status === 'ACTIVE');
               const hasCompleted = nodeExecs.some(e => e.status === 'COMPLETED');
               
               if (hasActive) {
                   executionStatus = 'active';
               } else if (hasCompleted) {
                   executionStatus = 'completed';
               }
           }
        } else if (activeStepId && node.id === activeStepId) {
          executionStatus = 'active';
        } 
        
        return {
          ...node,
          data: {
            ...node.data,
            executionStatus
          }
        };
      });

      return { nodes: nodesWithStatus, edges: result.edges as Edge[], error: null };
    } catch (err: unknown) {
      console.error("Failed to parse BPMN XML:", err);
      const errorMessage = err instanceof Error ? err.message : "Failed to parse BPMN";
      return { nodes: [] as Node[], edges: [] as Edge[], error: errorMessage };
    }
  }, [xml, executions, activeStepId]);

  return (
    <div className="h-full w-full border rounded-lg overflow-hidden bg-gray-50 relative">
      {error && (
        <div className="absolute top-4 left-4 z-50 bg-red-50 text-red-600 p-3 rounded border border-red-200 shadow-sm flex items-center gap-2 text-sm max-w-md">
            <AlertCircle className="w-4 h-4" />
            <span>{error}</span>
        </div>
      )}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={defaultEdgeOptions}
        connectionMode={ConnectionMode.Loose}
        fitView
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}

export default function BpmnViewer(props: BpmnViewerProps) {
  return (
    <ReactFlowProvider>
      <BpmnViewerContent {...props} />
    </ReactFlowProvider>
  );
}
