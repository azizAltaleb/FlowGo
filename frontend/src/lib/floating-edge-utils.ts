import { Position, type Node, type InternalNode } from '@xyflow/react';

function getNodeDimensions(node: InternalNode | Node) {
  const measured = node.measured || {};
  const width = measured.width ?? (node.style?.width as number) ?? (node.data?.width as number) ?? 0;
  const height = measured.height ?? (node.style?.height as number) ?? (node.data?.height as number) ?? 0;
  return { width, height };
}

// Get the intersection point between the line from center(A) to center(B) and the bounding box of Node A
function getNodeIntersection(intersectionNode: InternalNode | Node, targetNode: InternalNode | Node) {
  const { width: w, height: h } = getNodeDimensions(intersectionNode);
  const intersectionNodePosition = intersectionNode.position;
  
  const { width: targetW, height: targetH } = getNodeDimensions(targetNode);
  const targetNodePosition = targetNode.position;

  const w2 = w / 2;
  const h2 = h / 2;

  const x2 = intersectionNodePosition.x + w2;
  const y2 = intersectionNodePosition.y + h2;

  const targetX2 = targetNodePosition.x + targetW / 2;
  const targetY2 = targetNodePosition.y + targetH / 2;

  const xx1 = (targetX2 - x2) / (w2 || 1); // Avoid divide by zero
  const yy1 = (targetY2 - y2) / (h2 || 1);

  const a = 1 / (Math.abs(xx1) + Math.abs(yy1) || 1);
  const xx3 = a * xx1;
  const yy3 = a * yy1;

  const x = w2 * (xx3 + 1);
  const y = h2 * (yy3 + 1);

  return { x: intersectionNodePosition.x + x, y: intersectionNodePosition.y + y }; // Return absolute coordinate
}

// Returns the parameters (sx, sy, tx, ty, sourcePos, targetPos) for the edge
export function getEdgeParams(source: InternalNode | Node, target: InternalNode | Node) {
  const sourceIntersectionPoint = getNodeIntersection(source, target);
  const targetIntersectionPoint = getNodeIntersection(target, source);

  const sourcePos = getEdgePosition(source, sourceIntersectionPoint);
  const targetPos = getEdgePosition(target, targetIntersectionPoint);

  return {
    sx: sourceIntersectionPoint.x,
    sy: sourceIntersectionPoint.y,
    tx: targetIntersectionPoint.x,
    ty: targetIntersectionPoint.y,
    sourcePos,
    targetPos,
  };
}

function getEdgePosition(node: InternalNode | Node, intersectionPoint: { x: number; y: number }) {
  const { width, height } = getNodeDimensions(node);
  const position = node.position;
  
  const center = {
    x: position.x + width / 2,
    y: position.y + height / 2,
  };

  if (intersectionPoint.x < center.x && Math.abs(intersectionPoint.y - center.y) < height / 2) {
    return Position.Left;
  }
  if (intersectionPoint.x > center.x && Math.abs(intersectionPoint.y - center.y) < height / 2) {
    return Position.Right;
  }
  if (intersectionPoint.y < center.y && Math.abs(intersectionPoint.x - center.x) < width / 2) {
    return Position.Top;
  }
  return Position.Bottom;
}
