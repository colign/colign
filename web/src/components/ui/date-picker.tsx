"use client";

import { useState, useRef, useEffect } from "react";
import { Calendar as CalendarIcon } from "lucide-react";
import { Calendar } from "@/components/ui/calendar";
import { cn } from "@/lib/utils";

interface DatePickerProps {
  value?: string;
  placeholder?: string;
  onChange?: (value: string | null) => void;
}

export function DatePicker({ value, placeholder = "Pick a date", onChange }: DatePickerProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const selected = value ? new Date(value) : undefined;

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const handleSelect = (day: Date | undefined) => {
    if (day) {
      onChange?.(day.toISOString().slice(0, 10));
    } else {
      onChange?.(null);
    }
    setOpen(false);
  };

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2 py-1 text-sm transition-colors hover:bg-accent",
          selected ? "text-foreground/80" : "text-muted-foreground/60",
        )}
      >
        <CalendarIcon className="size-3.5 text-muted-foreground/60" />
        <span>{selected ? selected.toLocaleDateString() : placeholder}</span>
      </button>

      {open && (
        <div className="absolute top-full left-0 z-50 mt-1 rounded-lg border border-border bg-popover shadow-lg">
          <Calendar
            mode="single"
            selected={selected}
            onSelect={handleSelect}
          />
        </div>
      )}
    </div>
  );
}
