import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ContextPad } from "../ContextPad";
import { EventMarkerGlyph } from "../BpmnIcons";
import { eventThrowStyle } from "../bpmn-icon-types";
import { inferEventDefinitionKind } from "@/lib/bpmn-event-definitions";

export function EventNode({ id, data, selected }: NodeProps) {
  const type = (data.originalType as string) || "";
  const label = (data.label as string) || "";
  const status = (data.executionStatus as string) || "pending";
  const eventKind =
    (data.eventDefinitionType as string) ||
    inferEventDefinitionKind(data as Record<string, unknown>);
  const markerStyle = eventThrowStyle(type);

  const isStart = type === "bpmn:startEvent";
  const isEnd = type === "bpmn:endEvent";
  const isIntermediate = type.includes("intermediate") || type === "bpmn:boundaryEvent";

  // BPMN-standard default: dark slate stroke. Status accents only when executing.
  let strokeColor = "#1e293b";
  let strokeWidth = isEnd ? 3.5 : 2;
  let fillColor = "#ffffff";

  if (status === "active") {
    strokeColor = "#3b82f6";
    fillColor = "#eff6ff";
    strokeWidth = isEnd ? 4 : 3;
  } else if (status === "completed") {
    strokeColor = "#16a34a";
    fillColor = "#f0fdf4";
  }

  const markerColor =
    status === "active" ? "#1d4ed8" : status === "completed" ? "#15803d" : "#1e293b";

  return (
    <div className="w-full h-full relative flex items-center justify-center">
      <ContextPad id={id} isVisible={!!selected} />
      {!isStart && !isEnd && (
        <>
          <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(-50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(-50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(50%)", opacity: 0, zIndex: 10 }} />
        </>
      )}
      {isStart && (
        <>
          <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(-50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(-50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(50%)", opacity: 0, zIndex: 10 }} />
        </>
      )}
      {isEnd && (
        <>
          <Handle type="source" position={Position.Left} id="left" style={{ left: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(-50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Top} id="top" style={{ top: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(-50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Right} id="right" style={{ right: 0, top: 0, bottom: 0, height: "100%", width: "10px", transform: "translateX(50%)", opacity: 0, zIndex: 10 }} />
          <Handle type="source" position={Position.Bottom} id="bottom" style={{ bottom: 0, left: 0, right: 0, width: "100%", height: "10px", transform: "translateY(50%)", opacity: 0, zIndex: 10 }} />
        </>
      )}

      <svg className="w-full h-full overflow-visible" viewBox="0 0 36 36">
        <circle
          cx="18"
          cy="18"
          r="16"
          fill={fillColor}
          stroke={selected ? "var(--color-primary, #2563eb)" : strokeColor}
          strokeWidth={selected ? strokeWidth + 1 : strokeWidth}
          className="drop-shadow-sm transition-colors"
        />
        {isIntermediate && (
          <circle cx="18" cy="18" r="12.5" fill="none" stroke={strokeColor} strokeWidth="1.4" />
        )}
        <g transform="translate(18 18) scale(0.55) translate(-12 -12)">
          <EventMarkerGlyph
            kind={eventKind === "none" ? "none" : eventKind}
            style={eventKind === "none" ? "catch" : markerStyle}
            color={markerColor}
          />
        </g>
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
