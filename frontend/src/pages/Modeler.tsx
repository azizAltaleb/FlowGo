import { useCallback, useEffect, useRef, useState } from "react";
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
  type NodeChange,
  type EdgeChange,
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
import { setArtificialFlowAttribute } from "@/lib/bpmn-namespaces";
import { VisualArtifactNode } from "@/components/bpmn/nodes/VisualArtifactNode";
import { setPendingWorkflowSync, waitForWorkflowInCatalog } from "@/lib/cqrsSync";
import PropertiesPanel from "@/components/bpmn/PropertiesPanel";
import Palette from "@/components/bpmn/Palette";
import { TaskNode } from "@/components/bpmn/nodes/TaskNode";
import { EventNode } from "@/components/bpmn/nodes/EventNode";
import { GatewayNode } from "@/components/bpmn/nodes/GatewayNode";
import { useDiagramHistory } from "@/components/bpmn/useDiagramHistory";

/** Palette markers that preselect a connector job type on service/send tasks. */
const CONNECTOR_PALETTE_KINDS: Record<string, string> = {
  http: "io.artificialflow.connector.http",
  webhook: "io.artificialflow.connector.webhook",
  kafka: "io.artificialflow.connector.kafka",
  email: "io.artificialflow.connector.email",
  s3: "io.artificialflow.connector.s3",
};

const ATTACHABLE_NODE_TYPES = new Set([
  "userTask",
  "serviceTask",
  "scriptTask",
  "businessRuleTask",
  "sendTask",
  "receiveTask",
  "manualTask",
  "callActivity",
  "subProcess",
]);

import { Button } from "@/components/ui/button";
import { Save, Play, Undo2, Redo2 } from "lucide-react";

/**
 * Initial viewport: keep a readable zoom. minZoom here is a floor for fitView so
 * wide processes are NOT shrunk to a thin strip — user pans instead.
 */
