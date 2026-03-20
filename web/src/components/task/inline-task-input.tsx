"use client";

import { useState, useRef } from "react";
import { Plus } from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface InlineTaskInputProps {
  onSubmit: (title: string) => void;
}

export function InlineTaskInput({ onSubmit }: InlineTaskInputProps) {
  const { t } = useI18n();
  const [isEditing, setIsEditing] = useState(false);
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  function handleSubmit() {
    const trimmed = value.trim();
    if (!trimmed) return;
    onSubmit(trimmed);
    setValue("");
  }

  if (!isEditing) {
    return (
      <button
        onClick={() => {
          setIsEditing(true);
          setTimeout(() => inputRef.current?.focus(), 0);
        }}
        className="flex w-full cursor-pointer items-center gap-2 rounded-md border border-dashed border-border/50 px-3 py-2 text-sm text-muted-foreground hover:border-border hover:text-foreground transition-colors duration-200"
      >
        <Plus className="h-4 w-4" />
        {t("tasks.addTask")}
      </button>
    );
  }

  return (
    <input
      ref={inputRef}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === "Enter") { e.preventDefault(); handleSubmit(); }
        if (e.key === "Escape") { setIsEditing(false); setValue(""); }
      }}
      onBlur={() => { if (!value.trim()) setIsEditing(false); }}
      placeholder={t("tasks.titlePlaceholder")}
      className="w-full rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none focus:border-primary transition-colors duration-200"
    />
  );
}
