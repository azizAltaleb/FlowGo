import type { ReactNode, SVGProps } from "react";
import type { MarkerStyle } from "@/components/bpmn/bpmn-icon-types";

type IconProps = SVGProps<SVGSVGElement> & {
  size?: number;
  color?: string;
};

const DEFAULT = "#1e293b";

function Frame({ size = 20, className, children, ...rest }: IconProps & { children: ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden
      {...rest}
    >
      {children}
    </svg>
  );
}

/** Inner event marker glyphs in a 24×24 space (paths/groups only). */
export function EventMarkerGlyph({
  kind,
  style = "catch",
  color = DEFAULT,
}: {
  kind: string;
  style?: MarkerStyle;
  color?: string;
}) {
  const filled = style === "throw" || kind === "terminate";
  const fill = filled ? color : "none";

  switch (kind) {
    case "message":
      return (
        <g>
          <rect x="3" y="6" width="18" height="12" rx="1.5" stroke={color} strokeWidth={1.6} fill={fill} />
          <path d="M3.5 7.5 L12 13.5 L20.5 7.5" stroke={filled ? "#fff" : color} strokeWidth={1.6} fill="none" strokeLinejoin="round" />
        </g>
      );
    case "timer":
      return (
        <g>
          <circle cx="12" cy="13" r="8" stroke={color} strokeWidth={1.6} fill="none" />
          <path d="M12 13 V8.5 M12 13 L15.5 15" stroke={color} strokeWidth={1.6} strokeLinecap="round" />
          <path d="M9 3.5 H15 M12 3.5 V5.5" stroke={color} strokeWidth={1.6} strokeLinecap="round" />
        </g>
      );
    case "signal":
      return (
        <path d="M12 4 L20 19 H4 Z" stroke={color} strokeWidth={1.6} strokeLinejoin="round" fill={fill} />
      );
    case "error":
      return (
        <path
          d="M8 4 L14 11 H11 L16 20 L10 13 H13 Z"
          stroke={color}
          strokeWidth={filled ? 0 : 1.4}
          strokeLinejoin="round"
          fill={filled ? color : "none"}
        />
      );
    case "escalation":
      return (
        <path d="M12 3.5 L19.5 20 H4.5 Z" stroke={color} strokeWidth={1.6} strokeLinejoin="round" fill={fill} />
      );
    case "cancel":
      return <path d="M6 6 L18 18 M18 6 L6 18" stroke={color} strokeWidth={2.2} strokeLinecap="round" />;
    case "compensate":
      return (
        <g>
          <path d="M12 5 L4 12 L12 19 Z" stroke={color} strokeWidth={1.4} strokeLinejoin="round" fill={fill} />
          <path d="M20 5 L12 12 L20 19 Z" stroke={color} strokeWidth={1.4} strokeLinejoin="round" fill={fill} />
        </g>
      );
    case "conditional":
      return (
        <g>
          <rect x="5" y="3" width="14" height="18" rx="1" stroke={color} strokeWidth={1.6} fill="none" />
          <path d="M8 8 H16 M8 12 H16 M8 16 H13" stroke={color} strokeWidth={1.4} strokeLinecap="round" />
        </g>
      );
    case "link":
      return style === "throw" ? (
        <path d="M5 12 H15 M11 7 L16 12 L11 17" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" fill="none" />
      ) : (
        <path d="M19 12 H9 M13 7 L8 12 L13 17" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" fill="none" />
      );
    case "terminate":
      return <circle cx="12" cy="12" r="7" fill={color} />;
    case "none":
    default:
      return null;
  }
}

type EventShape = "start" | "end" | "intermediate" | "boundary";

/** Full palette/legend icon: BPMN event shape + marker. */
export function BpmnEventIcon({
  shape,
  kind = "none",
  style = "catch",
  size = 20,
  color = DEFAULT,
  className,
}: {
  shape: EventShape;
  kind?: string;
  style?: MarkerStyle;
  size?: number;
  color?: string;
  className?: string;
}) {
  const outerR = 10;
  const outerSw = shape === "end" ? 2.8 : 1.5;
  const showInner = shape === "intermediate" || shape === "boundary";
  const markerStyle: MarkerStyle = shape === "end" || style === "throw" ? "throw" : "catch";

  return (
    <Frame size={size} className={className}>
      <circle cx="12" cy="12" r={outerR} stroke={color} strokeWidth={outerSw} fill="#fff" />
      {showInner && <circle cx="12" cy="12" r={7.6} stroke={color} strokeWidth={1.2} fill="none" />}
      <g transform="translate(12 12) scale(0.4) translate(-12 -12)">
        <EventMarkerGlyph kind={kind} style={kind === "none" ? "catch" : markerStyle} color={color} />
      </g>
    </Frame>
  );
}

