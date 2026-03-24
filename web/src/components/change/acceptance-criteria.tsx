"use client";

import { useCallback, useEffect, useState } from "react";
import { acceptanceClient } from "@/lib/acceptance";
import { useI18n } from "@/lib/i18n";
import { Plus, Trash2, GripVertical, Check, X } from "lucide-react";
import { showError } from "@/lib/toast";
import { AIACGenerator } from "@/components/ai/ai-ac-generator";
import type { GeneratedAC } from "@/lib/ai";

interface Step {
  keyword: string;
  text: string;
}

interface ACItem {
  id: bigint;
  scenario: string;
  steps: Step[];
  met: boolean;
  sortOrder: number;
}

const KEYWORDS = ["Given", "When", "Then", "And", "But"] as const;
const keywordColor: Record<string, string> = {
  Given: "text-blue-400",
  When: "text-amber-400",
  Then: "text-emerald-400",
  And: "text-purple-400",
  But: "text-red-400",
};

interface AcceptanceCriteriaProps {
  changeId: bigint;
  projectId: bigint;
  reviewMode?: boolean;
  hasProposal?: boolean;
}

export function AcceptanceCriteria({ changeId, projectId, reviewMode = false, hasProposal = false }: AcceptanceCriteriaProps) {
  const { t } = useI18n();
  const [items, setItems] = useState<ACItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);

  const loadItems = useCallback(async () => {
    try {
      const res = await acceptanceClient.listAC({ changeId, projectId });
      setItems(
        res.criteria.map((c) => ({
          id: c.id,
          scenario: c.scenario,
          steps: c.steps.map((s) => ({ keyword: s.keyword, text: s.text })),
          met: c.met,
          sortOrder: c.sortOrder,
        })),
      );
    } catch (err) {
      showError(t("toast.acLoadFailed"), err);
    } finally {
      setLoading(false);
    }
  }, [changeId, projectId]);

  useEffect(() => {
    loadItems();
  }, [loadItems]);

  const handleCreate = async (scenario: string, steps: Step[]) => {
    try {
      await acceptanceClient.createAC({
        changeId,
        scenario,
        steps,
        sortOrder: items.length,
        projectId,
      });
      setAdding(false);
      loadItems();
    } catch (err) {
      showError(t("toast.acCreateFailed"), err);
    }
  };

  const handleUpdate = async (item: ACItem) => {
    try {
      await acceptanceClient.updateAC({
        id: item.id,
        scenario: item.scenario,
        steps: item.steps,
        sortOrder: item.sortOrder,
        projectId,
      });
      loadItems();
    } catch (err) {
      showError(t("toast.acUpdateFailed"), err);
    }
  };

  const handleDelete = async (id: bigint) => {
    try {
      await acceptanceClient.deleteAC({ id, projectId });
      loadItems();
    } catch (err) {
      showError(t("toast.acDeleteFailed"), err);
    }
  };

  const handleToggle = async (id: bigint, met: boolean) => {
    try {
      await acceptanceClient.toggleAC({ id, met, projectId });
      loadItems();
    } catch (err) {
      showError(t("toast.acToggleFailed"), err);
    }
  };

  const handleAIApply = async (acs: GeneratedAC[]) => {
    for (const ac of acs) {
      await acceptanceClient.createAC({
        changeId,
        scenario: ac.scenario,
        steps: ac.steps,
        sortOrder: items.length,
        projectId,
        testRef: "",
      });
    }
    loadItems();
  };

  const metCount = items.filter((i) => i.met).length;

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="mt-6">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold">{t("ac.title")}</h3>
          {items.length > 0 && (
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
              {metCount}/{items.length}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          {!reviewMode && items.length > 0 && (
            <AIACGenerator
              changeId={changeId}
              hasProposal={hasProposal}
              hasExistingAC={true}
              onApply={handleAIApply}
            />
          )}
          {!reviewMode && !adding && (
            <button
              onClick={() => setAdding(true)}
              className="flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-xs text-primary hover:bg-primary/10 transition-colors"
            >
              <Plus className="size-3.5" />
              {t("ac.add")}
            </button>
          )}
        </div>
      </div>

      {items.length > 0 && (
        <div className="mb-4 h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full rounded-full bg-emerald-500 transition-all duration-300"
            style={{ width: `${(metCount / items.length) * 100}%` }}
          />
        </div>
      )}

      <div className="space-y-2">
        {items.map((item) =>
          reviewMode ? (
            <ReviewACCard key={String(item.id)} item={item} onToggle={handleToggle} />
          ) : (
            <EditACCard key={String(item.id)} item={item} onUpdate={handleUpdate} onDelete={handleDelete} t={t} />
          ),
        )}
      </div>

      {adding && (
        <ACForm
          t={t}
          onSave={(scenario, steps) => handleCreate(scenario, steps)}
          onCancel={() => setAdding(false)}
        />
      )}

      {items.length === 0 && !adding && (
        <div className="rounded-lg border border-dashed border-border/50 py-6 text-center">
          <p className="text-sm text-muted-foreground">{t("ac.empty")}</p>
          {!reviewMode && (
            <>
              <button
                onClick={() => setAdding(true)}
                className="mt-2 cursor-pointer text-xs text-primary hover:text-primary/80 transition-colors"
              >
                {t("ac.addFirst")}
              </button>
              <div className="mt-3 flex justify-center">
                <AIACGenerator
                  changeId={changeId}
                  hasProposal={hasProposal}
                  hasExistingAC={false}
                  onApply={handleAIApply}
                />
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

// Shared form for creating/editing AC
function ACForm({
  t,
  initialScenario = "",
  initialSteps,
  onSave,
  onCancel,
}: {
  t: (key: string) => string;
  initialScenario?: string;
  initialSteps?: Step[];
  onSave: (scenario: string, steps: Step[]) => void;
  onCancel: () => void;
}) {
  const [scenario, setScenario] = useState(initialScenario);
  const [steps, setSteps] = useState<Step[]>(
    initialSteps || [
      { keyword: "Given", text: "" },
      { keyword: "When", text: "" },
      { keyword: "Then", text: "" },
    ],
  );

  const addStep = () => {
    // Default to And, user can change
    setSteps([...steps, { keyword: "And", text: "" }]);
  };

  const removeStep = (index: number) => {
    if (steps.length <= 1) return;
    setSteps(steps.filter((_, i) => i !== index));
  };

  const updateStep = (index: number, field: "keyword" | "text", value: string) => {
    setSteps(steps.map((s, i) => (i === index ? { ...s, [field]: value } : s)));
  };

  return (
    <div className="mt-2 rounded-lg border border-primary/30 bg-muted/30 p-3">
      <input
        value={scenario}
        onChange={(e) => setScenario(e.target.value)}
        placeholder={t("ac.scenarioPlaceholder")}
        className="mb-3 w-full rounded-md border border-border/50 bg-transparent px-2 py-1.5 text-sm font-medium outline-none focus:border-primary transition-colors"
        autoFocus
      />

      <div className="space-y-1.5">
        {steps.map((step, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <select
              value={step.keyword}
              onChange={(e) => updateStep(i, "keyword", e.target.value)}
              className={`w-20 shrink-0 cursor-pointer rounded-md border border-border/50 bg-transparent px-1.5 py-1.5 text-xs font-semibold outline-none focus:border-primary transition-colors ${keywordColor[step.keyword] || "text-muted-foreground"}`}
            >
              {KEYWORDS.map((kw) => (
                <option key={kw} value={kw}>{kw}</option>
              ))}
            </select>
            <input
              value={step.text}
              onChange={(e) => updateStep(i, "text", e.target.value)}
              placeholder={t("ac.stepPlaceholder")}
              className="min-w-0 flex-1 rounded-md border border-border/50 bg-transparent px-2 py-1.5 text-sm outline-none focus:border-primary transition-colors"
            />
            <button
              onClick={() => removeStep(i)}
              className="cursor-pointer rounded p-1 text-muted-foreground/40 hover:text-destructive transition-colors"
            >
              <X className="size-3.5" />
            </button>
          </div>
        ))}
      </div>

      <button
        onClick={addStep}
        className="mt-1.5 flex cursor-pointer items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
      >
        <Plus className="size-3" />
        {t("ac.addStep")}
      </button>

      <div className="mt-3 flex justify-end gap-1.5">
        <button
          onClick={onCancel}
          className="cursor-pointer rounded px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          {t("common.cancel")}
        </button>
        <button
          onClick={() => onSave(scenario, steps.filter((s) => s.text.trim()))}
          disabled={!scenario.trim() || steps.every((s) => !s.text.trim())}
          className="cursor-pointer rounded bg-primary px-2 py-1 text-xs text-primary-foreground disabled:opacity-50 transition-colors"
        >
          {t("common.save")}
        </button>
      </div>
    </div>
  );
}

function StepDisplay({ steps, strikethrough = false }: { steps: Step[]; strikethrough?: boolean }) {
  const isSubStep = (keyword: string) => keyword === "And" || keyword === "But";

  return (
    <div className={`text-xs ${strikethrough ? "text-muted-foreground line-through" : "text-foreground/70"}`}>
      {steps.map((step, i) => {
        const sub = isSubStep(step.keyword);
        const isLast = i === steps.length - 1;

        return (
          <div key={i} className={`flex ${sub ? "pl-5" : ""}`}>
            {/* Timeline track */}
            <div className="flex flex-col items-center mr-2.5">
              <div
                className={`shrink-0 rounded-full border-2 ${
                  sub
                    ? `h-2 w-2 ${strikethrough ? "border-muted-foreground/30" : "border-muted-foreground/40"}`
                    : `h-3 w-3 ${strikethrough ? "border-muted-foreground/30" : keywordColor[step.keyword]?.replace("text-", "border-") || "border-muted-foreground"}`
                }`}
              />
              {!isLast && (
                <div className={`w-px flex-1 min-h-3 ${strikethrough ? "bg-muted-foreground/15" : "bg-border"}`} />
              )}
            </div>
            {/* Content */}
            <div className="pb-2">
              <span className={`font-semibold ${keywordColor[step.keyword] || "text-muted-foreground"}`}>
                {step.keyword}
              </span>{" "}
              <span>{step.text}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function EditACCard({
  item,
  onUpdate,
  onDelete,
  t,
}: {
  item: ACItem;
  onUpdate: (item: ACItem) => void;
  onDelete: (id: bigint) => void;
  t: (key: string) => string;
}) {
  const [editing, setEditing] = useState(false);

  if (editing) {
    return (
      <ACForm
        t={t}
        initialScenario={item.scenario}
        initialSteps={item.steps}
        onSave={(scenario, steps) => {
          onUpdate({ ...item, scenario, steps });
          setEditing(false);
        }}
        onCancel={() => setEditing(false)}
      />
    );
  }

  return (
    <div className="group flex items-start gap-2 rounded-lg border border-border/50 p-3 transition-colors hover:border-border">
      <GripVertical className="mt-0.5 size-4 shrink-0 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab" />
      <div className="min-w-0 flex-1 cursor-pointer" onClick={() => setEditing(true)}>
        <p className="mb-1 text-sm font-medium">{item.scenario}</p>
        <StepDisplay steps={item.steps} />
      </div>
      <button
        onClick={() => onDelete(item.id)}
        className="cursor-pointer rounded p-1 text-muted-foreground/50 opacity-0 group-hover:opacity-100 hover:text-destructive transition-all"
      >
        <Trash2 className="size-3.5" />
      </button>
    </div>
  );
}

function ReviewACCard({
  item,
  onToggle,
}: {
  item: ACItem;
  onToggle: (id: bigint, met: boolean) => void;
}) {
  return (
    <div
      className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-all ${
        item.met ? "border-emerald-500/30 bg-emerald-500/5" : "border-border/50 hover:border-border"
      }`}
      onClick={() => onToggle(item.id, !item.met)}
    >
      <div
        className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border-2 transition-colors ${
          item.met ? "border-emerald-500 bg-emerald-500" : "border-muted-foreground/30"
        }`}
      >
        {item.met && <Check className="size-3 text-white" />}
      </div>
      <div className="min-w-0 flex-1">
        <p className={`mb-1 text-sm font-medium ${item.met ? "text-muted-foreground line-through" : ""}`}>
          {item.scenario}
        </p>
        <StepDisplay steps={item.steps} strikethrough={item.met} />
      </div>
    </div>
  );
}
