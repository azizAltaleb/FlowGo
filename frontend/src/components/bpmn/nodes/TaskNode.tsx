import { Handle, Position, type NodeProps, NodeResizer } from '@xyflow/react';
import { User, Settings, FileCode, MessageSquare, ClipboardList, HelpingHand, ExternalLink, Layers } from "lucide-react";
import { ContextPad } from "../ContextPad";

const icons: Record<string, React.ElementType> = {
  userTask: User,
  serviceTask: Settings,
  scriptTask: FileCode,
  receiveTask: MessageSquare,
  manualTask: HelpingHand,
  businessRuleTask: ClipboardList,
  callActivity: ExternalLink,
  subProcess: Layers,
};

export function TaskNode({ id, data, selected }: NodeProps) {
  const type = data.originalType as string;
  const label = data.label as string;
  const status = (data.executionStatus as string) || 'pending';
  const Icon = icons[type.replace('bpmn:', '')] || Settings;
  const isSubProcess = type === 'bpmn:subProcess';

  // SVG Styling
  let strokeColor = "#334155"; // slate-700
  let strokeWidth = 2;
  let fillColor = "#ffffff";

  if (status === 'active') {
    strokeColor = "#3b82f6"; // blue-500
    fillColor = "#eff6ff"; // blue-50
    strokeWidth = 3;
  } else if (status === 'completed') {
    strokeColor = "#16a34a"; // green-600
    fillColor = "#f0fdf4"; // green-50
  }

  // SubProcess specific styling
  if (isSubProcess) {
    return (
        <div className={`w-full h-full relative`}>
          <NodeResizer minWidth={200} minHeight={150} isVisible={selected} />
          <Handle type="target" position={Position.Left} style={{ background: '#94a3b8' }} />
          
          <svg className="w-full h-full overflow-visible">
            <rect
              x="0"
              y="0"
              width="100%"
              height="100%"
              rx="10"
              ry="10"
              fill={fillColor}
              stroke={strokeColor}
              strokeWidth={strokeWidth}
              className="drop-shadow-sm"
            />
            {/* SubProcess Marker (Collapsed look or just icon) */}
            <foreignObject x="10" y="10" width="100%" height="24">
               <div className="flex items-center gap-2 pl-2">
                  <Icon className="w-4 h-4 text-slate-500" />
                  <span className="text-xs font-semibold text-slate-700 truncate pr-4">
                    {label || "Sub Process"}
                  </span>
               </div>
            </foreignObject>
          </svg>
          
          <div className="absolute inset-0 top-8 px-2 pointer-events-none">
             {/* Content area for visual nesting (React Flow handles nodes separately) */}
          </div>

          <Handle type="source" position={Position.Right} style={{ background: '#94a3b8' }} />
        </div>
    );
  }

  return (
    <div className={`w-full h-full relative flex items-center justify-center`}>
      <ContextPad id={id} isVisible={!!selected} />
      
      {/* Handles - Invisible bars along edges. 
          With ConnectionMode.Loose, 'source' handles can also accept connections. 
          This keeps the center free for moving the node. 
      */}
      <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(-50%)', opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(-50%)', opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: '100%', width: '10px', transform: 'translateX(50%)', opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: '100%', height: '10px', transform: 'translateY(50%)', opacity: 0, zIndex: 10 }} />
      
      <svg className="w-full h-full overflow-visible">
        <rect
          x="0"
          y="0"
          width="100%"
          height="100%"
          rx="10"
          ry="10"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          className={`transition-colors ${selected ? 'stroke-primary stroke-[3px]' : ''} drop-shadow-sm`}
        />
        {/* Icon */}
        <foreignObject x="5" y="5" width="20" height="20">
           <Icon className={`w-4 h-4 ${status === 'active' ? 'text-blue-600' : (status === 'completed' ? 'text-green-600' : 'text-slate-500')}`} />
        </foreignObject>
        
        {/* Label */}
        <foreignObject x="0" y="30" width="100%" height="40">
          <div className="w-full h-full flex items-center justify-center px-1">
             <span className="text-[10px] text-center font-medium leading-tight line-clamp-2 select-none pointer-events-none text-slate-800">
                {label || type.replace('bpmn:', '')}
             </span>
          </div>
        </foreignObject>
      </svg>
    </div>
  );
}
