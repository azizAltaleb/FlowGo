import { useMemo, useState } from "react";
import { Plus, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Variables = Record<string, unknown>;
type VariableType = "string" | "number" | "boolean" | "null" | "object" | "array";

interface VariablesEditorProps {
  value: Variables;
  editable?: boolean;
  onChange?: (value: Variables) => void;
  compact?: boolean;
}

function variableType(value: unknown): VariableType {
  if (value === null) return "null";
  if (Array.isArray(value)) return "array";
  if (typeof value === "number") return "number";
  if (typeof value === "boolean") return "boolean";
  if (typeof value === "object") return "object";
  return "string";
}

function defaultValue(type: VariableType): unknown {
  switch (type) {
    case "number":
      return 0;
    case "boolean":
      return false;
    case "null":
      return null;
    case "object":
      return {};
    case "array":
      return [];
    default:
      return "";
  }
}

function typeLabel(type: VariableType): string {
  if (type === "string") return "Text";
  if (type === "array") return "List";
  return type.charAt(0).toUpperCase() + type.slice(1);
}

function displayPrimitive(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "boolean") return value ? "True" : "False";
  if (value === undefined || value === "") return "—";
  return String(value);
}

function TypeSelect({
  value,
  onChange,
  ariaLabel,
}: {
  value: VariableType;
  onChange: (type: VariableType) => void;
  ariaLabel: string;
}) {
  return (
    <select
      aria-label={ariaLabel}
      className="h-9 rounded-md border border-input bg-background px-2 text-xs"
      value={value}
      onChange={(event) => onChange(event.target.value as VariableType)}
    >
      <option value="string">Text</option>
      <option value="number">Number</option>
      <option value="boolean">Boolean</option>
      <option value="null">Null</option>
      <option value="object">Object</option>
      <option value="array">List</option>
    </select>
  );
}

