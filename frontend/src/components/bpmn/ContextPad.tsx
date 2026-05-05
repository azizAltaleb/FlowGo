import { NodeToolbar, Position, type Node } from '@xyflow/react';
import { useReactFlow } from '@xyflow/react';
import { Square, Trash2, Circle, GitMerge } from 'lucide-react';
import { Button } from "@/components/ui/button";
import { getElementSize } from '@/lib/bpmn-parser';

interface ContextPadProps {
  id: string;
  isVisible: boolean;
}

export function ContextPad({ id, isVisible }: ContextPadProps) {
  const { getNodes, setNodes, setEdges, addNodes, addEdges } = useReactFlow();

  const handleAppendNode = (type: string, originalType: string, label: string) => {
    const parentNode = getNodes().find((n) => n.id === id);
    if (!parentNode) return;

    // Calculate new position (to the right by default)
    // We need to account for parent node size to place it nicely
    const parentWidth = parentNode.measured?.width || parentNode.style?.width || 100;
    const offset = 100;
    
    // Position: Center Y, Right X + offset
    const newPos = {
        x: parentNode.position.x + (Number(parentWidth)) + offset,
        y: parentNode.position.y
    };

    const size = getElementSize(originalType);
    const newNodeId = `${type}_${Date.now()}`;

    const newNode: Node = {
      id: newNodeId,
      type,
      position: newPos,
      data: { label, originalType },
      style: { width: size.width, height: size.height },
    };

    const newEdge = {
        id: `flow_${Date.now()}`,
        source: id,
        target: newNodeId,
        type: 'smoothstep', // Standard orthogonal edges
        data: {}
    };

    addNodes(newNode);
    addEdges(newEdge);
  };

  const handleDelete = () => {
    setNodes((nodes) => nodes.filter((n) => n.id !== id));
    setEdges((edges) => edges.filter((e) => e.source !== id && e.target !== id));
  };

  return (
    <NodeToolbar isVisible={isVisible} position={Position.Right} align="start" offset={20} className="flex flex-col gap-2 p-2 bg-white rounded-lg shadow-xl border border-gray-200 w-12 items-center z-50">
        {/* Append Actions */}
        <Button variant="ghost" size="icon" className="h-8 w-8 hover:bg-slate-100 text-slate-600" title="Append Task" onClick={() => handleAppendNode('userTask', 'bpmn:userTask', 'User Task')}>
            <Square className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" className="h-8 w-8 hover:bg-slate-100 text-slate-600" title="Append Gateway" onClick={() => handleAppendNode('exclusiveGateway', 'bpmn:exclusiveGateway', '')}>
            <GitMerge className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" className="h-8 w-8 hover:bg-slate-100 text-slate-600" title="Append End Event" onClick={() => handleAppendNode('endEvent', 'bpmn:endEvent', 'End')}>
            <Circle className="h-4 w-4 stroke-[4px]" />
        </Button>
        
        <div className="w-full h-px bg-gray-200 my-1"></div>

        {/* Edit Actions */}
        <Button variant="ghost" size="icon" className="h-8 w-8 hover:bg-red-50 text-red-500" title="Delete" onClick={handleDelete}>
            <Trash2 className="h-4 w-4" />
        </Button>
    </NodeToolbar>
  );
}
