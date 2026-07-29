import React from 'react';
import { 
  User, Settings, FileCode, MessageSquare, 
  ClipboardList, HelpingHand, ExternalLink, 
  Layers, X, Plus, Circle, Play, Ban, Timer, Zap, Send,
  type LucideProps
} from "lucide-react";

interface PaletteProps {
  onDragStart: (event: React.DragEvent, nodeType: string, originalType: string, label: string) => void;
}

interface PaletteItem {
  type: string;
  originalType: string;
  label: string;
  icon: React.ForwardRefExoticComponent<Omit<LucideProps, "ref"> & React.RefAttributes<SVGSVGElement>>;
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
        { type: "startEvent", originalType: "bpmn:startEvent", label: "Start Event", icon: Play, color: "text-green-600" },
        { type: "endEvent", originalType: "bpmn:endEvent", label: "End Event", icon: Ban, color: "text-red-600" },
        { type: "intermediateCatchEvent", originalType: "bpmn:intermediateCatchEvent", label: "Timer Catch", icon: Timer, color: "text-yellow-600" },
        { type: "intermediateThrowEvent", originalType: "bpmn:intermediateThrowEvent", label: "Message Throw", icon: Zap, color: "text-blue-600" },
      ]
    },
    {
      title: "Tasks",
      items: [
        { type: "userTask", originalType: "bpmn:userTask", label: "User Task", icon: User },
        { type: "serviceTask", originalType: "bpmn:serviceTask", label: "Service Task", icon: Settings },
        { type: "scriptTask", originalType: "bpmn:scriptTask", label: "Script Task", icon: FileCode },
        { type: "businessRuleTask", originalType: "bpmn:businessRuleTask", label: "Business Rule", icon: ClipboardList },
        { type: "sendTask", originalType: "bpmn:sendTask", label: "Send Task", icon: Send },
        { type: "receiveTask", originalType: "bpmn:receiveTask", label: "Receive Task", icon: MessageSquare },
        { type: "manualTask", originalType: "bpmn:manualTask", label: "Manual Task", icon: HelpingHand },
        { type: "callActivity", originalType: "bpmn:callActivity", label: "Call Activity", icon: ExternalLink },
        { type: "subProcess", originalType: "bpmn:subProcess", label: "Sub Process", icon: Layers },
      ]
    },
    {
      title: "Gateways",
      items: [
        { type: "exclusiveGateway", originalType: "bpmn:exclusiveGateway", label: "Exclusive", icon: X, color: "text-amber-600" },
        { type: "parallelGateway", originalType: "bpmn:parallelGateway", label: "Parallel", icon: Plus, color: "text-amber-600" },
        { type: "inclusiveGateway", originalType: "bpmn:inclusiveGateway", label: "Inclusive", icon: Circle, color: "text-amber-600" },
        { type: "eventBasedGateway", originalType: "bpmn:eventBasedGateway", label: "Event Based", icon: Zap, color: "text-amber-600" },
      ]
    }
  ];

  return (
    <div className="w-48 border-r bg-gray-50 flex flex-col py-3 shrink-0 overflow-y-auto overflow-x-hidden">
      {categories.map((category, idx) => (
        <div key={category.title} className="flex flex-col gap-1.5 w-full px-2 mb-3">
          <div className="px-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {category.title}
          </div>
          {category.items.map((item) => (
            <div
              key={item.originalType}
              title={item.label}
              aria-label={item.label}
              className="w-full min-h-9 bg-white border rounded-md shadow-sm flex items-center gap-2 px-2 py-1.5 cursor-grab hover:bg-slate-100 hover:border-primary transition-colors"
              onDragStart={(event) => onDragStart(event, item.type, item.originalType, item.label)}
              draggable
            >
              <item.icon className={`w-4 h-4 shrink-0 ${item.color || "text-slate-600"}`} />
              <span className="text-xs text-slate-700 truncate">{item.label}</span>
            </div>
          ))}
          {idx < categories.length - 1 && <div className="w-full h-px bg-slate-200 mt-2" />}
        </div>
      ))}
    </div>
  );
}
