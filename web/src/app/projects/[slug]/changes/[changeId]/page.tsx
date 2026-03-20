"use client";

import { useEffect, useState, useRef } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";
import { workflowClient } from "@/lib/workflow";
import { DocumentTab } from "@/components/change/document-tab";

interface GateCondition {
  name: string;
  description: string;
  met: boolean;
}

interface WorkflowEvent {
  id: bigint;
  fromStage: string;
  toStage: string;
  action: string;
  reason: string;
  userName: string;
}

const stageConfig: Record<string, { label: string; color: string; activeColor: string; icon: string }> = {
  draft: {
    label: "Draft",
    color: "text-yellow-400",
    activeColor: "border-yellow-400 bg-yellow-400/10",
    icon: "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z",
  },
  design: {
    label: "Design",
    color: "text-blue-400",
    activeColor: "border-blue-400 bg-blue-400/10",
    icon: "M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm0 8a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6z",
  },
  review: {
    label: "Review",
    color: "text-purple-400",
    activeColor: "border-purple-400 bg-purple-400/10",
    icon: "M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z",
  },
  ready: {
    label: "Ready",
    color: "text-emerald-400",
    activeColor: "border-emerald-400 bg-emerald-400/10",
    icon: "M5 13l4 4L19 7",
  },
};

const stages = ["draft", "design", "review", "ready"];

type TabId = "proposal" | "design" | "specs" | "tasks" | "history";

const tabI18nKeys: Record<TabId, string> = {
  proposal: "change.proposal",
  design: "change.design",
  specs: "change.specs",
  tasks: "change.tasks",
  history: "change.history",
};

