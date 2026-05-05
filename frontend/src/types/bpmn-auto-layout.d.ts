declare module 'bpmn-auto-layout' {
  export function layoutProcess(xml: string): Promise<string>;
}
