import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ContextPad } from "../ContextPad";
import { GatewayMarkerGlyph } from "../BpmnIcons";

function gatewayKind(type: string): "exclusive" | "parallel" | "inclusive" | "eventBased" {
  if (type.includes("parallel")) return "parallel";
  if (type.includes("inclusive")) return "inclusive";
  if (type.includes("eventBased")) return "eventBased";
  return "exclusive";
}

export function GatewayNode({ id, data, selected }: NodeProps) {
  const type = (data.originalType as string) || "";
  const status = (data.executionStatus as string) || "pending";
  const kind = gatewayKind(type);

  let strokeColor = "#1e293b";
  let strokeWidth = 2;
  let fillColor = "#ffffff";

  if (status === "active") {
    strokeColor = "#3b82f6";
    fillColor = "#eff6ff";
    strokeWidth = 3;
  } else if (status === "completed") {
    strokeColor = "#16a34a";
    fillColor = "#f0fdf4";
  }

  const markerColor =
    status === "active" ? "#1d4ed8" : status === "completed" ? "#15803d" : "#1e293b";

  return (
    <div className="w-full h-full relative flex items-center justify-center">
      <ContextPad id={id} isVisible={!!selected} />
      <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(-50%)", opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(-50%)", opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(50%)", opacity: 0, zIndex: 10 }} />
      <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(50%)", opacity: 0, zIndex: 10 }} />

      <svg className="w-full h-full overflow-visible" viewBox="0 0 50 50">
        <polygon
          points="25,1 49,25 25,49 1,25"
          fill={fillColor}
          stroke={selected ? "var(--color-primary, #2563eb)" : strokeColor}
          strokeWidth={selected ? strokeWidth + 1 : strokeWidth}
          className="transition-colors drop-shadow-sm"
        />
        <g transform="translate(25 25) scale(0.85) translate(-12 -12)">
          <GatewayMarkerGlyph kind={kind} color={markerColor} />
        </g>
      </svg>
    </div>
  );
}