export default function ChangeDetailPage() {
  const params = useParams();
  const slug = params.slug as string;
  const changeId = BigInt(params.changeId as string);
  const { t } = useI18n();

  const [activeTab, setActiveTab] = useState<TabId>("proposal");
  const [stage, setStage] = useState("");
  const [conditions, setConditions] = useState<GateCondition[]>([]);
  const [history, setHistory] = useState<WorkflowEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [animatingFrom, setAnimatingFrom] = useState<number | null>(null);
  const [showConfirmAdvance, setShowConfirmAdvance] = useState(false);

  const prevStageRef = useRef(stage);

  async function loadAll() {
    try {
      const [statusRes, historyRes] = await Promise.all([
        workflowClient.getStatus({ changeId }),
        workflowClient.getHistory({ changeId }),
      ]);
      setStage(statusRes.stage);
      setConditions(
        statusRes.conditions.map((c) => ({ name: c.name, description: c.description, met: c.met })),
      );
      setHistory(
        historyRes.events.map((e) => ({
          id: e.id,
          fromStage: e.fromStage,
          toStage: e.toStage,
          action: e.action,
          reason: e.reason,
          userName: e.userName,
        })),
      );
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAll();
  }, []);

  // Trigger particle animation on stage change
  useEffect(() => {
    if (prevStageRef.current && prevStageRef.current !== stage) {
      const fromIdx = stages.indexOf(prevStageRef.current);
      if (fromIdx >= 0) {
        setAnimatingFrom(fromIdx);
        const timer = setTimeout(() => setAnimatingFrom(null), 700);
        return () => clearTimeout(timer);
      }
    }
    prevStageRef.current = stage;
  }, [stage]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const currentIdx = stages.indexOf(stage);
  const allGatesMet = conditions.every((c) => c.met);

  async function handleAdvance() {
    if (!allGatesMet) {
      setShowConfirmAdvance(true);
      return;
    }
    await doAdvance();
  }

  async function doAdvance() {
    setShowConfirmAdvance(false);
    await workflowClient.advance({ changeId });
    loadAll();
  }

  // Mobile: show prev/current/next only
  function getMobileVisibleStages(): { index: number; stage: string }[] {
    if (currentIdx < 0) return [{ index: 0, stage: stages[0] }];
    const visible: { index: number; stage: string }[] = [];
    if (currentIdx > 0) visible.push({ index: currentIdx - 1, stage: stages[currentIdx - 1] });
    visible.push({ index: currentIdx, stage: stages[currentIdx] });
    if (currentIdx < stages.length - 1) visible.push({ index: currentIdx + 1, stage: stages[currentIdx + 1] });
    return visible;
  }

  return (
    <div className="flex min-h-screen flex-col">
      {/* Breadcrumb Header */}
      <header className="border-b border-border/50 bg-background/80 backdrop-blur-md">
        <div className="flex items-center gap-2 px-6 py-2.5">
          <Link
            href={`/projects/${slug}`}
            className="text-sm text-muted-foreground hover:text-foreground transition-colors duration-200"
          >
            Project
          </Link>
          <svg className="h-3.5 w-3.5 text-muted-foreground/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.25 4.5l7.5 7.5-7.5 7.5" />
          </svg>
          <span className="text-sm font-medium">Change</span>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl px-6 py-6">
            {/* Stage Progress — Desktop */}
            <div className="mb-4 hidden md:block">
              <div className="flex items-start">
                {stages.map((s, i) => {
                  const cfg = stageConfig[s];
                  const isActive = i === currentIdx;
                  const isPast = i < currentIdx;
                  return (
                    <div key={s} className="contents">
                      {/* Stage icon */}
                      <div className="relative z-10 flex shrink-0 flex-col items-center">
                        <div className="relative flex items-center justify-center">
                          {isActive && (
                            <>
                              <div
                                className="animate-stepper-ripple absolute h-9 w-9 rounded-full"
                                style={{ background: "radial-gradient(circle, transparent 40%, var(--color-primary) 60%, transparent 75%)" }}
                              />
                              <div className="animate-stepper-glow absolute h-16 w-16 rounded-full bg-primary/30 blur-xl" />
                            </>
                          )}
                          <div
                            className={`relative flex h-9 w-9 items-center justify-center rounded-full border-2 bg-background transition-all duration-300 ${
                              isActive
                                ? cfg.activeColor
                                : isPast
                                  ? "border-emerald-500 bg-emerald-500/10"
                                  : "border-border bg-muted"
                            }`}
                          >
                            <svg
                              className={`h-4 w-4 ${isActive ? cfg.color : isPast ? "text-emerald-400" : "text-muted-foreground"}`}
                              fill="none"
                              stroke="currentColor"
                              viewBox="0 0 24 24"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d={isPast ? "M5 13l4 4L19 7" : cfg.icon}
                              />
                            </svg>
                          </div>
                        </div>
                        <span
                          className={`mt-1.5 text-[11px] font-medium ${isActive ? "text-foreground" : "text-muted-foreground"}`}
                        >
                          {t(`stages.${s}`)}
                        </span>
                      </div>
                      {/* Connection line */}
                      {i < stages.length - 1 && (
                        <div className="relative mx-3 mt-[18px] flex-1" style={{ height: "2px" }}>
                          {isPast && i === currentIdx - 1 ? (
                            /* Active segment: flowing dots from completed to current */
                            <div
                              className="animate-stepper-dots-flow h-full rounded-full"
                              style={{
                                backgroundImage: "repeating-linear-gradient(90deg, var(--color-primary) 0, var(--color-primary) 4px, transparent 4px, transparent 12px)",
                                backgroundSize: "24px 2px",
                              }}
                            />
                          ) : (
                            <div
                              className={`h-full rounded-full transition-colors duration-500 ${
                                isPast ? "bg-emerald-500/50" : ""
                              }`}
                              style={isPast ? undefined : {
                                backgroundImage: "repeating-linear-gradient(90deg, var(--color-border) 0, var(--color-border) 6px, transparent 6px, transparent 12px)",
                              }}
                            />
                          )}
                          {animatingFrom === i && (
                            <div className="animate-stepper-particle absolute top-1/2 h-2 w-2 -translate-y-1/2 rounded-full bg-primary shadow-[0_0_8px_var(--color-primary)]" />
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Stage Progress — Mobile */}
            <div className="mb-4 md:hidden">
              <div className="relative flex items-center justify-center gap-4">
                {getMobileVisibleStages().map(({ index: i, stage: s }) => {
                  const cfg = stageConfig[s];
                  const isActive = i === currentIdx;
                  const isPast = i < currentIdx;
                  return (
                    <div key={s} className="flex flex-col items-center">
                      <div className="relative flex items-center justify-center">
                        {isActive && (
                          <>
                            <div
                              className="animate-stepper-ripple absolute h-10 w-10 rounded-full"
                              style={{ background: "radial-gradient(circle, transparent 40%, var(--color-primary) 60%, transparent 75%)" }}
                            />
                            <div className="animate-stepper-glow absolute h-16 w-16 rounded-full bg-primary/30 blur-xl" />
                          </>
                        )}
                        <div
                          className={`relative flex items-center justify-center rounded-full border-2 transition-all duration-300 ${
                            isActive
                              ? `h-10 w-10 ${cfg.activeColor}`
                              : isPast
                                ? "h-8 w-8 border-emerald-500 bg-emerald-500/10"
                                : "h-8 w-8 border-border bg-muted"
                          }`}
                        >
                          <svg
                            className={`${isActive ? "h-4.5 w-4.5" : "h-3.5 w-3.5"} ${isActive ? cfg.color : isPast ? "text-emerald-400" : "text-muted-foreground"}`}
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d={isPast ? "M5 13l4 4L19 7" : cfg.icon}
                            />
                          </svg>
                        </div>
                      </div>
                      <span
                        className={`mt-1 text-[10px] font-medium ${isActive ? "text-foreground" : "text-muted-foreground"}`}
                      >
                        {t(`stages.${s}`)}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Gate Conditions + Advance (always visible below stepper) */}
            <div className="mb-6 rounded-lg border border-border/50 p-4">
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex flex-1 flex-wrap items-center gap-2">
                  {conditions.map((c) => (
                    <div
                      key={c.name}
                      className="flex items-center gap-1.5 rounded-md border border-border/50 px-2.5 py-1"
                    >
                      {c.met ? (
                        <svg className="h-3.5 w-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                      ) : (
                        <svg className="h-3.5 w-3.5 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01" />
                        </svg>
                      )}
                      <span className={`text-xs ${c.met ? "text-foreground" : "text-muted-foreground"}`}>{c.description}</span>
                    </div>
                  ))}
                  {conditions.length === 0 && (
                    <span className="text-xs text-muted-foreground">No gate conditions</span>
                  )}
                </div>
                {stage !== "ready" && (
                  <Button onClick={handleAdvance} size="sm" className="cursor-pointer">
                    Advance to {stageConfig[stages[currentIdx + 1]]?.label ?? "next"}
                  </Button>
                )}
              </div>

              {/* Confirm dialog for advancing with unmet gates */}
              {showConfirmAdvance && (
                <div className="mt-3 rounded-md border border-yellow-500/30 bg-yellow-500/5 p-3">
                  <p className="text-sm text-yellow-400">
                    Gate conditions are not fully met. Advance anyway?
                  </p>
                  <div className="mt-2 flex gap-2">
                    <Button onClick={doAdvance} size="sm" variant="outline" className="cursor-pointer">
                      Yes, advance
                    </Button>
                    <Button
                      onClick={() => setShowConfirmAdvance(false)}
                      size="sm"
                      variant="ghost"
                      className="cursor-pointer"
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              )}
            </div>

            {/* Tab Navigation */}
            <div className="mb-6 flex gap-1 overflow-x-auto border-b border-border/50">
              {(Object.keys(tabI18nKeys) as TabId[]).map((tabId) => (
                <button
                  key={tabId}
                  onClick={() => setActiveTab(tabId)}
                  className={`cursor-pointer whitespace-nowrap px-4 py-2.5 text-sm font-medium transition-colors duration-200 ${
                    activeTab === tabId
                      ? "border-b-2 border-primary text-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {t(tabI18nKeys[tabId])}
                </button>
              ))}
            </div>

            {/* Tab Content */}
            {activeTab === "proposal" && <DocumentTab changeId={changeId} docType="proposal" />}
            {activeTab === "design" && <DocumentTab changeId={changeId} docType="design" />}
            {activeTab === "specs" && <DocumentTab changeId={changeId} docType="spec" />}
            {activeTab === "tasks" && <DocumentTab changeId={changeId} docType="tasks" />}
            {activeTab === "history" && (
              <div className="space-y-4">
                {history.length === 0 ? (
                  <p className="text-sm text-muted-foreground">{t("change.noEvents")}</p>
                ) : (
                  <ul className="space-y-4">
                    {history.map((event) => (
                      <li key={String(event.id)} className="relative pl-5">
                        <div className="absolute left-0 top-1.5 h-2 w-2 rounded-full bg-primary/50" />
                        <div className="flex items-baseline gap-2">
                          <p className="text-sm font-medium">{event.action.replace("_", " ")}</p>
                          {event.userName && (
                            <span className="text-xs text-muted-foreground">by {event.userName}</span>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {event.fromStage} → {event.toStage}
                        </p>
                        {event.reason && (
                          <p className="mt-0.5 text-xs text-muted-foreground">{event.reason}</p>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </div>
        </div>
    </div>
  );
}

