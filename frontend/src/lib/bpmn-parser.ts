import { XMLParser, XMLBuilder } from "fast-xml-parser";
import { type Edge, type Node } from "@xyflow/react";

const BPMN_NS = "http://www.omg.org/spec/BPMN/20100524/MODEL";
const BPMNDI_NS = "http://www.omg.org/spec/BPMN/20100524/DI";
const DC_NS = "http://www.omg.org/spec/DD/20100524/DC";
const DI_NS = "http://www.omg.org/spec/DD/20100524/DI";
const WORKFLOWSA_NS = "http://workflowsa.com/schema/1.0/bpmn";

const DEFAULT_OPTIONS = {
  ignoreAttributes: false,
  attributeNamePrefix: "@_",
  format: true,
  ignoreNameSpace: false,
  suppressBooleanAttributes: false,
  parseTagValue: false, // Ensure values stay as strings
  parseAttributeValue: false, // Ensure attributes stay as strings
};

export interface BpmnParseResult {
  nodes: Node[];
  edges: Edge[];
  processId: string;
  processName: string;
}

interface BPMNShape {
  "@_bpmnElement": string;
  "dc:Bounds": {
    "@_x": string;
    "@_y": string;
    "@_width": string;
    "@_height": string;
  };
}

interface BPMNElement {
  "@_id": string;
  "@_name"?: string;
  [key: string]: unknown;
}

interface BPMNFlow {
  "@_id": string;
  "@_sourceRef": string;
  "@_targetRef": string;
  "@_name"?: string;
  [key: string]: unknown;
}

export const getElementSize = (type: string) => {
  const t = type.replace('bpmn:', '');
  if (t.toLowerCase().includes('event')) return { width: 36, height: 36 };
  if (t.toLowerCase().includes('gateway')) return { width: 50, height: 50 };
  if (t.toLowerCase().includes('subprocess')) return { width: 200, height: 150 };
  return { width: 100, height: 80 }; // Default for tasks
};