function PrimitiveEditor({
  value,
  onChange,
}: {
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const type = variableType(value);
  if (type === "boolean") {
    return (
      <select
        className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
        value={value ? "true" : "false"}
        onChange={(event) => onChange(event.target.value === "true")}
      >
        <option value="true">True</option>
        <option value="false">False</option>
      </select>
    );
  }
  if (type === "null") {
    return <div className="h-9 rounded-md border bg-muted px-3 py-2 text-sm text-muted-foreground">Null</div>;
  }
  return (
    <Input
      type={type === "number" ? "number" : "text"}
      value={String(value ?? "")}
      onChange={(event) => {
        if (type === "number") {
          const next = event.target.value;
          onChange(next === "" ? 0 : Number(next));
          return;
        }
        onChange(event.target.value);
      }}
    />
  );
}

function NestedObjectEditor({
  value,
  editable,
  onChange,
  depth,
}: {
  value: Variables;
  editable: boolean;
  onChange: (value: Variables) => void;
  depth: number;
}) {
  const [newKey, setNewKey] = useState("");
  const [newType, setNewType] = useState<VariableType>("string");
  const entries = Object.entries(value || {});

  return (
    <div className={`space-y-2 ${depth > 0 ? "border-l pl-3" : ""}`}>
      {entries.length === 0 ? (
        <div className="text-sm italic text-muted-foreground">Empty object</div>
      ) : (
        entries.map(([key, child]) => (
          <div key={key} className="grid grid-cols-[minmax(90px,0.3fr)_1fr_auto] items-start gap-2">
            <div className="pt-2 text-xs font-medium text-muted-foreground break-all">{key}</div>
            <VariableValue
              value={child}
              editable={editable}
              depth={depth + 1}
              onChange={(next) => onChange({ ...value, [key]: next })}
            />
            {editable ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-9 w-9 shrink-0 text-muted-foreground hover:text-destructive"
                aria-label={`Remove ${key}`}
                onClick={() => {
                  const next = { ...value };
                  delete next[key];
                  onChange(next);
                }}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            ) : null}
          </div>
        ))
      )}
      {editable ? (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <Input
            className="min-w-[120px] flex-1"
            placeholder="Property name"
            value={newKey}
            onChange={(event) => setNewKey(event.target.value)}
          />
          <TypeSelect
            ariaLabel="New object property type"
            value={newType}
            onChange={setNewType}
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!newKey.trim() || Object.prototype.hasOwnProperty.call(value, newKey.trim())}
            onClick={() => {
              const key = newKey.trim();
              if (!key) return;
              onChange({ ...value, [key]: defaultValue(newType) });
              setNewKey("");
              setNewType("string");
            }}
          >
            <Plus className="mr-1 h-3.5 w-3.5" />
            Add field
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function NestedArrayEditor({
  value,
  editable,
  onChange,
  depth,
}: {
  value: unknown[];
  editable: boolean;
  onChange: (value: unknown[]) => void;
  depth: number;
}) {
  const [newType, setNewType] = useState<VariableType>("string");

  return (
    <div className={`space-y-2 ${depth > 0 ? "border-l pl-3" : ""}`}>
      {value.length === 0 ? (
        <div className="text-sm italic text-muted-foreground">Empty list</div>
      ) : (
        value.map((child, index) => (
          <div key={index} className="grid grid-cols-[minmax(70px,0.25fr)_1fr_auto] items-start gap-2">
            <div className="pt-2 text-xs font-medium text-muted-foreground">Item {index + 1}</div>
            <VariableValue
              value={child}
              editable={editable}
              depth={depth + 1}
              onChange={(next) => {
                const copy = [...value];
                copy[index] = next;
                onChange(copy);
              }}
            />
            {editable ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-9 w-9 shrink-0 text-muted-foreground hover:text-destructive"
                aria-label={`Remove item ${index + 1}`}
                onClick={() => onChange(value.filter((_, i) => i !== index))}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            ) : null}
          </div>
        ))
      )}
      {editable ? (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <TypeSelect
            ariaLabel="New list item type"
            value={newType}
            onChange={setNewType}
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              onChange([...value, defaultValue(newType)]);
              setNewType("string");
            }}
          >
            <Plus className="mr-1 h-3.5 w-3.5" />
            Add item
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function VariableValue({
  value,
  editable,
  onChange,
  depth = 0,
}: {
  value: unknown;
  editable: boolean;
  onChange: (value: unknown) => void;
  depth?: number;
}) {
  const type = variableType(value);
  if (type === "object") {
    return (
      <NestedObjectEditor
        value={(value as Variables) || {}}
        editable={editable}
        onChange={onChange}
        depth={depth}
      />
    );
  }
  if (type === "array") {
    return (
      <NestedArrayEditor
        value={Array.isArray(value) ? value : []}
        editable={editable}
        onChange={onChange}
        depth={depth}
      />
    );
  }
  return editable ? (
    <PrimitiveEditor value={value} onChange={onChange} />
  ) : (
    <span className="break-words text-sm">{displayPrimitive(value)}</span>
  );
}

export default function VariablesEditor({
  value,
  editable = false,
  onChange,
  compact = false,
}: VariablesEditorProps) {
  const [newName, setNewName] = useState("");
  const [newType, setNewType] = useState<VariableType>("string");
  const entries = useMemo(() => Object.entries(value || {}), [value]);

  const update = (next: Variables) => onChange?.(next);

  return (
    <div className="space-y-3">
      {entries.length === 0 ? (
        <div className="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
          No variables available.
        </div>
      ) : (
        <div className="overflow-hidden rounded-md border">
          <div className="grid grid-cols-[minmax(130px,0.32fr)_100px_1fr_auto] gap-3 border-b bg-muted/50 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            <span>Name</span>
            <span>Type</span>
            <span>Value</span>
            {editable ? <span className="sr-only">Actions</span> : <span />}
          </div>
          <div className="divide-y">
            {entries.map(([name, current]) => (
              <div
                key={name}
                className={`grid grid-cols-[minmax(130px,0.32fr)_100px_1fr_auto] items-start gap-3 px-3 ${
                  compact ? "py-2" : "py-3"
                }`}
              >
                <div className="break-all pt-2 text-sm font-medium">{name}</div>
                {editable ? (
                  <TypeSelect
                    ariaLabel={`Type for ${name}`}
                    value={variableType(current)}
                    onChange={(type) => update({ ...value, [name]: defaultValue(type) })}
                  />
                ) : (
                  <span className="pt-2 text-xs capitalize text-muted-foreground">
                    {typeLabel(variableType(current))}
                  </span>
                )}
                <VariableValue
                  value={current}
                  editable={editable}
                  onChange={(next) => update({ ...value, [name]: next })}
                />
                {editable ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 shrink-0 text-muted-foreground hover:text-destructive"
                    aria-label={`Remove ${name}`}
                    onClick={() => {
                      const next = { ...value };
                      delete next[name];
                      update(next);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                ) : (
                  <span />
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {editable && (
        <div className="rounded-md border border-dashed bg-muted/20 p-3">
          <Label className="mb-2 block text-xs uppercase tracking-wide text-muted-foreground">
            Add variable
          </Label>
          <div className="flex flex-wrap gap-2">
            <Input
              className="min-w-[180px] flex-1"
              placeholder="Variable name"
              value={newName}
              onChange={(event) => setNewName(event.target.value)}
            />
            <TypeSelect
              ariaLabel="New variable type"
              value={newType}
              onChange={setNewType}
            />
            <Button
              type="button"
              variant="outline"
              disabled={!newName.trim() || Object.prototype.hasOwnProperty.call(value, newName.trim())}
              onClick={() => {
                const name = newName.trim();
                if (!name) return;
                update({ ...value, [name]: defaultValue(newType) });
                setNewName("");
                setNewType("string");
              }}
            >
              <Plus className="mr-2 h-4 w-4" />
              Add
            </Button>
          </div>
          {newName.trim() && Object.prototype.hasOwnProperty.call(value, newName.trim()) ? (
            <div className="mt-2 flex items-center gap-1 text-xs text-destructive">
              <X className="h-3 w-3" />
              This variable already exists.
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
}
