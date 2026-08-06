import React from "react";
import {
  Database,
  Box,
  Users,
  Rows3,
  Square,
  ArrowLeftRight,
  GitBranch,
  FileText,
  Globe,
  Webhook,
  Mail,
  HardDrive,
  Zap,
  type LucideProps,
} from "lucide-react";
import { PaletteBpmnIcon } from "./BpmnIcons";

interface PaletteProps {
  onDragStart: (event: React.DragEvent, nodeType: string, originalType: string, label: string) => void;
}

type LucideIcon = React.ForwardRefExoticComponent<
  Omit<LucideProps, "ref"> & React.RefAttributes<SVGSVGElement>
>;

interface PaletteItem {
  type: string;
  originalType: string;
  label: string;
  /** Product/connector icons stay Lucide; BPMN elements use PaletteBpmnIcon. */
  lucideIcon?: LucideIcon;
  color?: string;
}

interface PaletteCategory {
  title: string;
  items: PaletteItem[];
}

export default function Palette({ onDragStart }: PaletteProps) {
  const categories: PaletteCategory[] = [
    {
      title: "Events",
      items: [
        { type: "startEvent", originalType: "bpmn:startEvent", label: "Start Event" },
        { type: "startEvent", originalType: "bpmn:startEvent:message", label: "Message Start" },
        { type: "startEvent", originalType: "bpmn:startEvent:timer", label: "Timer Start" },
        { type: "startEvent", originalType: "bpmn:startEvent:signal", label: "Signal Start" },
        { type: "startEvent", originalType: "bpmn:startEvent:conditional", label: "Conditional Start" },
        { type: "startEvent", originalType: "bpmn:startEvent:escalation", label: "Escalation Start" },
        { type: "endEvent", originalType: "bpmn:endEvent", label: "End Event" },
        { type: "endEvent", originalType: "bpmn:endEvent:terminate", label: "Terminate End" },
        { type: "endEvent", originalType: "bpmn:endEvent:error", label: "Error End" },
        { type: "endEvent", originalType: "bpmn:endEvent:escalation", label: "Escalation End" },
        { type: "endEvent", originalType: "bpmn:endEvent:cancel", label: "Cancel End" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent:timer", label: "Timer Catch" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent:message", label: "Message Throw" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent:message", label: "Message Catch" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent:signal", label: "Signal Throw" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent:signal", label: "Signal Catch" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent:escalation", label: "Escalation Throw" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent:escalation", label: "Escalation Catch" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent:link", label: "Link Throw" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent:link", label: "Link Catch" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent:conditional", label: "Conditional Catch" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent:error", label: "Error Throw" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent:compensate", label: "Compensate Throw" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:timer", label: "Boundary Timer" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:message", label: "Boundary Message" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:signal", label: "Boundary Signal" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:error", label: "Boundary Error" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:escalation", label: "Boundary Escalation" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:cancel", label: "Boundary Cancel" },
        { type: "boundaryEvent", originalType: "bpmn:boundaryEvent:compensate", label: "Boundary Compensate" },
      ],
    },
    {
      title: "Connectors",
      items: [
        { type: "serviceTask", originalType: "bpmn:serviceTask:http", label: "HTTP Call", lucideIcon: Globe, color: "text-sky-600" },
        { type: "serviceTask", originalType: "bpmn:serviceTask:webhook", label: "Webhook", lucideIcon: Webhook, color: "text-sky-600" },
        { type: "serviceTask", originalType: "bpmn:serviceTask:kafka", label: "Kafka", lucideIcon: Zap, color: "text-sky-600" },
        { type: "serviceTask", originalType: "bpmn:serviceTask:email", label: "Email", lucideIcon: Mail, color: "text-sky-600" },
        { type: "serviceTask", originalType: "bpmn:serviceTask:s3", label: "S3", lucideIcon: HardDrive, color: "text-sky-600" },
      ],
    },
    {
      title: "Tasks",
      items: [
        { type: "userTask", originalType: "bpmn:userTask", label: "User Task" },
        { type: "serviceTask", originalType: "bpmn:serviceTask", label: "Service Task" },
        { type: "scriptTask", originalType: "bpmn:scriptTask", label: "Script Task" },
        { type: "businessRuleTask", originalType: "bpmn:businessRuleTask", label: "Business Rule" },
        { type: "sendTask", originalType: "bpmn:sendTask:http", label: "Send Task" },
        { type: "receiveTask", originalType: "bpmn:receiveTask", label: "Receive Task" },
        { type: "manualTask", originalType: "bpmn:manualTask", label: "Manual Task" },
        { type: "callActivity", originalType: "bpmn:callActivity", label: "Call Activity" },
        { type: "subProcess", originalType: "bpmn:subProcess", label: "Sub Process" },
        { type: "subProcess", originalType: "bpmn:transaction", label: "Transaction" },
      ],
    },
    {
      title: "Gateways",
      items: [
        { type: "exclusiveGateway", originalType: "bpmn:exclusiveGateway", label: "Exclusive" },
        { type: "parallelGateway", originalType: "bpmn:parallelGateway", label: "Parallel" },
        { type: "inclusiveGateway", originalType: "bpmn:inclusiveGateway", label: "Inclusive" },
        { type: "eventBasedGateway", originalType: "bpmn:eventBasedGateway", label: "Event Based" },
      ],
    },
    {
      title: "Visual",
      items: [
        { type: "visualArtifact", originalType: "bpmn:participant", label: "Pool", lucideIcon: Users, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:lane", label: "Lane", lucideIcon: Rows3, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:group", label: "Group", lucideIcon: Square, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:messageFlow", label: "Message Flow", lucideIcon: ArrowLeftRight, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:association", label: "Association", lucideIcon: GitBranch, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:dataObject", label: "Data Object", lucideIcon: Box, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:dataStoreReference", label: "Data Store", lucideIcon: Database, color: "text-slate-500" },
        { type: "visualArtifact", originalType: "bpmn:textAnnotation", label: "Annotation", lucideIcon: FileText, color: "text-slate-500" },
      ],
    },
  ];

  return (
    <div className="w-48 border-r bg-gray-50 flex flex-col py-3 shrink-0 overflow-y-auto overflow-x-hidden">
      {categories.map((category, idx) => (
        <div key={category.title} className="flex flex-col gap-1.5 w-full px-2 mb-3">
          <div className="px-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {category.title}
          </div>
          {category.items.map((item) => {
            const Lucide = item.lucideIcon;
            return (
              <div
                key={`${item.originalType}-${item.label}`}
                title={item.label}
                aria-label={item.label}
                className="w-full min-h-9 bg-white border rounded-md shadow-sm flex items-center gap-2 px-2 py-1.5 cursor-grab hover:bg-slate-100 hover:border-primary transition-colors"
                onDragStart={(event) => onDragStart(event, item.type, item.originalType, item.label)}
                draggable
              >
                {Lucide ? (
                  <Lucide className={`w-[18px] h-[18px] shrink-0 ${item.color || "text-slate-600"}`} />
                ) : (
                  <span className="shrink-0 w-[18px] h-[18px] flex items-center justify-center text-slate-800">
                    <PaletteBpmnIcon originalType={item.originalType} size={18} />
                  </span>
                )}
                <span className="text-xs text-slate-700 truncate">{item.label}</span>
              </div>
            );
          })}
          {idx < categories.length - 1 && <div className="w-full h-px bg-slate-200 mt-2" />}
        </div>
      ))}
    </div>
  );
}
