import { type NodeProps } from "@xyflow/react";

/** Tier-3 visual-only collaboration / documentation shapes (not executable). */
export function VisualArtifactNode({ data, selected }: NodeProps) {
  const label = (data.label as string) || (data.originalType as string) || "Artifact";
  const kind = String(data.visualKind || "artifact");
  return (
    <div
      className={`rounded border border-dashed px-2 py-1 text-[10px] text-slate-600 bg-slate-50/80 ${
        selected ? "border-primary ring-1 ring-primary/30" : "border-slate-300"
      }`}
      title="Visual-only (not executed by the engine)"
      data-visual-only="true"
      data-visual-kind={kind}
    >
      <div className="uppercase tracking-wide text-[9px] text-muted-foreground">{kind}</div>
      <div className="truncate max-w-[140px]">{label}</div>
    </div>
  );
}
