import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Play, Ban, Timer, Zap, Circle as CircleIcon } from "lucide-react";
import { ContextPad } from "../ContextPad";

const icons: Record<string, React.ElementType> = {
  "bpmn:startEvent": Play,
  "bpmn:endEvent": Ban,
  "bpmn:intermediateCatchEvent": Timer,
  "bpmn:intermediateThrowEvent": Zap,
  "bpmn:boundaryEvent": Zap,
};

export function EventNode({ id, data, selected }: NodeProps) {
  const type = data.originalType as string;
  const label = data.label as string;
  const status = (data.executionStatus as string) || 'pending';
  // Use icons mainly for semantic meaning if needed, but standard BPMN often just uses the shape or specific markers.
  // For now, we keep the icons inside to differentiate types visually beyond just the border.
  const Icon = icons[type] || CircleIcon;

  const isStart = type === "bpmn:startEvent";
  const isEnd = type === "bpmn:endEvent";
  const isIntermediate = type.includes("intermediate") || type === "bpmn:boundaryEvent";

  // SVG Styles
  let strokeColor = "#1e293b"; // slate-800
  let strokeWidth = 2;
  let fillColor = "#ffffff";

  // Status overrides
  if (status === 'active') {
    strokeColor = "#3b82f6"; // blue-500
    fillColor = "#eff6ff"; // blue-50
    strokeWidth = 4;
  } else if (status === 'completed') {
    strokeColor = "#16a34a"; // green-600
    fillColor = "#f0fdf4"; // green-50
  } else {
    // Standard BPMN Type Styling (if not active/completed)
    if (isStart) {
        strokeColor = "#16a34a"; // green-600 often used for start
        strokeWidth = 2;
    } else if (isEnd) {
        strokeColor = "#dc2626"; // red-600 often used for end
        strokeWidth = 4;
    } else if (isIntermediate) {
        strokeColor = "#ca8a04"; // yellow-600
        // Double border is hard in single path, usually strictly nested circles. 
        // We'll stick to a specific look or double circle simulation if possible, 
        // but for now distinct color/width is a good start.
        strokeWidth = 2;
    }
  }

  return (
    <div className={`w-full h-full relative flex items-center justify-center`}>
       <ContextPad id={id} isVisible={!!selected} />
       {/* Handles - All sides - Invisible bars. Source only (Loose mode enables targeting) */}
       {!isStart && !isEnd && (
         <>
            <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(-50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(-50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(50%)', opacity: 0, zIndex: 10 }} />
         </>
       )}
       {/* Special cases for Start/End if needed, but for now apply consistent generic handles unless strict BPMN rules enforced */}
       {isStart && (
          <>
            <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(50%)', opacity: 0, zIndex: 10 }} />
            {/* Start events can typically connect from any side in loose modeling */}
            <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(-50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(-50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(50%)', opacity: 0, zIndex: 10 }} />
          </>
       )}
       {isEnd && (
          <>
             {/* End events technically only receive, but with Loose mode we might want to start sequence flows from them if user makes mistake or for other extensions. 
                 But strictly they are targets. With Loose mode, Source handles act as Targets too. So we provide Source handles. 
             */}
            <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(-50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(-50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(50%)', opacity: 0, zIndex: 10 }} />
            <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(50%)', opacity: 0, zIndex: 10 }} />
          </>
       )}
      
      <svg className="w-full h-full overflow-visible" viewBox="0 0 36 36">
        {/* Main Circle */}
        <circle
          cx="18"
          cy="18"
          r="17"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          className={`transition-colors ${selected ? 'stroke-primary stroke-[3px]' : ''} drop-shadow-sm`}
        />
        
        {/* Inner Circle for Intermediate events */}
        {isIntermediate && (
            <circle
            cx="18"
            cy="18"
            r="14"
            fill="none"
            stroke={strokeColor}
            strokeWidth="1"
            />
        )}

        {/* Icon / Marker */}
        <foreignObject x="8" y="8" width="20" height="20">
             <div className="w-full h-full flex items-center justify-center">
                <Icon className={`w-4 h-4 ${status === 'active' ? 'text-blue-600' : (status === 'completed' ? 'text-green-600' : 'text-slate-700')}`} />
             </div>
        </foreignObject>
        
        {/* Label */}
        <foreignObject x="-22" y="38" width="80" height="20">
            <div className="w-full flex justify-center">
                <span className="text-[10px] font-medium text-slate-600 leading-tight text-center bg-white/80 rounded px-1 text-nowrap">
                    {label}
                </span>
            </div>
        </foreignObject>
      </svg>
    </div>
  );
}