function TaskMarkerGlyph({ kind, color = DEFAULT }: { kind: string; color?: string }) {
  switch (kind) {
    case "userTask":
      return (
        <g>
          <circle cx="12" cy="8" r="3.5" stroke={color} strokeWidth={1.5} fill="none" />
          <path d="M5 20 C5 15.5 8 14 12 14 C16 14 19 15.5 19 20" stroke={color} strokeWidth={1.5} fill="none" strokeLinecap="round" />
        </g>
      );
    case "serviceTask":
      return (
        <g>
          <circle cx="12" cy="12" r="3" stroke={color} strokeWidth={1.5} fill="none" />
          <path
            d="M12 3.5 V6.5 M12 17.5 V20.5 M3.5 12 H6.5 M17.5 12 H20.5 M6.2 6.2 L8.3 8.3 M15.7 15.7 L17.8 17.8 M17.8 6.2 L15.7 8.3 M8.3 15.7 L6.2 17.8"
            stroke={color}
            strokeWidth={1.5}
            strokeLinecap="round"
          />
        </g>
      );
    case "scriptTask":
      return (
        <g>
          <path d="M7 4 H15 L18 7 V20 H7 Z" stroke={color} strokeWidth={1.5} fill="none" strokeLinejoin="round" />
          <path d="M9 10 C11 9 11 12 13 11 M9 14 C11 13 11 16 13 15" stroke={color} strokeWidth={1.4} strokeLinecap="round" fill="none" />
        </g>
      );
    case "businessRuleTask":
      return (
        <g>
          <rect x="4" y="5" width="16" height="14" rx="1" stroke={color} strokeWidth={1.5} fill="none" />
          <path d="M4 10 H20 M10 5 V19" stroke={color} strokeWidth={1.4} />
        </g>
      );
    case "sendTask":
      return (
        <g>
          <rect x="3" y="6" width="18" height="12" rx="1" fill={color} stroke={color} strokeWidth={1.2} />
          <path d="M3.5 7.5 L12 13 L20.5 7.5" stroke="#fff" strokeWidth={1.4} fill="none" />
        </g>
      );
    case "receiveTask":
      return (
        <g>
          <rect x="3" y="6" width="18" height="12" rx="1" stroke={color} strokeWidth={1.5} fill="none" />
          <path d="M3.5 7.5 L12 13 L20.5 7.5" stroke={color} strokeWidth={1.5} fill="none" />
        </g>
      );
    case "manualTask":
      return (
        <path
          d="M8 12 V7.5 C8 6.1 9.1 5 10.5 5 S13 6.1 13 7.5 V13 M13 13 V9.5 C13 8.7 13.7 8 14.5 8 S16 8.7 16 9.5 V14 M16 14 V11.5 C16 10.7 16.7 10 17.5 10 S19 10.7 19 11.5 V16 C19 18.8 16.8 20.5 14 20.5 H11 C8.2 20.5 6 18.5 6 16 V12"
          stroke={color}
          strokeWidth={1.5}
          fill="none"
          strokeLinecap="round"
        />
      );
    case "callActivity":
      return <rect x="4" y="6" width="16" height="12" rx="1.5" stroke={color} strokeWidth={2.4} fill="none" />;
    case "subProcess":
    case "transaction":
      return (
        <g>
          <rect x="5" y="7" width="14" height="10" rx="1" stroke={color} strokeWidth={1.5} fill="none" />
          <path d="M9 12 H15 M12 9 V15" stroke={color} strokeWidth={1.6} strokeLinecap="round" />
        </g>
      );
    default:
      return <rect x="5" y="7" width="14" height="10" rx="2" stroke={color} strokeWidth={1.5} fill="none" />;
  }
}

/** Task type markers for the canvas (top-left). */
export function BpmnTaskMarker({
  kind,
  size = 14,
  color = DEFAULT,
  className,
}: {
  kind: string;
  size?: number;
  color?: string;
  className?: string;
}) {
  return (
    <Frame size={size} className={className}>
      <TaskMarkerGlyph kind={kind} color={color} />
    </Frame>
  );
}

/** Full task legend icon for the palette. */
export function BpmnTaskIcon({
  kind,
  size = 20,
  color = DEFAULT,
  className,
  doubleBorder = false,
}: {
  kind: string;
  size?: number;
  color?: string;
  className?: string;
  doubleBorder?: boolean;
}) {
  return (
    <Frame size={size} className={className}>
      <rect x="2" y="5" width="20" height="14" rx="2.5" stroke={color} strokeWidth={doubleBorder ? 2.4 : 1.5} fill="#fff" />
      {kind === "transaction" && (
        <rect x="4.5" y="7.5" width="15" height="9" rx="1.5" stroke={color} strokeWidth={1.1} fill="none" />
      )}
      <g transform="translate(3.2 6.2) scale(0.48)">
        <TaskMarkerGlyph kind={kind === "transaction" ? "subProcess" : kind} color={color} />
      </g>
    </Frame>
  );
}

