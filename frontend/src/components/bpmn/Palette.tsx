import React from 'react';
import { 
  User, Settings, FileCode, MessageSquare, 
  ClipboardList, HelpingHand, ExternalLink, 
  Layers, X, Plus, Circle, Play, Ban, Timer, Zap,
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
    <div className="w-[60px] border-r bg-gray-50 flex flex-col items-center py-4 gap-6 overflow-y-auto shrink-0">
      {categories.map((category, idx) => (
        <div key={idx} className="flex flex-col gap-2 w-full px-2">
          {category.items.map((item) => (
            <div
              key={item.originalType}
              className="w-full aspect-square bg-white border rounded-md shadow-sm flex items-center justify-center cursor-grab hover:bg-slate-100 hover:border-primary transition-colors group relative"
              onDragStart={(event) => onDragStart(event, item.type, item.originalType, item.label)}
              draggable
            >
              <item.icon className={`w-5 h-5 ${item.color || "text-slate-600"}`} />
              
              {/* Tooltip */}
              <div className="absolute left-full ml-2 px-2 py-1 bg-slate-800 text-white text-xs rounded opacity-0 group-hover:opacity-100 pointer-events-none whitespace-nowrap z-50 transition-opacity">
                {item.label}
              </div>
            </div>
          ))}
          {idx < categories.length - 1 && <div className="w-full h-px bg-slate-200 my-1" />}
        </div>
      ))}
    </div>
  );
}
