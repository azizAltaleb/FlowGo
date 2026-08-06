import { useCallback, useRef } from "react";
import type { Edge, Node } from "@xyflow/react";

type Snapshot = { nodes: Node[]; edges: Edge[] };

const MAX_HISTORY = 60;

function cloneGraph(nodes: Node[], edges: Edge[]): Snapshot {
  return {
    nodes: structuredClone(nodes),
    edges: structuredClone(edges),
  };
}

/**
 * Undo/redo stack for React Flow diagrams.
 * Call `record()` before a mutating edit; `undo`/`redo` restore previous graphs.
 */
export function useDiagramHistory(
  getGraph: () => Snapshot,
  applyGraph: (snapshot: Snapshot) => void,
) {
  const pastRef = useRef<Snapshot[]>([]);
  const futureRef = useRef<Snapshot[]>([]);
  const applyingRef = useRef(false);

  const reset = useCallback(() => {
    pastRef.current = [];
    futureRef.current = [];
  }, []);

  const record = useCallback(() => {
    if (applyingRef.current) return;
    const current = getGraph();
    pastRef.current.push(cloneGraph(current.nodes, current.edges));
    if (pastRef.current.length > MAX_HISTORY) {
      pastRef.current.shift();
    }
    futureRef.current = [];
  }, [getGraph]);

  const undo = useCallback(() => {
    const past = pastRef.current;
    if (past.length === 0) return false;
    const current = getGraph();
    const previous = past.pop()!;
    futureRef.current.push(cloneGraph(current.nodes, current.edges));
    applyingRef.current = true;
    applyGraph(previous);
    applyingRef.current = false;
    return true;
  }, [applyGraph, getGraph]);

  const redo = useCallback(() => {
    const future = futureRef.current;
    if (future.length === 0) return false;
    const current = getGraph();
    const next = future.pop()!;
    pastRef.current.push(cloneGraph(current.nodes, current.edges));
    applyingRef.current = true;
    applyGraph(next);
    applyingRef.current = false;
    return true;
  }, [applyGraph, getGraph]);

  return { record, undo, redo, reset, isApplying: () => applyingRef.current };
}
