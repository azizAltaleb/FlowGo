import React, { useEffect, useRef } from "react";
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
  const inputRef = useRef<HTMLInputElement>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isLocalUpdate = useRef(false);

  useEffect(() => {
    // Only update from props if we're not currently typing
    if (!isLocalUpdate.current && inputRef.current) {
      inputRef.current.value = initialValue;
    }
  }, [initialValue]);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const nextValue = e.target.value;
    isLocalUpdate.current = true;

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      isLocalUpdate.current = false;
      onValueChange(nextValue);
    }, debounce);
  };

  return (
    <Input
      {...props}
      ref={inputRef}
      defaultValue={initialValue}
      onChange={handleChange}
    />
  );
}
