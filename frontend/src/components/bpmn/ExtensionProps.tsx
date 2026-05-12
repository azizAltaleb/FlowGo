import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Plus, Trash2 } from "lucide-react";
import { DebouncedInput } from "@/components/ui/debounced-input";

interface ExtensionPropsProps {
  data: Record<string, unknown> | null;
  onUpdate: (newData: Record<string, unknown>) => void;
}

interface WorkflowProperty {
  "@_name": string;
  "@_value": string;
}

// Fallback for crypto.randomUUID if not available
const generateId = () => {
    if (typeof crypto !== 'undefined' && crypto.randomUUID) {
        return crypto.randomUUID();
    }
    return Math.random().toString(36).substring(2) + Date.now().toString(36);
};

export default function ExtensionProps({ data, onUpdate }: ExtensionPropsProps) {
  const [properties, setProperties] = useState<{ id: string; name: string; value: string }[]>([]);
  const isLocalChange = useRef(false);

  useEffect(() => {
    const loadProperties = () => {
      let propsArray: WorkflowProperty[] = [];
      
      if (data && data["bpmn:extensionElements"]) {
          const extensionElements = data["bpmn:extensionElements"] as Record<string, unknown>;
          const workflowProperties = extensionElements["flowgo:properties"] as Record<string, unknown>;
          
          if (workflowProperties && workflowProperties["flowgo:property"]) {
              const props = workflowProperties["flowgo:property"];
              propsArray = (Array.isArray(props) ? props : [props]) as WorkflowProperty[];
          }
      }

      // Check if incoming props match our local state
      // We compare only name/value, as ID is local-only concept
      const isSynced = propsArray.length === properties.length && propsArray.every((p, i) => 
          p["@_name"] === properties[i].name && p["@_value"] === properties[i].value
      );

      if (isSynced) {
          // We are in sync with parent, reset local change flag
          isLocalChange.current = false;
          return;
      }

      // If we have a local change pending, and props don't match, assume props are stale
      if (isLocalChange.current) {
          return;
      }

      // External update or initial load - Sync local state
      // Reconcile IDs to preserve focus
      setProperties(prevProps => {
          return propsArray.map((p, i) => {
              const existing = prevProps[i];
              // Try to reuse existing ID by index if reasonable, or generate new
              return {
                  id: (existing && existing.id) || generateId(), 
                  name: p["@_name"],
                  value: p["@_value"]
              };
          });
      });
    };

    loadProperties();
  }, [data, properties]);

  const updateBackend = (newProps: { name: string; value: string }[]) => {
    isLocalChange.current = true;
    
    let newExtensionElements = null;
    
    if (newProps.length > 0) {
        newExtensionElements = {
            "flowgo:properties": {
                "flowgo:property": newProps.map(p => ({
                    "@_name": p.name,
                    "@_value": p.value
                }))
            }
        };
    }

    const newData = { ...data };
    if (newExtensionElements) {
        newData["bpmn:extensionElements"] = newExtensionElements;
    } else {
        delete newData["bpmn:extensionElements"];
    }

    onUpdate(newData);
  };

  const handleAdd = () => {
    const newProps = [...properties, { id: generateId(), name: "", value: "" }];
    setProperties(newProps);
    updateBackend(newProps);
  };

  const handleRemove = (index: number) => {
    const newProps = properties.filter((_, i) => i !== index);
    setProperties(newProps);
    updateBackend(newProps);
  };

  const handleChange = (index: number, key: "name" | "value", val: string) => {
    const newProps = [...properties];
    newProps[index] = { ...newProps[index], [key]: val };
    setProperties(newProps);
    updateBackend(newProps);
  };

  return (
    <div className="pt-4 border-t">
        <div className="flex justify-between items-center mb-3">
            <h4 className="text-sm font-semibold">Extension Properties</h4>
            <Button variant="ghost" size="sm" onClick={handleAdd}>
                <Plus className="h-4 w-4" />
            </Button>
        </div>
        <div className="space-y-2">
            {properties.map((prop, idx) => (
                <div key={prop.id} className="flex space-x-2">
                    <DebouncedInput 
                        placeholder="Name" 
                        value={prop.name} 
                        onValueChange={(val) => handleChange(idx, "name", val)}
                        className="h-8 text-xs bg-white"
                    />
                    <DebouncedInput 
                        placeholder="Value" 
                        value={prop.value} 
                        onValueChange={(val) => handleChange(idx, "value", val)}
                        className="h-8 text-xs bg-white"
                    />
                    <Button variant="ghost" size="sm" onClick={() => handleRemove(idx)} className="h-8 w-8 p-0">
                        <Trash2 className="h-4 w-4" />
                    </Button>
                </div>
            ))}
            {properties.length === 0 && (
                <p className="text-xs text-muted-foreground italic">No properties defined.</p>
            )}
        </div>
    </div>
  );
}
