import React, { useState, useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";

interface DebouncedInputProps extends React.ComponentProps<typeof Input> {
  value: string;
  onValueChange: (value: string) => void;
  debounce?: number;
}

export function DebouncedInput({ 
  value: initialValue, 
  onValueChange, 
  debounce = 300,
  ...props 
}: DebouncedInputProps) {
  const [value, setValue] = useState(initialValue);
  const isLocalUpdate = useRef(false);

  useEffect(() => {
    // Only update from props if we're not currently typing
    if (!isLocalUpdate.current) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setValue(initialValue);
    }
  }, [initialValue]);

  useEffect(() => {
    if (!isLocalUpdate.current) return;
    
    const timeout = setTimeout(() => {
      onValueChange(value);
      isLocalUpdate.current = false;
    }, debounce);

    return () => clearTimeout(timeout);
  }, [value, debounce, onValueChange]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    isLocalUpdate.current = true;
    setValue(e.target.value);
  };

  return (
    <Input
      {...props}
      value={value}
      onChange={handleChange}
    />
  );
}