export const parseBpmnXml = (xml: string): BpmnParseResult => {
  const parser = new XMLParser(DEFAULT_OPTIONS);
  const jsonObj = parser.parse(xml);

  const definitions = jsonObj["bpmn:definitions"];
  if (!definitions) {
    throw new Error("Invalid BPMN XML: Missing definitions");
  }

  const process = definitions["bpmn:process"];
  if (!process) {
    throw new Error("Invalid BPMN XML: Missing process");
  }
  
  const processId = process["@_id"] || "Process_1";
  const processName = process["@_name"] || "Process";

  const nodes: Node[] = [];
  const edges: Edge[] = [];

  // Helper to find shape bounds from BPMNDI
  const diagram = definitions["bpmndi:BPMNDiagram"];
  const plane = diagram ? diagram["bpmndi:BPMNPlane"] : null;
  const shapes = plane && plane["bpmndi:BPMNShape"] ? (Array.isArray(plane["bpmndi:BPMNShape"]) ? plane["bpmndi:BPMNShape"] : [plane["bpmndi:BPMNShape"]]) : [];
  
  const getBounds = (id: string, type: string) => {
    const shape = shapes.find((s: BPMNShape) => s["@_bpmnElement"] === id);
    if (shape && shape["dc:Bounds"]) {
      return {
        x: parseFloat(shape["dc:Bounds"]["@_x"]),
        y: parseFloat(shape["dc:Bounds"]["@_y"]),
        width: parseFloat(shape["dc:Bounds"]["@_width"]),
        height: parseFloat(shape["dc:Bounds"]["@_height"]),
      };
    }
    // Default sizes based on type
    const size = getElementSize(type);
    return { x: 0, y: 0, ...size };
  };

  // Parsing elements
  const elementTypes = [
    { tag: "bpmn:startEvent", type: "startEvent" },
    { tag: "bpmn:endEvent", type: "endEvent" },
    { tag: "bpmn:userTask", type: "userTask" },
    { tag: "bpmn:serviceTask", type: "serviceTask" },
    { tag: "bpmn:scriptTask", type: "scriptTask" },
    { tag: "bpmn:businessRuleTask", type: "businessRuleTask" },
    { tag: "bpmn:receiveTask", type: "receiveTask" },
    { tag: "bpmn:manualTask", type: "manualTask" },
    { tag: "bpmn:exclusiveGateway", type: "exclusiveGateway" },
    { tag: "bpmn:parallelGateway", type: "parallelGateway" },
    { tag: "bpmn:inclusiveGateway", type: "inclusiveGateway" },
    { tag: "bpmn:eventBasedGateway", type: "eventBasedGateway" },
    { tag: "bpmn:intermediateCatchEvent", type: "intermediateCatchEvent" },
    { tag: "bpmn:intermediateThrowEvent", type: "intermediateThrowEvent" },
    { tag: "bpmn:boundaryEvent", type: "boundaryEvent" },
    { tag: "bpmn:callActivity", type: "callActivity" },
    { tag: "bpmn:subProcess", type: "subProcess" },
  ];

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const parseContainer = (container: Record<string, any>, parentId?: string) => {
      elementTypes.forEach((def) => {
        const elements = container[def.tag];
        if (elements) {
          const list = Array.isArray(elements) ? elements : [elements];
          list.forEach((el: BPMNElement) => {
            const id = el["@_id"];
            const bounds = getBounds(id, def.tag);
            
            // Calculate relative position if parent exists
            let position = { x: bounds.x, y: bounds.y };
            if (parentId) {
                const parentBounds = getBounds(parentId, "bpmn:subProcess");
                position = {
                    x: bounds.x - parentBounds.x,
                    y: bounds.y - parentBounds.y
                };
            }

            // Extract properties
            const data: Record<string, unknown> = { 
                label: el["@_name"] || "",
                ...el 
            };
            // Clean up attributes from data for cleaner usage
            delete data["@_id"];
            delete data["@_name"];

            const node: Node = {
              id,
              type: def.type,
              position,
              data: { 
                originalType: def.tag,
                width: bounds.width,
                height: bounds.height,
                ...data
              },
              style: { width: bounds.width, height: bounds.height },
            };

            if (parentId) {
                node.parentId = parentId;
                node.extent = 'parent';
            }

            nodes.push(node);

            // Recurse if subprocess
            if (def.tag === 'bpmn:subProcess') {
                parseContainer(el, id);
            }
          });
        }
      });

      // Parse Sequence Flows in this container
      const flows = container["bpmn:sequenceFlow"];
      if (flows) {
        const list = Array.isArray(flows) ? flows : [flows];
        list.forEach((flow: BPMNFlow) => {
          // Parse handles from extension elements if available
          let extSourceHandle: string | undefined;
          let extTargetHandle: string | undefined;
          
          if (flow["bpmn:extensionElements"]) {
              const extEl = flow["bpmn:extensionElements"];
              // Handle potential array (though unlikely for extensionElements container)
              const extensions = Array.isArray(extEl) ? extEl[0] : extEl;
              
              if (extensions) {
                  // Check for connector with or without namespace, and handle array
                  const connectorRaw = extensions["workflowsa:connector"] || extensions["connector"];
                  
                  if (connectorRaw) {
                      const connector = Array.isArray(connectorRaw) ? connectorRaw[0] : connectorRaw;
                      extSourceHandle = connector["@_sourceHandle"];
                      extTargetHandle = connector["@_targetHandle"];
                  }
              }
          }

          // Determine handles with immediate fallback
          const sourceHandle = extSourceHandle || (flow["@_workflowsa:sourceHandle"] || flow["@_workflowsa_sourceHandle"] || flow["@_sourceHandle"]) as string || 'right';
          const targetHandle = extTargetHandle || (flow["@_workflowsa:targetHandle"] || flow["@_workflowsa_targetHandle"] || flow["@_targetHandle"]) as string || 'left';

          edges.push({
            id: String(flow["@_id"]),
            source: String(flow["@_sourceRef"]),
            target: String(flow["@_targetRef"]),
            label: flow["@_name"],
            sourceHandle: sourceHandle,
            targetHandle: targetHandle,
            type: 'smoothstep', 
            style: { strokeWidth: 2, stroke: '#334155' },
            markerEnd: {
                type: 'arrowclosed',
                width: 20,
                height: 20,
                color: '#334155',
            },
            data: { ...flow }
          });
        });
      }
  };

  parseContainer(process);
  

  // Helper to get bounds for handle calculation
  const getBoundsForHandle = (id: string, type: string) => {
    const shape = shapes.find((s: BPMNShape) => s["@_bpmnElement"] === id);
    if (shape && shape["dc:Bounds"]) {
      return {
        x: parseFloat(shape["dc:Bounds"]["@_x"]),
        y: parseFloat(shape["dc:Bounds"]["@_y"]),
        width: parseFloat(shape["dc:Bounds"]["@_width"]),
        height: parseFloat(shape["dc:Bounds"]["@_height"]),
      };
    }
    const size = getElementSize(type);
    return { x: 0, y: 0, ...size };
  };

  const getClosestHandle = (nodeId: string, point: { "@_x": string, "@_y": string }) => {
      const node = nodes.find(n => n.id === nodeId);
      if (!node) return null;

      const bounds = getBoundsForHandle(nodeId, node.data.originalType as string);
      
      const px = parseFloat(point["@_x"]);
      const py = parseFloat(point["@_y"]);

      const distLeft = Math.abs(px - bounds.x);
      const distRight = Math.abs(px - (bounds.x + bounds.width));
      const distTop = Math.abs(py - bounds.y);
      const distBottom = Math.abs(py - (bounds.y + bounds.height));

      const min = Math.min(distLeft, distRight, distTop, distBottom);

      if (min === distLeft) return 'left';
      if (min === distRight) return 'right';
      if (min === distTop) return 'top';
      if (min === distBottom) return 'bottom';
      return null;
  };

  // Assign handles based on DI waypoints
  const bpmndiEdges = plane && plane["bpmndi:BPMNEdge"] ? (Array.isArray(plane["bpmndi:BPMNEdge"]) ? plane["bpmndi:BPMNEdge"] : [plane["bpmndi:BPMNEdge"]]) : [];

  edges.forEach(edge => {
      const diEdge = bpmndiEdges.find((e: Record<string, unknown>) => e["@_bpmnElement"] === edge.id);
      if (diEdge && diEdge["di:waypoint"]) {
          const waypoints = Array.isArray(diEdge["di:waypoint"]) ? diEdge["di:waypoint"] : [diEdge["di:waypoint"]];
          if (waypoints.length >= 2) {
              const first = waypoints[0];
              const last = waypoints[waypoints.length - 1];
              
              // Only calculate if not already set from custom attributes
              if (!edge.sourceHandle) {
                  const sourceHandle = getClosestHandle(edge.source, first);
                  if (sourceHandle) {
                      edge.sourceHandle = sourceHandle;
                  } else {
                      console.warn(`[BPMN] Could not determine source handle for edge ${edge.id}, defaulting to 'right'`);
                      edge.sourceHandle = 'right';
                  }
              }
              if (!edge.targetHandle) {
                  const targetHandle = getClosestHandle(edge.target, last);
                  if (targetHandle) {
                      edge.targetHandle = targetHandle;
                  } else {
                      console.warn(`[BPMN] Could not determine target handle for edge ${edge.id}, defaulting to 'left'`);
                      edge.targetHandle = 'left';
                  }
              }
          } else {
               // Fallback if no waypoints available at all
               if (!edge.sourceHandle) edge.sourceHandle = 'right';
               if (!edge.targetHandle) edge.targetHandle = 'left';
          }
      } else {
           // Fallback if no DI edge found
           if (!edge.sourceHandle) edge.sourceHandle = 'right';
           if (!edge.targetHandle) edge.targetHandle = 'left';
      }
  });

  return { nodes, edges, processId, processName };
};

