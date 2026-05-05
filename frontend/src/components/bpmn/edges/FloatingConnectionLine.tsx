import { getBezierPath, type ConnectionLineComponentProps, type Node } from '@xyflow/react';

import { getEdgeParams } from '@/lib/floating-edge-utils';

function FloatingConnectionLine({
  toX,
  toY,
  fromNode,
  connectionLineStyle,
}: ConnectionLineComponentProps) {
  if (!fromNode) {
    return null;
  }

  // Create a temporary target node for calculation
  const targetNode = {
    id: 'connection-target',
    measured: { width: 1, height: 1 },
    position: { x: toX, y: toY },
    data: {},
  };

  const { sx, sy, sourcePos, targetPos } = getEdgeParams(fromNode, targetNode as unknown as Node);
  
  const [edgePath] = getBezierPath({
    sourceX: sx,
    sourceY: sy,
    sourcePosition: sourcePos,
    targetX: toX,
    targetY: toY,
    targetPosition: targetPos,
  });

  return (
    <g>
      <path
        fill="none"
        stroke="#222"
        strokeWidth={1.5}
        className="animated"
        d={edgePath}
        style={connectionLineStyle}
      />
      <circle cx={toX} cy={toY} fill="#fff" r={3} stroke="#222" strokeWidth={1.5} />
    </g>
  );
}

export default FloatingConnectionLine;
