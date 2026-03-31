"use client";

import { Check, FileText, ListChecks, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";
import type { PendingToolCall } from "./types";

interface ChatToolConfirmProps {
  toolCall: PendingToolCall;
  onConfirm: () => void;
  onReject: () => void;
  executed?: boolean;
  disabled?: boolean;
}

const toolLabels: Record<string, { icon: typeof FileText; label: string }> = {
  write_proposal: { icon: FileText, label: "ai.toolWriteProposal" },
  create_acceptance_criteria: { icon: ListChecks, label: "ai.toolCreateAC" },
};

export function ChatToolConfirm({ toolCall, onConfirm, onReject, executed, disabled }: ChatToolConfirmProps) {
  const { t } = useI18n();
  const meta = toolLabels[toolCall.name] ?? { icon: FileText, label: toolCall.name };
  const Icon = meta.icon;

  // Show a preview of what will be written
  const preview = formatToolPreview(toolCall);

  return (
    <div className="mt-2 rounded-lg border border-amber-500/20 bg-amber-500/5 p-3 space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium text-amber-400">
        <Icon className="size-3.5" />
        {t(meta.label)}
      </div>

      {preview && (
        <div className="max-h-32 overflow-y-auto rounded-md bg-background/50 p-2 text-xs text-foreground/70">
          <pre className="whitespace-pre-wrap">{preview}</pre>
        </div>
      )}

      {executed ? (
        <div className="flex items-center gap-1 text-xs text-emerald-500">
          <Check className="size-3" />
          {t("ai.toolExecuted")}
        </div>
      ) : (
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={onConfirm} disabled={disabled} className="cursor-pointer">
            <Check className="size-3.5" />
            {t("ai.toolConfirm")}
          </Button>
          <Button size="sm" variant="ghost" onClick={onReject} disabled={disabled} className="cursor-pointer">
            <X className="size-3.5" />
            {t("ai.toolReject")}
          </Button>
        </div>
      )}
    </div>
  );
}

function formatToolPreview(toolCall: PendingToolCall): string {
  const args = toolCall.args;

  if (toolCall.name === "write_proposal") {
    const parts: string[] = [];
    if (args.problem) parts.push(`Problem: ${String(args.problem).slice(0, 200)}`);
    if (args.scope) parts.push(`Scope: ${String(args.scope).slice(0, 200)}`);
    if (args.out_of_scope) parts.push(`Out of Scope: ${String(args.out_of_scope).slice(0, 200)}`);
    return parts.join("\n\n");
  }

  if (toolCall.name === "create_acceptance_criteria") {
    const criteria = args.criteria;
    if (Array.isArray(criteria)) {
      return criteria.map((c: Record<string, unknown>) => `- ${c.scenario ?? ""}`).join("\n");
    }
  }

  return JSON.stringify(args, null, 2);
}