const DIAGRAM_ZOOM = 1.25;
const FIT_VIEW_OPTIONS = { padding: 0.2, maxZoom: DIAGRAM_ZOOM, minZoom: DIAGRAM_ZOOM } as const;

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
  const { screenToFlowPosition, fitView, setCenter } = useReactFlow();
  const nodesRef = useRef(nodes);
  const edgesRef = useRef(edges);
  const fittedLoadKeyRef = useRef<string | null>(null);

  useEffect(() => {
    nodesRef.current = nodes;
  }, [nodes]);
  useEffect(() => {
    edgesRef.current = edges;
  }, [edges]);

  const getGraph = useCallback(
    () => ({ nodes: nodesRef.current, edges: edgesRef.current }),
    [],
  );
  const applyGraph = useCallback(
    (snapshot: { nodes: Node[]; edges: Edge[] }) => {
      setNodes(snapshot.nodes);
      setEdges(snapshot.edges);
      nodesRef.current = snapshot.nodes;
      edgesRef.current = snapshot.edges;
    },
    [setNodes, setEdges],
  );
  const { record, undo, redo, reset: resetHistory } = useDiagramHistory(getGraph, applyGraph);
  
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
      record();
      setEdges((eds) => addEdge(params, eds));
    },
    [canEditDiagram, record, setEdges],
  );

  const onReconnect = useCallback(
    (oldEdge: Edge, newConnection: Connection) => {
      if (!canEditDiagram) return;
      record();
      setEdges((els) => reconnectEdge(oldEdge, newConnection, els));
    },
    [canEditDiagram, record, setEdges],
  );

  const handleNodesChange = useCallback(
    (changes: NodeChange<Node>[]) => {
      const structural = changes.some(
        (c) => c.type === "remove" || c.type === "add",
      );
      if (structural && canEditDiagram) record();
      onNodesChange(changes);
    },
    [canEditDiagram, onNodesChange, record],
  );

  const handleEdgesChange = useCallback(
    (changes: EdgeChange<Edge>[]) => {
      const structural = changes.some(
        (c) => c.type === "remove" || c.type === "add",
      );
      if (structural && canEditDiagram) record();
      onEdgesChange(changes);
    },
    [canEditDiagram, onEdgesChange, record],
  );

  const onNodeDragStart = useCallback(() => {
    if (!canEditDiagram) return;
    record();
  }, [canEditDiagram, record]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!canEditDiagram) return;
      const target = event.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "SELECT" ||
          target.isContentEditable)
      ) {
        return;
      }
      const mod = event.metaKey || event.ctrlKey;
      if (!mod) return;
      const key = event.key.toLowerCase();
      if (key === "z" && !event.shiftKey) {
        event.preventDefault();
        undo();
        return;
      }
      if (key === "y" || (key === "z" && event.shiftKey)) {
        event.preventDefault();
        redo();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [canEditDiagram, undo, redo]);

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
      resetHistory();
      setLoadStatus("ready");
      return;
    }

    if (!idParam) {
      resetHistory();
      setLoadStatus("ready");
      return;
    }

    setLoadStatus("loading");
    fittedLoadKeyRef.current = null;
    try {
      const wf = await api.getWorkflow(idParam);
      if (!wf.bpmn_xml) {
        throw new Error("Workflow has no BPMN XML to load");
      }

      const result = parseBpmnXml(wf.bpmn_xml);
      setNodes(result.nodes);
      setEdges(result.edges);
      nodesRef.current = result.nodes;
      edgesRef.current = result.edges;
      setProcessId(result.processId);
      setProcessName(result.processName);
      resetHistory();
      setLoadStatus("ready");
    } catch (err) {
      console.error("Failed to load workflow:", err);
      setLoadError(err instanceof Error ? err.message : "Failed to load workflow");
      setLoadStatus("error");
    }
  }, [searchParams, setNodes, setEdges, resetHistory]);

  // Load Diagram
  useEffect(() => {
    loadDiagram();
  }, [loadDiagram]);

  // After load: open at a readable zoom centered on the start (or first) node.
  // Avoid fitView of the whole graph — wide processes were shrinking to a strip.
  useEffect(() => {
    if (loadStatus !== "ready") return;
    const key = `${workflowId || "new"}:${processId}`;
    if (fittedLoadKeyRef.current === key) return;
    fittedLoadKeyRef.current = key;
    const t = window.setTimeout(() => {
      const graph = nodesRef.current;
      const focus =
        graph.find((n) => n.type === "startEvent" || n.data?.originalType === "bpmn:startEvent") ||
        graph[0];
      if (focus) {
        const w = Number(focus.style?.width ?? focus.width ?? 100);
        const h = Number(focus.style?.height ?? focus.height ?? 80);
        setCenter(focus.position.x + w / 2, focus.position.y + h / 2, {
          zoom: DIAGRAM_ZOOM,
          duration: 220,
        });
      } else {
        fitView({ ...FIT_VIEW_OPTIONS, duration: 220 });
      }
    }, 80);
    return () => window.clearTimeout(t);
  }, [loadStatus, workflowId, processId, fitView, setCenter]);

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
      record();

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
      const size = getElementSize(originalType === "bpmn:transaction" ? "bpmn:subProcess" : originalType);
      const visualKind = type === "visualArtifact" ? originalType.replace("bpmn:", "") : undefined;
      const connectorJobType =
        originalType === "bpmn:serviceTask" || originalType === "bpmn:sendTask"
          ? CONNECTOR_PALETTE_KINDS[eventKind]
          : undefined;
      // Event-definition markers vs connector palette markers on service/send tasks.
      const eventDefinitionKind = connectorJobType ? "" : eventKind;

      // Auto-attach boundary events to the nearest activity under the drop point.
      let attachedToRef = "";
      if (type === "boundaryEvent") {
        let bestDist = Number.POSITIVE_INFINITY;
        for (const n of nodes) {
          if (!ATTACHABLE_NODE_TYPES.has(String(n.type || ""))) continue;
          const w = Number(n.style?.width ?? n.width ?? 100);
          const h = Number(n.style?.height ?? n.height ?? 80);
          const cx = n.position.x + w / 2;
          const cy = n.position.y + h / 2;
          const dist = Math.hypot(position.x - cx, position.y - cy);
          if (dist < bestDist) {
            bestDist = dist;
            attachedToRef = n.id;
          }
        }
        // Only snap when the drop is reasonably near an activity (~2× diagonal of a task).
        if (bestDist > 220) attachedToRef = "";
      }

      let nodeData: Record<string, unknown> = {
        label,
        originalType,
        ...(eventDefinitionKind ? { eventDefinitionType: eventDefinitionKind } : {}),
        ...(visualKind ? { visualKind, visualOnly: true } : {}),
        ...(originalType === "bpmn:transaction" ? { transaction: true } : {}),
        ...(eventDefinitionKind === "link"
          ? { link_name: "Link_1", "bpmn:linkEventDefinition": { "@_name": "Link_1" } }
          : {}),
        ...(eventDefinitionKind === "escalation"
          ? {
              escalation_code: "ESC_1",
              escalation_id: "Escalation_ESC_1",
              "@_artificialflow:escalationCode": "ESC_1",
              "bpmn:escalationEventDefinition": { "@_escalationRef": "Escalation_ESC_1" },
            }
          : {}),
        ...(eventDefinitionKind === "conditional"
          ? {
              condition: "true",
              "bpmn:conditionalEventDefinition": { "bpmn:condition": { "#text": "true" } },
            }
          : {}),
        ...(eventDefinitionKind === "timer"
          ? { "bpmn:timerEventDefinition": { "bpmn:timeDuration": "PT1H" } }
          : {}),
        ...(eventDefinitionKind === "message"
          ? {
              message_name: "Message1",
              message_id: "Message_Message1",
              "bpmn:messageEventDefinition": { "@_messageRef": "Message_Message1" },
            }
          : {}),
        ...(eventDefinitionKind === "signal"
          ? {
              signal_name: "Signal1",
              signal_id: "Signal_Signal1",
              "bpmn:signalEventDefinition": { "@_signalRef": "Signal_Signal1" },
            }
          : {}),
        ...(eventDefinitionKind === "error"
          ? {
              error_code: "ERROR_1",
              error_id: "Error_ERROR_1",
              "@_artificialflow:errorCode": "ERROR_1",
              "bpmn:errorEventDefinition": { "@_errorRef": "Error_ERROR_1" },
            }
          : {}),
        ...(eventDefinitionKind === "compensate"
          ? {
              activity_ref: "",
              "bpmn:compensateEventDefinition": { "@_activityRef": "" },
            }
          : {}),
        ...(eventDefinitionKind === "cancel" ? { "bpmn:cancelEventDefinition": {} } : {}),
        ...(eventDefinitionKind === "terminate" ? { "bpmn:terminateEventDefinition": {} } : {}),
        ...(type === "boundaryEvent"
          ? {
              "@_attachedToRef": attachedToRef,
              "@_cancelActivity":
                eventDefinitionKind === "error" || eventDefinitionKind === "cancel" ? "true" : "false",
            }
          : {}),
      };
      if (connectorJobType) {
        nodeData = setArtificialFlowAttribute(nodeData, "taskType", connectorJobType);
      }
      if (eventDefinitionKind === "message" || originalType === "bpmn:receiveTask") {
        nodeData = setArtificialFlowAttribute(nodeData, "correlationKey", "correlationKey");
      }
      if (originalType === "bpmn:receiveTask") {
        nodeData["@_messageRef"] = "Message_Message1";
        nodeData.message_name = "Message1";
        nodeData.message_id = "Message_Message1";
      }

      const newNode: Node = {
        id: `${type}_${Date.now()}`,
        type,
        position,
        data: nodeData,
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
    [canEditDiagram, nodes, record, screenToFlowPosition, setNodes],
  );

  // Find the actual object for properties panel
  const selectedElement = 
    nodes.find((n) => n.id === selectedElementId) || 
    edges.find((e) => e.id === selectedElementId) || 
    null;

  const handleUpdateElement = (id: string, newData: Record<string, unknown>) => {
    if (!canEditDiagram) return;
    record();
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
    record();
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
          <Button
            variant="outline"
            size="sm"
            onClick={() => undo()}
            disabled={!canEditDiagram}
            title="Undo (⌘Z / Ctrl+Z)"
          >
            <Undo2 className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => redo()}
            disabled={!canEditDiagram}
            title="Redo (⌘Y / Ctrl+Y)"
          >
            <Redo2 className="h-4 w-4" />
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
                onNodesChange={handleNodesChange}
                onEdgesChange={handleEdgesChange}
                onConnect={onConnect}
                onReconnect={onReconnect}
                onNodeDragStart={onNodeDragStart}
                onSelectionChange={onSelectionChange}
                onDrop={onDrop}
                onDragOver={onDragOver}
                nodeTypes={nodeTypes}
                edgeTypes={edgeTypes}
                nodesDraggable={canEditDiagram}
                nodesConnectable={canEditDiagram}
                elementsSelectable={canEditDiagram}
                connectionMode={ConnectionMode.Loose}
                fitView={false}
                fitViewOptions={FIT_VIEW_OPTIONS}
                minZoom={0.2}
                maxZoom={2}
                defaultViewport={{ x: 0, y: 0, zoom: DIAGRAM_ZOOM }}
                snapToGrid={true}
                snapGrid={[15, 15]}
                defaultEdgeOptions={defaultEdgeOptions}
                deleteKeyCode={['Backspace', 'Delete']}
                multiSelectionKeyCode={['Meta', 'Shift', 'Ctrl']}
            >
                <Background color="#ccc" gap={15} size={1} />
                <Controls showInteractive={canEditDiagram} />
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
            nodes={nodes}
            edges={edges}
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
