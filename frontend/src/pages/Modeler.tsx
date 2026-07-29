import { useCallback, useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router";
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
import { VisualArtifactNode } from "@/components/bpmn/nodes/VisualArtifactNode";
import { setPendingWorkflowSync, waitForWorkflowInCatalog } from "@/lib/cqrsSync";
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
  sendTask: TaskNode,
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
  visualArtifact: VisualArtifactNode,
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

type DiagramLoadStatus = "idle" | "loading" | "ready" | "error";

function ModelerContent() {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { screenToFlowPosition } = useReactFlow();
  
  const [processId, setProcessId] = useState("Process_1");
  const [processName, setProcessName] = useState("New Process");
  const [selectedElementId, setSelectedElementId] = useState<string | null>(null);
  const [loadStatus, setLoadStatus] = useState<DiagramLoadStatus>("idle");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [deploySyncNotice, setDeploySyncNotice] = useState<string | null>(null);
  const [deploySyncing, setDeploySyncing] = useState(false);
  const [deployError, setDeployError] = useState<string | null>(null);

  const isNewDiagram = searchParams.get("new") === "true";
  const workflowId = searchParams.get("id");
  const isEditMode = !isNewDiagram && Boolean(workflowId);
  const canEditDiagram = !isEditMode || loadStatus === "ready";

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
    (params: Connection) => {
      if (!canEditDiagram) return;
      setEdges((eds) => addEdge(params, eds));
    },
    [canEditDiagram, setEdges],
  );

  const onReconnect = useCallback(
    (oldEdge: Edge, newConnection: Connection) => {
      if (!canEditDiagram) return;
      setEdges((els) => reconnectEdge(oldEdge, newConnection, els));
    },
    [canEditDiagram, setEdges],
  );

  const loadDiagram = useCallback(async () => {
    const isNew = searchParams.get("new") === "true";
    const idParam = searchParams.get("id");
    setLoadError(null);

    if (isNew) {
      // Initialize from query params
      const paramName = searchParams.get("name");
      // When new=true, 'id' param is the BPMN Process ID (e.g. Process_1)
      // When new=false, 'id' param is the Database UUID
      const paramId = searchParams.get("id");

      let finalName = paramName || "";
      let finalId = paramId || "";

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
      setLoadStatus("ready");
      return;
    }

    if (!idParam) {
      setLoadStatus("ready");
      return;
    }

    setLoadStatus("loading");
    try {
      const wf = await api.getWorkflow(idParam);
      if (!wf.bpmn_xml) {
        throw new Error("Workflow has no BPMN XML to load");
      }

      const result = parseBpmnXml(wf.bpmn_xml);
      setNodes(result.nodes);
      setEdges(result.edges);
      setProcessId(result.processId);
      setProcessName(result.processName);
      setLoadStatus("ready");
    } catch (err) {
      console.error("Failed to load workflow:", err);
      setLoadError(err instanceof Error ? err.message : "Failed to load workflow");
      setLoadStatus("error");
    }
  }, [searchParams, setNodes, setEdges]);

  // Load Diagram
  useEffect(() => {
    loadDiagram();
  }, [loadDiagram]);

  const handleSaveXML = () => {
    if (!canEditDiagram) return;
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
    if (!canEditDiagram) return;
    setDeployError(null);
    try {
        if (!processId.trim()) {
          setDeployError("Process ID is required before deploy.");
          return;
        }
        if (!nodes.some((n) => n.type === "startEvent" || n.data?.originalType === "bpmn:startEvent")) {
          setDeployError("BPMN lint: process needs at least one start event.");
          return;
        }
        // Visual-only nodes must not be wired into sequence flows (engine has no tokens for them).
        const visualIds = new Set(
          nodes.filter((n) => n.data?.visualOnly || n.type === "visualArtifact").map((n) => n.id)
        );
        const badEdge = edges.find((e) => visualIds.has(e.source) || visualIds.has(e.target));
        if (badEdge) {
          setDeployError("BPMN lint: visual-only shapes (pool/lane/data) cannot be connected with sequence flows.");
          return;
        }
        const xml = generateBpmnXml(nodes, edges, processId, processName);
        const wf = await api.deployWorkflow(xml);
        setPendingWorkflowSync(wf.id);
        setDeploySyncing(true);
        setDeploySyncNotice("Deploy accepted. Syncing process catalog from the query projection...");
        if (searchParams.get("new") === "true") {
             navigate(`/modeler?id=${wf.id}`, { replace: true });
        }
        const result = await waitForWorkflowInCatalog(wf.id, {
          processDefinitionId: wf.process_definition_id,
          version: wf.version,
        });
        setDeploySyncNotice(
          result === "synced"
            ? "Deploy complete. Process catalog is up to date."
            : "Deploy succeeded. Processes list may take longer to update; use Refresh on Processes.",
        );
    } catch (err) {
        console.error("Deploy failed", err);
        const message = err instanceof Error ? err.message : "Failed to deploy workflow";
        setDeployError(message);
    } finally {
        setDeploySyncing(false);
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
      if (!canEditDiagram) return;

      const type = event.dataTransfer.getData('application/reactflow/type');
      const rawOriginalType = event.dataTransfer.getData('application/reactflow/originalType');
      const label = event.dataTransfer.getData('application/reactflow/label');

      // check if the dropped element is valid
      if (typeof type === 'undefined' || !type) {
        return;
      }

      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      // Palette may use "bpmn:endEvent:terminate" style markers for event definitions.
      const [baseType, eventKind] = rawOriginalType.split(":")[0] === "bpmn"
        ? (() => {
            const parts = rawOriginalType.split(":");
            if (parts.length >= 3) return [`${parts[0]}:${parts[1]}`, parts[2]];
            return [rawOriginalType, ""];
          })()
        : [rawOriginalType, ""];
      const originalType = baseType;
      const size = getElementSize(originalType);
      const visualKind = type === "visualArtifact" ? originalType.replace("bpmn:", "") : undefined;

      const newNode: Node = {
        id: `${type}_${Date.now()}`,
        type,
        position,
        data: {
          label,
          originalType,
          ...(eventKind ? { eventDefinitionType: eventKind } : {}),
          ...(visualKind ? { visualKind, visualOnly: true } : {}),
          ...(eventKind === "link" ? { link_name: "Link_1" } : {}),
          ...(eventKind === "escalation" ? { escalation_code: "ESC_1" } : {}),
          ...(eventKind === "conditional" ? { condition: "true" } : {}),
        },
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
    [canEditDiagram, screenToFlowPosition, setNodes],
  );

  // Find the actual object for properties panel
  const selectedElement = 
    nodes.find((n) => n.id === selectedElementId) || 
    edges.find((e) => e.id === selectedElementId) || 
    null;

  const handleUpdateElement = (id: string, newData: Record<string, unknown>) => {
    if (!canEditDiagram) return;
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

  const handleRenameId = (oldId: string, newId: string): string | null => {
    if (!canEditDiagram) return "Diagram is read-only";
    if (nodes.some((n) => n.id === newId) || edges.some((e) => e.id === newId)) {
      return "ID must be unique in this diagram";
    }
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === oldId) {
          return { ...node, id: newId };
        }
        return node;
      }),
    );
    setEdges((eds) =>
      eds.map((edge) => {
        let next = edge;
        if (edge.id === oldId) {
          next = { ...next, id: newId };
        }
        if (edge.source === oldId) {
          next = { ...next, source: newId };
        }
        if (edge.target === oldId) {
          next = { ...next, target: newId };
        }
        const attached = next.data?.["@_attachedToRef"];
        if (attached === oldId) {
          next = {
            ...next,
            data: { ...next.data, "@_attachedToRef": newId },
          };
        }
        return next;
      }),
    );
    setNodes((nds) =>
      nds.map((node) => {
        const attached = node.data?.["@_attachedToRef"];
        if (attached === oldId) {
          return { ...node, data: { ...node.data, "@_attachedToRef": newId } };
        }
        return node;
      }),
    );
    if (selectedElementId === oldId) {
      setSelectedElementId(newId);
    }
    return null;
  };

  const diagramIds = [
    ...nodes.map((n) => n.id),
    ...edges.map((e) => e.id),
  ];

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
          <Button variant="outline" size="sm" onClick={handleSaveXML} disabled={!canEditDiagram}>
            <Save className="mr-2 h-4 w-4" />
            Save XML
          </Button>
        </div>
        <div className="flex items-center space-x-2">
           <Button variant="default" size="sm" onClick={handleDeploy} disabled={!canEditDiagram || deploySyncing}>
            <Play className="mr-2 h-4 w-4" />
            Deploy Process
          </Button>
        </div>
      </div>

      {isEditMode && loadStatus === "loading" ? (
        <div className="rounded-md border bg-muted p-3 text-sm text-muted-foreground">
          Loading workflow...
        </div>
      ) : null}

      {isEditMode && loadStatus === "error" ? (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          <div className="font-medium">Failed to load workflow</div>
          <div className="mt-1">{loadError}</div>
          <div className="mt-3 flex gap-2">
            <Button variant="outline" size="sm" onClick={loadDiagram}>
              Retry
            </Button>
            <Button variant="outline" size="sm" onClick={() => navigate("/processes")}>
              Back to Processes
            </Button>
          </div>
        </div>
      ) : null}

      {deployError ? (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive whitespace-pre-wrap" role="alert">
          <div className="font-semibold mb-1">BPMN validation / deploy failed</div>
          {deployError}
        </div>
      ) : null}
      {deploySyncNotice ? (
        <div className="rounded-md border bg-muted p-3 text-sm text-muted-foreground">
          {deploySyncing ? "Syncing... " : null}
          {deploySyncNotice}
        </div>
      ) : null}

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
                nodesDraggable={canEditDiagram}
                nodesConnectable={canEditDiagram}
                elementsSelectable={canEditDiagram}
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
            onRenameId={handleRenameId}
            existingIds={diagramIds} 
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
