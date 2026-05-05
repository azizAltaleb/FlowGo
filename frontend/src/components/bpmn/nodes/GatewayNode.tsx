import { Handle, Position, type NodeProps } from '@xyflow/react';
import { X, Plus, Circle } from "lucide-react";
import { ContextPad } from "../ContextPad";

const icons: Record<string, React.ElementType> = {
  "bpmn:exclusiveGateway": X,
  "bpmn:parallelGateway": Plus,
  "bpmn:inclusiveGateway": Circle,
};

export function GatewayNode({ id, data, selected }: NodeProps) {
  const type = data.originalType as string;
  const status = (data.executionStatus as string) || 'pending';
  const Icon = icons[type] || X;

  let strokeColor = "#d97706"; // amber-600
  let strokeWidth = 2;
  let fillColor = "#fffbeb"; // amber-50

  if (status === 'active') {
    strokeColor = "#3b82f6"; // blue-500
    fillColor = "#eff6ff"; // blue-50
    strokeWidth = 3;
  } else if (status === 'completed') {
    strokeColor = "#16a34a"; // green-600
    fillColor = "#f0fdf4"; // green-50
  }

  return (
    <div className={`w-full h-full relative flex items-center justify-center`}>
      <ContextPad id={id} isVisible={!!selected} />
      {/* Handles positioned around the bounding box - Invisible bars. Source only. */}
      <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(-50%)', opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(-50%)', opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(50%)', opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(50%)', opacity: 0, zIndex: 10 }} />
      
      <svg className="w-full h-full overflow-visible" viewBox="0 0 50 50">
        {/* Diamond shape using polygon */}
        {/* 50x50 box: Top(25,0), Right(50,25), Bottom(25,50), Left(0,25) */}
        <polygon
          points="25,0 50,25 25,50 0,25"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          className={`transition-colors ${selected ? 'stroke-primary stroke-[3px]' : ''} drop-shadow-sm`}
        />
        
        {/* Icon */}
        <foreignObject x="15" y="15" width="20" height="20">
           <div className="w-full h-full flex items-center justify-center">
              <Icon className={`w-5 h-5 ${status === 'active' ? 'text-blue-800' : (status === 'completed' ? 'text-green-800' : 'text-amber-800')}`} />
           </div>
        </foreignObject>
      </svg>
    </div>
  );
}