/** Gateway legend icon (diamond + marker). */
export function BpmnGatewayIcon({
  kind,
  size = 20,
  color = DEFAULT,
  className,
}: {
  kind: "exclusive" | "parallel" | "inclusive" | "eventBased";
  size?: number;
  color?: string;
  className?: string;
}) {
  return (
    <Frame size={size} className={className}>
      <polygon points="12,1.5 22.5,12 12,22.5 1.5,12" stroke={color} strokeWidth={1.5} fill="#fff" />
      {kind === "exclusive" && (
        <path d="M8 8 L16 16 M16 8 L8 16" stroke={color} strokeWidth={2} strokeLinecap="round" />
      )}
      {kind === "parallel" && (
        <path d="M12 6.5 V17.5 M6.5 12 H17.5" stroke={color} strokeWidth={2} strokeLinecap="round" />
      )}
      {kind === "inclusive" && <circle cx="12" cy="12" r="4.2" stroke={color} strokeWidth={1.8} fill="none" />}
      {kind === "eventBased" && (
        <g>
          <circle cx="12" cy="12" r="5.2" stroke={color} strokeWidth={1.3} fill="none" />
          <circle cx="12" cy="12" r="3.6" stroke={color} strokeWidth={1.1} fill="none" />
          <path d="M12 9.2 L14.2 14.2 H9.8 Z" stroke={color} strokeWidth={1} fill="none" strokeLinejoin="round" />
        </g>
      )}
    </Frame>
  );
}

/** Inner gateway marker only (for diamond canvas node). */
export function GatewayMarkerGlyph({
  kind,
  color = DEFAULT,
}: {
  kind: "exclusive" | "parallel" | "inclusive" | "eventBased";
  color?: string;
}) {
  if (kind === "exclusive") {
    return <path d="M8 8 L16 16 M16 8 L8 16" stroke={color} strokeWidth={2} strokeLinecap="round" />;
  }
  if (kind === "parallel") {
    return <path d="M12 6.5 V17.5 M6.5 12 H17.5" stroke={color} strokeWidth={2} strokeLinecap="round" />;
  }
  if (kind === "inclusive") {
    return <circle cx="12" cy="12" r="4.2" stroke={color} strokeWidth={1.8} fill="none" />;
  }
  return (
    <g>
      <circle cx="12" cy="12" r="5.2" stroke={color} strokeWidth={1.3} fill="none" />
      <circle cx="12" cy="12" r="3.6" stroke={color} strokeWidth={1.1} fill="none" />
      <path d="M12 9.2 L14.2 14.2 H9.8 Z" stroke={color} strokeWidth={1} fill="none" strokeLinejoin="round" />
    </g>
  );
}

export function PaletteBpmnIcon({
  originalType,
  size = 20,
}: {
  originalType: string;
  size?: number;
}): ReactNode {
  const parts = originalType.replace(/^bpmn:/, "").split(":");
  const base = parts[0];
  const kind = parts[1] || "none";

  if (base === "startEvent") return <BpmnEventIcon shape="start" kind={kind} style="catch" size={size} />;
  if (base === "endEvent") return <BpmnEventIcon shape="end" kind={kind} style="throw" size={size} />;
  if (base === "intermediateCatchEvent") {
    return <BpmnEventIcon shape="intermediate" kind={kind} style="catch" size={size} />;
  }
  if (base === "intermediateThrowEvent") {
    return <BpmnEventIcon shape="intermediate" kind={kind} style="throw" size={size} />;
  }
  if (base === "boundaryEvent") return <BpmnEventIcon shape="boundary" kind={kind} style="catch" size={size} />;

  if (base === "userTask") return <BpmnTaskIcon kind="userTask" size={size} />;
  if (base === "serviceTask") return <BpmnTaskIcon kind="serviceTask" size={size} />;
  if (base === "scriptTask") return <BpmnTaskIcon kind="scriptTask" size={size} />;
  if (base === "businessRuleTask") return <BpmnTaskIcon kind="businessRuleTask" size={size} />;
  if (base === "sendTask") return <BpmnTaskIcon kind="sendTask" size={size} />;
  if (base === "receiveTask") return <BpmnTaskIcon kind="receiveTask" size={size} />;
  if (base === "manualTask") return <BpmnTaskIcon kind="manualTask" size={size} />;
  if (base === "callActivity") return <BpmnTaskIcon kind="callActivity" size={size} doubleBorder />;
  if (base === "subProcess") return <BpmnTaskIcon kind="subProcess" size={size} />;
  if (base === "transaction") return <BpmnTaskIcon kind="transaction" size={size} doubleBorder />;

  if (base === "exclusiveGateway") return <BpmnGatewayIcon kind="exclusive" size={size} />;
  if (base === "parallelGateway") return <BpmnGatewayIcon kind="parallel" size={size} />;
  if (base === "inclusiveGateway") return <BpmnGatewayIcon kind="inclusive" size={size} />;
  if (base === "eventBasedGateway") return <BpmnGatewayIcon kind="eventBased" size={size} />;

  return null;
}
