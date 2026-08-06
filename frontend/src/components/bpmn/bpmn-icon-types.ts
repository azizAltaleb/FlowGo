/** BPMN throw markers are filled; catch/start markers are stroked. */
export type MarkerStyle = "catch" | "throw";

export function eventThrowStyle(originalType: string): MarkerStyle {
  if (originalType.includes("Throw") || originalType === "bpmn:endEvent") {
    return "throw";
  }
  return "catch";
}
