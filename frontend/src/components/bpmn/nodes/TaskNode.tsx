import { Handle, Position, type NodeProps, NodeResizer } from "@xyflow/react";
import { ContextPad } from "../ContextPad";
import { BpmnTaskMarker } from "../BpmnIcons";

export function TaskNode({ id, data, selected }: NodeProps) {
  const type = (data.originalType as string) || "";
  const label = (data.label as string) || "";
  const status = (data.executionStatus as string) || "pending";
  const taskKind = type.replace("bpmn:", "") || "serviceTask";
  const isSubProcess = type === "bpmn:subProcess" || type === "bpmn:transaction";
  const isCallActivity = type === "bpmn:callActivity";
  const isTransaction = type === "bpmn:transaction" || data.transaction === true;

  let strokeColor = "#1e293b";
  let strokeWidth = isCallActivity || isTransaction ? 3 : 2;
  let fillColor = "#ffffff";

  if (status === "active") {
    strokeColor = "#3b82f6";
    fillColor = "#eff6ff";
    strokeWidth = isCallActivity || isTransaction ? 3.5 : 3;
  } else if (status === "completed") {
    strokeColor = "#16a34a";
    fillColor = "#f0fdf4";
  }

  const markerColor =
    status === "active" ? "#1d4ed8" : status === "completed" ? "#15803d" : "#475569";

  if (isSubProcess) {
    return (
      <div className="w-full h-full relative">
        <NodeResizer minWidth={200} minHeight={150} isVisible={selected} />
        <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(-50%)", opacity: 0, zIndex: 10 }} />
        <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(-50%)", opacity: 0, zIndex: 10 }} />
        <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(50%)", opacity: 0, zIndex: 10 }} />
        <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(50%)", opacity: 0, zIndex: 10 }} />

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
          <foreignObject x="8" y="8" width="100%" height="24">
            <div className="flex items-center gap-2 pl-1">
              <BpmnTaskMarker kind={isTransaction ? "transaction" : "subProcess"} size={16} color={markerColor} />
              <span className="text-xs font-semibold text-slate-700 truncate pr-4">
                {label || (isTransaction ? "Transaction" : "Sub Process")}
              </span>
            </div>
          </foreignObject>
        </svg>
      </div>
    );
  }

  return (
    <div className="w-full h-full relative flex items-center justify-center">
      <ContextPad id={id} isVisible={!!selected} />
      <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(-50%)", opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(-50%)", opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(50%)", opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(50%)", opacity: 0, zIndex: 10 }} />

      <svg className="w-full h-full overflow-visible">
        <rect
          x="0"
          y="0"
          width="100%"
          height="100%"
          rx="10"
          ry="10"
          fill={fillColor}
          stroke={selected ? "var(--color-primary, #2563eb)" : strokeColor}
          strokeWidth={selected ? strokeWidth + 1 : strokeWidth}
          className="transition-colors drop-shadow-sm"
        />
        <foreignObject x="5" y="5" width="20" height="20">
          <BpmnTaskMarker kind={taskKind} size={16} color={markerColor} />
        </foreignObject>
        <foreignObject x="0" y="30" width="100%" height="40">
          <div className="w-full h-full flex items-center justify-center px-1">
            <span className="text-[10px] text-center font-medium leading-tight line-clamp-2 select-none pointer-events-none text-slate-800">
              {label || taskKind}
            </span>
          </div>
        </foreignObject>
      </svg>
    </div>
  );
}
