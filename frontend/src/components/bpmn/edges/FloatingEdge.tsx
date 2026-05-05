import { useCallback } from 'react';
import { useStore, getBezierPath, type EdgeProps, BaseEdge } from '@xyflow/react';

import { getEdgeParams } from '@/lib/floating-edge-utils';

function FloatingEdge({ source, target, markerEnd, style }: EdgeProps) {
  const sourceNode = useStore(
    useCallback((store) => store.nodeLookup.get(source), [source]),
  );
  const targetNode = useStore(
    useCallback((store) => store.nodeLookup.get(target), [target]),
  );

  if (!sourceNode || !targetNode) {
    return null;
  }

  const { sx, sy, tx, ty, sourcePos, targetPos } = getEdgeParams(
    sourceNode,
    targetNode,
  );

  const [edgePath] = getBezierPath({
    sourceX: sx,
    sourceY: sy,
    sourcePosition: sourcePos,
    targetPosition: targetPos,
    targetX: tx,
    targetY: ty,
  });

  return (
    <BaseEdge path={edgePath} markerEnd={markerEnd} style={style} />
  );
}

export default FloatingEdge;
