"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Check, RefreshCw, Sparkles, Square, CheckSquare, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { generateAC, GeneratedAC } from "@/lib/ai";
import { useI18n } from "@/lib/i18n";
import { showError } from "@/lib/toast";

type GeneratorState = "idle" | "loading" | "previewing" | "error";

interface AIACGeneratorProps {
  changeId: number | bigint;
  hasProposal: boolean;
  hasExistingAC: boolean;
  onApply: (acs: GeneratedAC[]) => Promise<void>;
}

const keywordColor: Record<string, string> = {
  Given: "text-blue-400",
  When: "text-amber-400",
  Then: "text-emerald-400",
  And: "text-purple-400",
  But: "text-red-400",
};

export function AIACGenerator({ changeId, hasProposal, hasExistingAC, onApply }: AIACGeneratorProps) {
  const { t } = useI18n();
  const [state, setState] = useState<GeneratorState>("idle");
  const [generatedACs, setGeneratedACs] = useState<GeneratedAC[]>([]);
  const [selected, setSelected] = useState<boolean[]>([]);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [regenerateCooldown, setRegenerateCooldown] = useState(false);
  const [applying, setApplying] = useState(false);

  const isMounted = useRef(true);
  useEffect(() => {
    return () => {
      isMounted.current = false;
    };
  }, []);

  const startGeneration = useCallback(async () => {
    setState("loading");
    setErrorMessage(null);
    setGeneratedACs([]);
    setSelected([]);

    try {
      const acs = await generateAC(changeId);
      if (!isMounted.current) return;
      setGeneratedACs(acs);
      setSelected(acs.map(() => true));
      setState("previewing");
    } catch (err) {
      if (!isMounted.current) return;
      const msg = err instanceof Error ? err.message : String(err);
      setErrorMessage(msg);
      setState("error");
      showError(t("ai.connectionError"), err);
    }
  }, [changeId, t]);

  function handleRegenerate() {
    if (regenerateCooldown) return;
    setRegenerateCooldown(true);
    setTimeout(() => setRegenerateCooldown(false), 3000);
    startGeneration();
  }

  async function handleApplySelected() {
    const selectedACs = generatedACs.filter((_, i) => selected[i]);
    if (selectedACs.length === 0) return;

    setApplying(true);
    try {
      await onApply(selectedACs);
      setState("idle");
      setGeneratedACs([]);
      setSelected([]);
    } catch (err) {
      showError(t("toast.acCreateFailed"), err);
    } finally {
      if (isMounted.current) {
        setApplying(false);
      }
    }
  }

  function handleCancel() {
    setState("idle");
    setGeneratedACs([]);
    setSelected([]);
    setErrorMessage(null);
  }

  function toggleSelect(index: number) {
    setSelected((prev) => prev.map((v, i) => (i === index ? !v : v)));
  }

  function handleSelectAll() {
    setSelected(generatedACs.map(() => true));
  }

  function handleDeselectAll() {
    setSelected(generatedACs.map(() => false));
  }

  const allSelected = selected.length > 0 && selected.every(Boolean);
  const selectedCount = selected.filter(Boolean).length;

  // --- Loading state ---
  if (state === "loading") {
    return (
      <div className="mt-4 space-y-2" aria-live="polite">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Sparkles className="size-3.5 animate-pulse text-primary" />
          <span>{t("ai.loading")}</span>
        </div>
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="animate-pulse rounded-lg border border-border/50 p-3 space-y-2">
            <div className="h-3 w-2/5 rounded bg-muted" />
            <div className="space-y-1.5">
              <div className="h-2.5 w-3/4 rounded bg-muted/70" />
              <div className="h-2.5 w-4/5 rounded bg-muted/70" />
              <div className="h-2.5 w-1/2 rounded bg-muted/70" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  // --- Previewing state ---
  if (state === "previewing") {
    return (
      <div className="mt-4 space-y-2" aria-live="polite">
        <div className="flex items-center justify-between mb-1">
          <div className="flex items-center gap-2">
            <Sparkles className="size-3.5 text-primary" />
            <span className="text-xs font-medium text-foreground">{t("ai.generateAC")}</span>
          </div>
          <button
            onClick={allSelected ? handleDeselectAll : handleSelectAll}
            className="flex cursor-pointer items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {allSelected ? (
              <>
                <CheckSquare className="size-3.5" />
                {t("ai.deselectAll")}
              </>
            ) : (
              <>
                <Square className="size-3.5" />
                {t("ai.selectAll")}
              </>
            )}
          </button>
        </div>

        {generatedACs.map((ac, index) => {
          const isSelected = selected[index] ?? false;
          return (
            <button
              key={index}
              onClick={() => toggleSelect(index)}
              className={`w-full cursor-pointer rounded-lg border p-3 text-left transition-all ${
                isSelected
                  ? "border-primary/40 bg-primary/5"
                  : "border-border/50 opacity-60 hover:border-border"
              }`}
            >
              <div className="flex items-start gap-2.5">
                <div
                  className={`mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border-2 transition-colors ${
                    isSelected ? "border-primary bg-primary" : "border-muted-foreground/30"
                  }`}
                >
                  {isSelected && <Check className="size-2.5 text-primary-foreground" />}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="mb-1.5 text-sm font-medium">{ac.scenario}</p>
                  <div className="text-xs text-foreground/70 space-y-0.5">
                    {ac.steps.map((step, si) => (
                      <div key={si} className={`flex gap-1.5 ${step.keyword === "And" || step.keyword === "But" ? "pl-4" : ""}`}>
                        <span className={`shrink-0 font-semibold ${keywordColor[step.keyword] || "text-muted-foreground"}`}>
                          {step.keyword}
                        </span>
                        <span className="text-foreground/60">{step.text}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </button>
          );
        })}

        <div className="flex items-center gap-2 pt-1">
          <Button
            size="sm"
            onClick={handleApplySelected}
            disabled={selectedCount === 0 || applying}
          >
            <Check className="size-3.5" />
            {t("ai.applySelected")} {selectedCount > 0 && `(${selectedCount})`}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={handleRegenerate}
            disabled={regenerateCooldown}
          >
            <RefreshCw className="size-3.5" />
            {t("ai.regenerateProposal")}
          </Button>
          <Button size="sm" variant="ghost" onClick={handleCancel}>
            <X className="size-3.5" />
            {t("ai.cancel")}
          </Button>
        </div>
      </div>
    );
  }

  // --- Error state ---
  if (state === "error") {
    return (
      <div className="mt-3 flex items-center gap-2" aria-live="polite">
        <span className="text-xs text-destructive">{errorMessage}</span>
        <Button size="sm" variant="ghost" onClick={startGeneration}>
          <RefreshCw className="size-3.5" />
          {t("toast.retryAction")}
        </Button>
        <Button size="sm" variant="ghost" onClick={handleCancel}>
          <X className="size-3.5" />
          {t("ai.cancel")}
        </Button>
      </div>
    );
  }

  // --- Idle state: no proposal ---
  if (!hasProposal) {
    return (
      <div className="mt-3 flex items-center gap-2">
        <Button size="sm" variant="ghost" disabled className="opacity-50">
          <Sparkles className="size-3.5" />
          {t("ai.generateAC")}
        </Button>
        <span className="text-xs text-muted-foreground">{t("ai.writeProposalFirst")}</span>
      </div>
    );
  }

  // --- Idle state: has existing AC ---
  if (hasExistingAC) {
    return (
      <Button size="sm" variant="ghost" onClick={startGeneration}>
        <Sparkles className="size-3.5" />
        {t("ai.addMoreAC")}
      </Button>
    );
  }

  // --- Idle state: no AC yet, has proposal ---
  return (
    <div className="rounded-lg border border-dashed border-primary/30 bg-primary/5 p-4">
      <div className="flex items-center gap-2 mb-2">
        <Sparkles className="size-4 text-primary/70" />
        <span className="text-sm font-medium text-foreground">{t("ai.generateAC")}</span>
      </div>
      <Button size="sm" onClick={startGeneration}>
        <Sparkles className="size-3.5" />
        {t("ai.generateAC")}
      </Button>
    </div>
  );
}
