import { useCallback, useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { 
  ReactFlow, 
  Controls, 
  Background, 
  useNodesState, 
  useEdgesState, 
  addEdge, 
  reconnectEdge,
  type Connection, 
  type Edge, 
  type Node, 
  ReactFlowProvider,
  Panel,
  type OnSelectionChangeParams,
  useReactFlow,
  MarkerType,
  MiniMap,
  ConnectionMode
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { api } from "@/lib/api";
import { parseBpmnXml, generateBpmnXml, getElementSize } from "@/lib/bpmn-parser";
import PropertiesPanel from "@/components/bpmn/PropertiesPanel";
import Palette from "@/components/bpmn/Palette";
import { TaskNode } from "@/components/bpmn/nodes/TaskNode";
import { EventNode } from "@/components/bpmn/nodes/EventNode";
import { GatewayNode } from "@/components/bpmn/nodes/GatewayNode";

import { Button } from "@/components/ui/button";
import { Save, Play } from "lucide-react";

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

const edgeTypes = {
  // floating: FloatingEdge,
};

// Initial empty graph
const initialNodes: Node[] = [
  { 
    id: 'StartEvent_1', 
    type: 'startEvent', 
    position: { x: 250, y: 250 }, 
    data: { label: 'Start', originalType: 'bpmn:startEvent' },
    style: { width: 36, height: 36 }
  }
];

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

function ModelerContent() {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { screenToFlowPosition } = useReactFlow();
  
  const [processId, setProcessId] = useState("Process_1");
  const [processName, setProcessName] = useState("New Process");
  const [selectedElementId, setSelectedElementId] = useState<string | null>(null);

  // Selection handling
  const onSelectionChange = useCallback(({ nodes, edges }: OnSelectionChangeParams) => {
    if (nodes.length > 0) {
      setSelectedElementId(nodes[0].id);
    } else if (edges.length > 0) {
      setSelectedElementId(edges[0].id);
    } else {
      setSelectedElementId(null);
    }
  }, []);

  // Connection handling
  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  const onReconnect = useCallback(
    (oldEdge: Edge, newConnection: Connection) => {
      setEdges((els) => reconnectEdge(oldEdge, newConnection, els));
    },
    [setEdges],
  );

  // Load Diagram
  useEffect(() => {
    const loadDiagram = async () => {
      const isNew = searchParams.get("new") === "true";
      const idParam = searchParams.get("id");
      
      if (isNew) {
          // Initialize from query params
          const paramName = searchParams.get("name");
          // When new=true, 'id' param is the BPMN Process ID (e.g. Process_1)
          // When new=false, 'id' param is the Database UUID
          const paramId = searchParams.get("id");

          let finalName = paramName;
          let finalId = paramId;

          // If missing, prompt user as fallback
          if (!finalName) {
             finalName = window.prompt("Enter Process Name:", "New Process") || "New Process";
          }
          
          if (!finalId) {
             const suggested = finalName ? finalName.replace(/\s+/g, '_') : `Process_${Date.now()}`;
             finalId = window.prompt("Enter Process ID:", suggested) || suggested;
          }

          setProcessName(finalName);
          setProcessId(finalId);

      } else if (idParam) {
        try {
          const wf = await api.getWorkflow(idParam);
          if (wf.bpmn_xml) {
             const result = parseBpmnXml(wf.bpmn_xml);
             setNodes(result.nodes);
             setEdges(result.edges);
             setProcessId(result.processId);
             setProcessName(result.processName);
          }
        } catch (err) {
          console.error("Failed to load workflow:", err);
        }
      }
    };
    loadDiagram();
  }, [searchParams, setNodes, setEdges]);

  const handleSaveXML = () => {
    const xml = generateBpmnXml(nodes, edges, processId, processName);
    const blob = new Blob([xml], { type: 'text/xml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${processName.replace(/\s+/g, '_')}.bpmn`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };
  
  const handleDeploy = async () => {
    try {
        const xml = generateBpmnXml(nodes, edges, processId, processName);
        const wf = await api.deployWorkflow(xml);
        alert("Workflow deployed successfully!");
        if (searchParams.get("new") === "true") {
             navigate(`/modeler?id=${wf.id}`, { replace: true });
        }
    } catch (err) {
        console.error("Deploy failed", err);
        alert("Failed to deploy workflow");
    }
  };

  // Drag and Drop
  const onDragStart = (event: React.DragEvent, nodeType: string, originalType: string, label: string) => {
    event.dataTransfer.setData('application/reactflow/type', nodeType);
    event.dataTransfer.setData('application/reactflow/originalType', originalType);
    event.dataTransfer.setData('application/reactflow/label', label);
    event.dataTransfer.effectAllowed = 'move';
  };

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const type = event.dataTransfer.getData('application/reactflow/type');
      const originalType = event.dataTransfer.getData('application/reactflow/originalType');
      const label = event.dataTransfer.getData('application/reactflow/label');

      // check if the dropped element is valid
      if (typeof type === 'undefined' || !type) {
        return;
      }

      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const size = getElementSize(originalType);

      const newNode: Node = {
        id: `${type}_${Date.now()}`,
        type,
        position,
        data: { label, originalType },
        style: { width: size.width, height: size.height },
        selected: true, // Auto-select on drop
      };

      // Deselect other nodes and add the new one
      setNodes((nds) => [
        ...nds.map((n) => ({ ...n, selected: false })),
        newNode
      ]);
      setSelectedElementId(newNode.id); // Immediately show properties
    },
    [screenToFlowPosition, setNodes],
  );

  // Find the actual object for properties panel
  const selectedElement = 
    nodes.find((n) => n.id === selectedElementId) || 
    edges.find((e) => e.id === selectedElementId) || 
    null;

  const handleUpdateElement = (id: string, newData: Record<string, unknown>) => {
    setNodes((nds) => nds.map((node) => {
      if (node.id === id) {
        return { ...node, data: newData };
      }
      return node;
    }));
    setEdges((eds) => eds.map((edge) => {
      if (edge.id === id) {
        const label = typeof newData.label === 'string' ? newData.label : undefined;
        return { ...edge, data: newData, label: label || edge.label };
      }
      return edge;
    }));
  };

  return (
    <div className="h-full flex flex-col space-y-4">
        {/* Toolbar */}
      <div className="flex justify-between items-center bg-card p-2 rounded-lg border">
        <div className="flex items-center space-x-2">
          {/* Add Node Dropdown Removed - Replaced by Palette */}
          <div className="px-2 flex flex-col">
              <span className="text-sm font-semibold text-foreground">{processName}</span>
              <span className="text-xs text-muted-foreground">{processId}</span>
          </div>
          <Button variant="outline" size="sm" onClick={handleSaveXML}>
            <Save className="mr-2 h-4 w-4" />
            Save XML
          </Button>
        </div>
        <div className="flex items-center space-x-2">
           <Button variant="default" size="sm" onClick={handleDeploy}>
            <Play className="mr-2 h-4 w-4" />
            Deploy Process
          </Button>
        </div>
      </div>

      <div className="flex-1 flex border rounded-lg overflow-hidden bg-white">
        <Palette onDragStart={onDragStart} />
        
        <div className="flex-1 relative h-full">
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                onReconnect={onReconnect}
                onSelectionChange={onSelectionChange}
                onDrop={onDrop}
                onDragOver={onDragOver}
                nodeTypes={nodeTypes}
                edgeTypes={edgeTypes}
                connectionMode={ConnectionMode.Loose}
                fitView
                snapToGrid={true}
                snapGrid={[15, 15]}
                defaultEdgeOptions={defaultEdgeOptions}
                deleteKeyCode={['Backspace', 'Delete']}
                multiSelectionKeyCode={['Meta', 'Shift', 'Ctrl']}
            >
                <Background color="#ccc" gap={15} size={1} />
                <Controls />
                <MiniMap className="border rounded shadow-sm" zoomable pannable />
                <Panel position="top-right" className="bg-white/80 p-2 rounded text-xs text-gray-500">
                    Standard BPMN 2.0
                </Panel>
            </ReactFlow>
        </div>
        <div className="w-[300px] border-l bg-gray-50 overflow-y-auto">
          <PropertiesPanel 
            key={selectedElement?.id} 
            element={selectedElement} 
            onUpdate={handleUpdateElement} 
          />
        </div>
      </div>
    </div>
  );
}

export default function Modeler() {
    return (
        <ReactFlowProvider>
            <ModelerContent />
        </ReactFlowProvider>
    );
}