export const generateBpmnXml = (nodes: Node[], edges: Edge[], processId: string = "Process_1", processName: string = "Process"): string => {
  // Helpers for geometry
  const getNodeSize = (nodeId: string) => {
      const node = nodes.find(n => n.id === nodeId);
      if (!node) return { width: 100, height: 80 };
      return {
          width: Number(node.style?.width || node.data.width || 100),
          height: Number(node.style?.height || node.data.height || 80)
      };
  };

  const getAbsolutePosition = (nodeId: string): {x: number, y: number} => {
      const node = nodes.find(n => n.id === nodeId);
      if (!node) return { x: 0, y: 0 };
      
      let x = node.position.x;
      let y = node.position.y;
      
      if (node.parentId) {
          const parentPos = getAbsolutePosition(node.parentId);
          x += parentPos.x;
          y += parentPos.y;
      }
      return { x, y };
  };

  const getHandlePosition = (nodeId: string, handleId: string | null | undefined) => {
      const pos = getAbsolutePosition(nodeId);
      const size = getNodeSize(nodeId);
      
      if (!handleId) return { x: pos.x + size.width / 2, y: pos.y + size.height / 2 };

      switch (handleId) {
          case 'left': return { x: pos.x, y: pos.y + size.height / 2 };
          case 'right': return { x: pos.x + size.width, y: pos.y + size.height / 2 };
          case 'top': return { x: pos.x + size.width / 2, y: pos.y };
          case 'bottom': return { x: pos.x + size.width / 2, y: pos.y + size.height };
          default: return { x: pos.x + size.width / 2, y: pos.y + size.height / 2 };
      }
  };

  // Determine effective parents (hierarchy)
  const effectiveParentMap: Record<string, string> = {};
  const subProcessNodes = nodes.filter(n => n.type === 'subProcess' || n.data.originalType === 'bpmn:subProcess');

  nodes.forEach(node => {
      // If explicit parent exists, honor it
      if (node.parentId) {
          effectiveParentMap[node.id] = node.parentId;
          return;
      }
      
      // Filter out self
      const potentialParents = subProcessNodes.filter(sp => sp.id !== node.id);
      
      if (potentialParents.length === 0) {
          effectiveParentMap[node.id] = processId;
          return;
      }

      // Check containment using absolute positions
      const nodePos = getAbsolutePosition(node.id);
      const nodeSize = getNodeSize(node.id);
      const nodeCenter = {
          x: nodePos.x + nodeSize.width / 2,
          y: nodePos.y + nodeSize.height / 2
      };

      const containers = potentialParents.filter(sp => {
          const spPos = getAbsolutePosition(sp.id);
          const spSize = getNodeSize(sp.id);
          
          return (
              nodeCenter.x >= spPos.x &&
              nodeCenter.x <= spPos.x + spSize.width &&
              nodeCenter.y >= spPos.y &&
              nodeCenter.y <= spPos.y + spSize.height
          );
      });

      if (containers.length > 0) {
          // Sort by size (area) ascending to find the innermost container
          containers.sort((a, b) => {
              const sizeA = getNodeSize(a.id);
              const sizeB = getNodeSize(b.id);
              return (sizeA.width * sizeA.height) - (sizeB.width * sizeB.height);
          });
          effectiveParentMap[node.id] = containers[0].id;
      } else {
          effectiveParentMap[node.id] = processId;
      }
  });

  // Map to store XML objects for each container (process or subprocess)
  const containers: Record<string, Record<string, unknown[]>> = {};
  
  // Initialize root container
  containers[processId] = {};
  
  // Initialize containers for all subprocesses
  nodes.filter(n => n.type === 'subProcess' || n.data.originalType === 'bpmn:subProcess').forEach(n => {
      containers[n.id] = {};
  });

  // Map to keep track of created XML objects to inject children later
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const nodeXmlObjects: Record<string, Record<string, any>> = {};
  
  // Group nodes by parent
  nodes.forEach(node => {
    const parentId = effectiveParentMap[node.id] || processId;
    const container = containers[parentId] || containers[processId];
    
    const tag = node.data.originalType as string || "bpmn:task";
    if (!container[tag]) container[tag] = [];
    
    const nodeEl: Record<string, unknown> = {
      "@_id": node.id,
      "@_name": node.data.label || "",
    };

    // Add other properties
    Object.keys(node.data).forEach(key => {
        if (key !== 'originalType' && key !== 'label' && key !== 'width' && key !== 'height') {
             const value = node.data[key];
             if (typeof value === 'object' && value !== null) {
                 // It's a child element (like extensionElements)
                 nodeEl[key] = value;
             } else if (key.startsWith('@_')) {
                 nodeEl[key] = value;
             } else {
                 nodeEl[`@_${key}`] = value;
             }
        }
    });
    
    // Add incoming/outgoing refs based on edges
    const incoming = edges.filter(e => e.target === node.id).map(e => e.id);
    const outgoing = edges.filter(e => e.source === node.id).map(e => e.id);
    
    if (incoming.length > 0) nodeEl["bpmn:incoming"] = incoming;
    if (outgoing.length > 0) nodeEl["bpmn:outgoing"] = outgoing;

    container[tag].push(nodeEl);
    nodeXmlObjects[node.id] = nodeEl;
  });

  // Add Sequence Flows
  edges.forEach(edge => {
      // Determine container based on source node's parent
      const parentId = effectiveParentMap[edge.source] || processId;
      const container = containers[parentId] || containers[processId];

      if (!container["bpmn:sequenceFlow"]) container["bpmn:sequenceFlow"] = [];

      const flow: Record<string, unknown> = {
        "@_id": edge.id,
        "@_name": edge.label || "",
        "@_sourceRef": edge.source,
        "@_targetRef": edge.target,
      };
      
      if (edge.sourceHandle || edge.targetHandle) {
          flow["bpmn:extensionElements"] = {
              "workflowsa:connector": {
                  "@_sourceHandle": edge.sourceHandle,
                  "@_targetHandle": edge.targetHandle
              }
          };
      }
      
      if (edge.sourceHandle) {
          flow["@_workflowsa:sourceHandle"] = edge.sourceHandle;
          flow["@_workflowsa_sourceHandle"] = edge.sourceHandle;
      }
      if (edge.targetHandle) {
          flow["@_workflowsa:targetHandle"] = edge.targetHandle;
          flow["@_workflowsa_targetHandle"] = edge.targetHandle;
      }

      if (edge.data) {
        const data = edge.data;
        Object.keys(data).forEach(key => {
            // Skip keys we explicitly handle or that are internal
            if ([
                '@_id', '@_name', '@_sourceRef', '@_targetRef', 'label',
                'bpmn:extensionElements',
                '@_workflowsa:sourceHandle', '@_workflowsa_sourceHandle', '@_sourceHandle',
                '@_workflowsa:targetHandle', '@_workflowsa_targetHandle', '@_targetHandle'
            ].includes(key)) return;
            
            const value = data[key];
            if (typeof value === 'object' && value !== null) {
                 flow[key] = value;
            } else if (key.startsWith('@_')) {
                 flow[key] = value;
            } else {
                 flow[`@_${key}`] = value;
            }
        });
      }
      container["bpmn:sequenceFlow"].push(flow);
    });

    // Nest SubProcesses
    Object.keys(containers).forEach(containerId => {
      if (containerId === processId) return; // Skip root
      
      const content = containers[containerId];
      const subProcessXml = nodeXmlObjects[containerId];
      
      if (subProcessXml) {
          Object.assign(subProcessXml, content);
      }
    });

  // DI Generation
  const bpmndiShapes = nodes.map(node => {
      const pos = getAbsolutePosition(node.id);
      const size = getNodeSize(node.id);
      return {
          "@_id": `_BPMNShape_${node.id}`,
          "@_bpmnElement": node.id,
          "@_isExpanded": "true",
          "dc:Bounds": {
              "@_x": pos.x,
              "@_y": pos.y,
              "@_width": size.width,
              "@_height": size.height
          }
      };
  });

  const bpmndiEdges = edges.map(edge => {
      const sourcePos = getHandlePosition(edge.source, edge.sourceHandle);
      const targetPos = getHandlePosition(edge.target, edge.targetHandle);
      
      // Basic orthogonal routing: Start -> Mid -> End
      // Ideally we'd have real waypoints from React Flow, but for now anchor to handles
      return {
        "@_id": `_BPMNEdge_${edge.id}`,
        "@_bpmnElement": edge.id,
        "di:waypoint": [
            { "@_x": sourcePos.x, "@_y": sourcePos.y }, 
            { "@_x": targetPos.x, "@_y": targetPos.y }
        ]
      };
  });

  const jsonObj = {
    "bpmn:definitions": {
      "@_xmlns:bpmn": BPMN_NS,
      "@_xmlns:bpmndi": BPMNDI_NS,
      "@_xmlns:dc": DC_NS,
      "@_xmlns:di": DI_NS,
      "@_xmlns:workflowsa": WORKFLOWSA_NS,
      "@_xmlns:xsi": "http://www.w3.org/2001/XMLSchema-instance",
      "@_id": "Definitions_1",
      "@_targetNamespace": "http://bpmn.org/schema/bpmn",
      "bpmn:process": {
        "@_id": processId,
        "@_name": processName,
        "@_isExecutable": "true",
        ...containers[processId]
      },
      "bpmndi:BPMNDiagram": {
        "@_id": "BPMNDiagram_1",
        "bpmndi:BPMNPlane": {
            "@_id": "BPMNPlane_1",
            "@_bpmnElement": processId,
            "bpmndi:BPMNShape": bpmndiShapes,
            "bpmndi:BPMNEdge": bpmndiEdges
        }
      }
    }
  };

  const builder = new XMLBuilder(DEFAULT_OPTIONS);
  return `<?xml version="1.0" encoding="UTF-8"?>\n` + builder.build(jsonObj);
};
