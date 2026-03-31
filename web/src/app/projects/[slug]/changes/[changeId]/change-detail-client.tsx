"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, useSearchParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";
import { workflowClient } from "@/lib/workflow";
import { projectClient } from "@/lib/project";
import { isCanonicalProjectRef, toProjectPath } from "@/lib/project-ref";
import { showError } from "@/lib/toast";
import { loadActivities, type ActivityItem } from "@/lib/ai";
import { Archive, ArchiveRestore, ArrowRight, FileText, ListChecks, MessageSquare, Plus, Trash2, Zap } from "lucide-react";
import { Header } from "@/components/layout/header";
import { DocumentTab } from "@/components/change/document-tab";
import { StructuredProposal } from "@/components/change/structured-proposal";
import { TaskBoard } from "@/components/task/task-board";
import { useEvents } from "@/lib/events";
import { AIPanelProvider } from "@/components/ai/ai-panel-context";
import { AISidePanel, AIPanelToggle } from "@/components/ai/ai-side-panel";

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
  createdAt?: { seconds: bigint };
}

const stageConfig: Record<
  string,
  { label: string; color: string; activeColor: string; icon: string }
> = {
  draft: {
    label: "Draft",
    color: "text-yellow-400",
    activeColor: "border-yellow-400 bg-yellow-400/10",
    icon: "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z",
  },
  spec: {
    label: "Spec",
    color: "text-blue-400",
    activeColor: "border-blue-400 bg-blue-400/10",
    icon: "M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm0 8a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6z",
  },
  approved: {
    label: "Approved",
    color: "text-emerald-400",
    activeColor: "border-emerald-400 bg-emerald-400/10",
    icon: "M5 13l4 4L19 7",
  },
};

const stages = ["draft", "spec", "approved"];

type TabId = "proposal" | "spec" | "tasks" | "history";

const tabI18nKeys: Record<TabId, string> = {
  proposal: "change.proposal",
  spec: "change.spec",
  tasks: "change.tasks",
  history: "change.history",
};

const validTabs: TabId[] = ["proposal", "spec", "tasks", "history"];

function activityIcon(type: string) {
  if (type === "stage") return Zap;
  if (type.startsWith("task")) return ListChecks;
  if (type.startsWith("doc")) return FileText;
  if (type === "ac_created") return ListChecks;
  if (type === "comment") return MessageSquare;
  return ArrowRight;
}

function activityColor(type: string): string {
  if (type === "stage") return "bg-primary/10 text-primary";
  if (type.startsWith("task")) return "bg-blue-500/10 text-blue-400";
  if (type.startsWith("doc")) return "bg-amber-500/10 text-amber-400";
  if (type === "ac_created") return "bg-emerald-500/10 text-emerald-400";
  if (type === "comment") return "bg-purple-500/10 text-purple-400";
  return "bg-muted text-muted-foreground";
}

function activityLabel(type: string, t: (key: string) => string): string {
  switch (type) {
    case "stage": return t("change.activityStage");
    case "task_created": return t("change.activityTaskCreated");
    case "task_done": return t("change.activityTaskDone");
    case "task_todo": return t("change.activityTaskCreated");
    case "task_in_progress": return t("change.activityTaskInProgress");
    case "doc_proposal": return t("change.activityProposalUpdated");
    case "doc_spec": return t("change.activitySpecUpdated");
    case "ac_created": return t("change.activityACCreated");
    case "comment": return t("change.activityComment");
    default: return type;
  }
}

export default function ChangeDetailClient() {
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  const projectRef = params.slug as string;
  const changeId = BigInt(params.changeId as string);
  const { t } = useI18n();
  const { on } = useEvents();
  const searchQuery = searchParams.toString();

  const tabParam = searchParams.get("tab") as TabId | null;
  const initialTab = tabParam && validTabs.includes(tabParam) ? tabParam : "proposal";
  const [activeTab, setActiveTabState] = useState<TabId>(initialTab);

  useEffect(() => {
    const nextTab = tabParam && validTabs.includes(tabParam) ? tabParam : "proposal";
    setActiveTabState(nextTab);
  }, [tabParam]);

  const setActiveTab = useCallback(
    (tab: TabId) => {
      setActiveTabState(tab);
      const url = new URL(window.location.href);
      url.searchParams.set("tab", tab);
      router.replace(url.pathname + url.search, { scroll: false });
    },
    [router],
  );
  const [stage, setStage] = useState("");
  const [conditions, setConditions] = useState<GateCondition[]>([]);
  const [history, setHistory] = useState<WorkflowEvent[]>([]);
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [animatingFrom, setAnimatingFrom] = useState<number | null>(null);
  const [showConfirmAdvance, setShowConfirmAdvance] = useState(false);
  const [members, setMembers] = useState<
    Array<{ userId: bigint; userName: string; userEmail?: string }>
  >([]);
  const [projectId, setProjectId] = useState<bigint>(BigInt(0));
  const [projectName, setProjectName] = useState("");
  const [projectSlug, setProjectSlug] = useState("");
  const [changeName, setChangeName] = useState("");
  const [changeIdentifier, setChangeIdentifier] = useState("");
  const [archivedAt, setArchivedAt] = useState<{ seconds: bigint; nanos: number } | undefined>(
    undefined,
  );
  const [archiving, setArchiving] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [changeLabels, setChangeLabels] = useState<
    Array<{ id: bigint; name: string; color: string }>
  >([]);
  const [orgLabels, setOrgLabels] = useState<Array<{ id: bigint; name: string; color: string }>>(
    [],
  );
  const [labelDropdownOpen, setLabelDropdownOpen] = useState(false);
  const [creatingLabel, setCreatingLabel] = useState(false);
  const [newLabelName, setNewLabelName] = useState("");
  const [newLabelColor, setNewLabelColor] = useState("#6B7280");
  const [showColorPicker, setShowColorPicker] = useState(false);
  const [pickerHue, setPickerHue] = useState(0);
  const [hexInput, setHexInput] = useState("");

  const prevStageRef = useRef(stage);

  const loadAll = useCallback(async () => {
    try {
      const projectRes = await projectClient.getProject({ slug: projectRef });
      const pid = projectRes.project!.id;
      if (!isCanonicalProjectRef(projectRef, projectRes.project!)) {
        const params = new URLSearchParams(searchQuery);
        const nextQuery = params.toString();
        router.replace(
          `${toProjectPath(projectRes.project!)}/changes/${changeId}${nextQuery ? `?${nextQuery}` : ""}`,
        );
        return;
      }

      setProjectId(pid);
      setProjectName(projectRes.project!.name);
      setProjectSlug(projectRes.project!.slug);
      setMembers(
        (projectRes.members || []).map((m) => ({
          userId: m.userId,
          userName: m.userName,
          userEmail: m.userEmail,
        })),
      );
      const [statusRes, historyRes, changeRes] = await Promise.all([
        workflowClient.getStatus({ changeId, projectId: pid }),
        workflowClient.getHistory({ changeId, projectId: pid }),
        projectClient.getChange({ id: changeId, projectId: pid }),
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
          createdAt: e.createdAt ? { seconds: e.createdAt.seconds } : undefined,
        })),
      );
      setChangeName(changeRes.change?.name ?? "");
      setChangeIdentifier(changeRes.change?.identifier ?? "");
      setArchivedAt(changeRes.change?.archivedAt);
      setChangeLabels(
        (changeRes.change?.labels ?? []).map((l) => ({ id: l.id, name: l.name, color: l.color })),
      );

      // Load org labels for the label picker
      projectClient
        .listLabels({})
        .then((res) => {
          setOrgLabels(res.labels.map((l) => ({ id: l.id, name: l.name, color: l.color })));
        })
        .catch(() => {});
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    } finally {
      setLoading(false);
    }
  }, [projectRef, changeId, t, router, searchQuery]);

  async function handleArchive() {
    setArchiving(true);
    try {
      await projectClient.archiveChange({ changeId, projectId });
      await loadAll();
    } catch (err) {
      showError(t("toast.archiveFailed"), err);
    } finally {
      setArchiving(false);
    }
  }

  async function handleAssignLabel(labelId: bigint) {
    try {
      await projectClient.assignChangeLabel({ changeId, labelId, projectId });
      const label = orgLabels.find((l) => l.id === labelId);
      if (label) {
        setChangeLabels((prev) => [...prev, label]);
      }
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    }
  }

  async function handleCreateAndAssignLabel() {
    const name = newLabelName.trim();
    if (!name) return;
    try {
      const res = await projectClient.createLabel({ name, color: newLabelColor });
      if (res.label) {
        const newLabel = { id: res.label.id, name: res.label.name, color: res.label.color };
        setOrgLabels((prev) => [...prev, newLabel]);
        await projectClient.assignChangeLabel({ changeId, labelId: newLabel.id, projectId });
        setChangeLabels((prev) => [...prev, newLabel]);
      }
      setNewLabelName("");
      setNewLabelColor("#6B7280");
      setCreatingLabel(false);
      setLabelDropdownOpen(false);
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    }
  }

  async function handleRemoveLabel(labelId: bigint) {
    try {
      await projectClient.removeChangeLabel({ changeId, labelId, projectId });
      setChangeLabels((prev) => prev.filter((l) => l.id !== labelId));
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    }
  }

  async function handleDeleteLabel(labelId: bigint) {
    try {
      await projectClient.deleteLabel({ id: labelId });
      setOrgLabels((prev) => prev.filter((l) => l.id !== labelId));
      setChangeLabels((prev) => prev.filter((l) => l.id !== labelId));
    } catch (err) {
      showError(t("toast.deleteLabelFailed"), err);
    }
  }

  async function handleUnarchive() {
    setArchiving(true);
    try {
      await projectClient.unarchiveChange({ changeId, projectId });
      await loadAll();
    } catch (err) {
      showError(t("toast.restoreFailed"), err);
    } finally {
      setArchiving(false);
    }
  }

  async function handleDelete() {
    setDeleting(true);
    try {
      await projectClient.deleteChange({ id: changeId, projectId });
      router.push(toProjectPath({ id: projectId, slug: projectSlug }));
    } catch (err) {
      showError(t("toast.deleteFailed"), err);
    } finally {
      setDeleting(false);
      setShowDeleteConfirm(false);
    }
  }

  async function handleRename(newName: string) {
    try {
      await projectClient.updateChange({ id: changeId, projectId, name: newName });
      setChangeName(newName);
    } catch (err) {
      showError(t("toast.renameFailed"), err);
    }
  }

  // Close label dropdown on click outside (ref-based)
  const labelDropdownRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!labelDropdownOpen) return;
    function handleClick(e: MouseEvent) {
      if (labelDropdownRef.current && !labelDropdownRef.current.contains(e.target as Node)) {
        setLabelDropdownOpen(false);
        setCreatingLabel(false);
        setNewLabelName("");
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [labelDropdownOpen]);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  // Load activities when history tab is active
  useEffect(() => {
    if (activeTab !== "history") return;
    loadActivities(changeId).then(setActivities).catch(() => {});
  }, [activeTab, changeId]);

  useEffect(() => {
    return on((event) => {
      if (event.changeId !== changeId) return;
      if (
        event.type === "change_updated" ||
        event.type === "task_created" ||
        event.type === "task_updated"
      ) {
        loadAll();
      }
    });
  }, [on, changeId, loadAll]);

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
    await workflowClient.advance({ changeId, projectId });
    loadAll();
  }

  // Mobile: show prev/current/next only
  function getMobileVisibleStages(): { index: number; stage: string }[] {
    if (currentIdx < 0) return [{ index: 0, stage: stages[0] }];
    const visible: { index: number; stage: string }[] = [];
    if (currentIdx > 0) visible.push({ index: currentIdx - 1, stage: stages[currentIdx - 1] });
    visible.push({ index: currentIdx, stage: stages[currentIdx] });
    if (currentIdx < stages.length - 1)
      visible.push({ index: currentIdx + 1, stage: stages[currentIdx + 1] });
    return visible;
  }

  return (
    <AIPanelProvider>
    <div className="flex h-dvh flex-col">
      <Header
        breadcrumbs={[
          { label: projectName, href: toProjectPath({ id: projectId, slug: projectSlug }) },
          {
            label: changeIdentifier ? `${changeIdentifier} ${changeName}` : changeName,
            editable: true,
            editablePrefix: changeIdentifier || undefined,
            editableValue: changeName,
            onSave: handleRename,
          },
        ]}
        actions={
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="cursor-pointer rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
            title={t("change.delete")}
          >
            <Trash2 className="h-4 w-4" />
          </button>
        }
      />

      {/* Main Content + AI Side Panel */}
      <div className="relative flex min-h-0 flex-1">
        <div className="min-w-0 flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl px-6 py-6">
          <div className="-mx-6 overflow-x-clip px-6 pb-2">
            {/* Stage Progress — Desktop */}
            <div className="mb-4 hidden pt-2 md:block">
              <div className="flex items-start">
                {stages.map((s, i) => {
                  const cfg = stageConfig[s];
                  const isActive = i === currentIdx;
                  const isPast = i < currentIdx;
                  return (
                    <div key={s} className="contents">
                      {/* Stage icon */}
                      <div className="relative flex shrink-0 flex-col items-center">
                        <div className="relative flex items-center justify-center">
                          {isActive && (
                            <>
                              <div
                                className="animate-stepper-ripple absolute h-9 w-9 rounded-full"
                                style={{
                                  background:
                                    "radial-gradient(circle, transparent 40%, var(--color-primary) 60%, transparent 75%)",
                                }}
                              />
                              <div className="animate-stepper-glow absolute h-10 w-10 rounded-full bg-primary/20 blur-md" />
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
                            <div
                              className="animate-stepper-dots-flow h-full rounded-full"
                              style={{
                                backgroundImage:
                                  "repeating-linear-gradient(90deg, var(--color-primary) 0, var(--color-primary) 4px, transparent 4px, transparent 12px)",
                                backgroundSize: "24px 2px",
                              }}
                            />
                          ) : (
                            <div
                              className={`h-full rounded-full transition-colors duration-500 ${
                                isPast ? "bg-emerald-500/50" : ""
                              }`}
                              style={
                                isPast
                                  ? undefined
                                  : {
                                      backgroundImage:
                                        "repeating-linear-gradient(90deg, var(--color-border) 0, var(--color-border) 6px, transparent 6px, transparent 12px)",
                                    }
                              }
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
            <div className="mb-4 pt-2 md:hidden">
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
                              style={{
                                background:
                                  "radial-gradient(circle, transparent 40%, var(--color-primary) 60%, transparent 75%)",
                              }}
                            />
                            <div className="animate-stepper-glow absolute h-10 w-10 rounded-full bg-primary/20 blur-md" />
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

            {/* Gate Conditions + Advance */}
            <div className="mb-4 rounded-lg border border-border/50 p-4">
              {/* Archived banner */}
              {archivedAt && (
                <div className="mb-3 flex items-center gap-2 rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2">
                  <Archive className="h-4 w-4 text-amber-400 shrink-0" />
                  <span className="text-sm text-amber-400">{t("change.archived")}</span>
                </div>
              )}
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex flex-1 flex-wrap items-center gap-2">
                  {conditions.map((c) => (
                    <div
                      key={c.name}
                      className="flex items-center gap-1.5 rounded-md border border-border/50 px-2.5 py-1"
                    >
                      {c.met ? (
                        <svg
                          className="h-3.5 w-3.5 text-emerald-400"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M5 13l4 4L19 7"
                          />
                        </svg>
                      ) : (
                        <svg
                          className="h-3.5 w-3.5 text-muted-foreground"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M12 8v4m0 4h.01"
                          />
                        </svg>
                      )}
                      <span
                        className={`text-xs ${c.met ? "text-foreground" : "text-muted-foreground"}`}
                      >
                        {c.description}
                      </span>
                    </div>
                  ))}
                  {conditions.length === 0 && (
                    <span className="text-xs text-muted-foreground">
                      {t("change.noGateConditions")}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {stage !== "approved" && !archivedAt && (
                    <Button
                      onClick={handleAdvance}
                      size="sm"
                      className="cursor-pointer"
                      disabled={!!archivedAt}
                    >
                      {t("change.advanceTo", { stage: t(`stages.${stages[currentIdx + 1]}`) })}
                    </Button>
                  )}
                  {stage === "approved" && !archivedAt && (
                    <Button
                      onClick={handleArchive}
                      size="sm"
                      variant="outline"
                      className="cursor-pointer"
                      disabled={archiving}
                    >
                      <Archive className="mr-1.5 h-3.5 w-3.5" />
                      {t("change.archive")}
                    </Button>
                  )}
                  {archivedAt && (
                    <Button
                      onClick={handleUnarchive}
                      size="sm"
                      variant="outline"
                      className="cursor-pointer"
                      disabled={archiving}
                    >
                      <ArchiveRestore className="mr-1.5 h-3.5 w-3.5" />
                      {t("change.restore")}
                    </Button>
                  )}
                </div>
              </div>

              {/* Confirm dialog for advancing with unmet gates */}
              {showConfirmAdvance && (
                <div className="mt-3 rounded-md border border-yellow-500/30 bg-yellow-500/5 p-3">
                  <p className="text-sm text-yellow-400">{t("change.advanceGateWarning")}</p>
                  <div className="mt-2 flex gap-2">
                    <Button
                      onClick={doAdvance}
                      size="sm"
                      variant="outline"
                      className="cursor-pointer"
                    >
                      {t("change.advanceConfirm")}
                    </Button>
                    <Button
                      onClick={() => setShowConfirmAdvance(false)}
                      size="sm"
                      variant="ghost"
                      className="cursor-pointer"
                    >
                      {t("common.cancel")}
                    </Button>
                  </div>
                </div>
              )}

              {/* Delete Confirm */}
              {showDeleteConfirm && (
                <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 p-3">
                  <p className="text-sm text-destructive">{t("change.deleteConfirm")}</p>
                  <div className="mt-2 flex gap-2">
                    <Button
                      onClick={handleDelete}
                      size="sm"
                      variant="destructive"
                      className="cursor-pointer"
                      disabled={deleting}
                    >
                      {t("change.deleteConfirmButton")}
                    </Button>
                    <Button
                      onClick={() => setShowDeleteConfirm(false)}
                      size="sm"
                      variant="ghost"
                      className="cursor-pointer"
                    >
                      {t("common.cancel")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Labels */}
          {!loading && (
            <div className="mb-4 flex flex-wrap items-center gap-2">
              {changeLabels.map((label) => (
                <span
                  key={String(label.id)}
                  className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
                  style={{
                    backgroundColor: `${label.color}18`,
                    color: label.color,
                  }}
                >
                  <span
                    className="h-1.5 w-1.5 rounded-full"
                    style={{ backgroundColor: label.color }}
                  />
                  {label.name}
                  <button
                    onClick={() => handleRemoveLabel(label.id)}
                    className="ml-0.5 cursor-pointer opacity-60 hover:opacity-100 transition-opacity"
                    title={t("change.removeLabel")}
                  >
                    ✕
                  </button>
                </span>
              ))}
              <div className="relative" ref={labelDropdownRef}>
                <button
                  onClick={() => setLabelDropdownOpen((v) => !v)}
                  className="cursor-pointer inline-flex items-center gap-1 rounded-full border border-dashed border-border/50 px-2 py-1 text-xs text-muted-foreground transition-colors hover:border-border hover:text-foreground"
                >
                  + {t("change.addLabel")}
                </button>
                {labelDropdownOpen && (
                  <div className="absolute left-0 top-full z-30 mt-1 min-w-[180px] rounded-lg border border-border/40 bg-popover p-1 shadow-lg">
                    {orgLabels.filter((ol) => !changeLabels.some((cl) => cl.id === ol.id))
                      .length === 0 ? (
                      <div className="px-3 py-2 text-xs text-muted-foreground">
                        {changeLabels.length === 0
                          ? "No labels in organization"
                          : "All labels assigned"}
                      </div>
                    ) : (
                      orgLabels
                        .filter((ol) => !changeLabels.some((cl) => cl.id === ol.id))
                        .map((label) => (
                          <div
                            key={String(label.id)}
                            className="group/label flex items-center rounded-md transition-colors hover:bg-accent"
                          >
                            <button
                              onClick={() => {
                                handleAssignLabel(label.id);
                                setLabelDropdownOpen(false);
                              }}
                              className="flex flex-1 cursor-pointer items-center gap-2 px-3 py-1.5 text-left text-xs"
                            >
                              <span
                                className="h-2 w-2 shrink-0 rounded-full"
                                style={{ backgroundColor: label.color }}
                              />
                              {label.name}
                            </button>
                            <button
                              onClick={() => handleDeleteLabel(label.id)}
                              className="cursor-pointer px-2 py-1.5 text-muted-foreground/0 transition-colors group-hover/label:text-muted-foreground hover:!text-destructive"
                              title={t("change.deleteLabel")}
                            >
                              <Trash2 className="size-3" />
                            </button>
                          </div>
                        ))
                    )}
                    {/* Create new label */}
                    <div className="border-t border-border/30 mt-1 pt-1">
                      {creatingLabel ? (
                        <div className="px-2 py-1.5 space-y-2">
                          <input
                            type="text"
                            value={newLabelName}
                            onChange={(e) => setNewLabelName(e.target.value)}
                            placeholder={t("change.addLabel")}
                            autoFocus
                            className="w-full rounded-md border border-border/40 bg-transparent px-2 py-1 text-xs text-foreground placeholder:text-muted-foreground/50 focus:border-primary/50 focus:outline-none"
                            onKeyDown={(e) => {
                              if (e.key === "Enter") handleCreateAndAssignLabel();
                              if (e.key === "Escape") {
                                setCreatingLabel(false);
                                setNewLabelName("");
                              }
                            }}
                          />
                          <div className="flex items-center gap-1.5">
                            {[
                              "#EF4444",
                              "#F59E0B",
                              "#10B981",
                              "#3B82F6",
                              "#8B5CF6",
                              "#EC4899",
                              "#6B7280",
                            ].map((c) => (
                              <button
                                key={c}
                                onClick={() => {
                                  setNewLabelColor(c);
                                  setShowColorPicker(false);
                                }}
                                className={`h-4 w-4 rounded-full cursor-pointer transition-transform ${newLabelColor === c ? "ring-2 ring-primary ring-offset-1 ring-offset-background scale-110" : "hover:scale-110"}`}
                                style={{ backgroundColor: c }}
                              />
                            ))}
                            <button
                              type="button"
                              onClick={() => {
                                setShowColorPicker(!showColorPicker);
                                setHexInput(newLabelColor);
                              }}
                              className={`h-4 w-4 rounded-full cursor-pointer transition-transform hover:scale-110 ${showColorPicker || !["#EF4444", "#F59E0B", "#10B981", "#3B82F6", "#8B5CF6", "#EC4899", "#6B7280"].includes(newLabelColor) ? "ring-2 ring-primary ring-offset-1 ring-offset-background scale-110" : ""}`}
                              style={{
                                background: ![
                                  "#EF4444",
                                  "#F59E0B",
                                  "#10B981",
                                  "#3B82F6",
                                  "#8B5CF6",
                                  "#EC4899",
                                  "#6B7280",
                                ].includes(newLabelColor)
                                  ? newLabelColor
                                  : "conic-gradient(red, yellow, lime, aqua, blue, magenta, red)",
                              }}
                            />
                          </div>
                          {showColorPicker && (
                            <div className="space-y-2 rounded-md border border-border/30 bg-accent/30 p-2">
                              <div
                                className="relative h-24 w-full cursor-crosshair rounded"
                                style={{
                                  background: `linear-gradient(to right, #fff, hsl(${pickerHue}, 100%, 50%))`,
                                }}
                                onClick={(e) => {
                                  const rect = e.currentTarget.getBoundingClientRect();
                                  const x = Math.max(
                                    0,
                                    Math.min(1, (e.clientX - rect.left) / rect.width),
                                  );
                                  const y = Math.max(
                                    0,
                                    Math.min(1, (e.clientY - rect.top) / rect.height),
                                  );
                                  const s = x * 100;
                                  const l = 100 - y * (50 + x * 50);
                                  const c = document.createElement("canvas");
                                  c.width = 1;
                                  c.height = 1;
                                  const ctx = c.getContext("2d")!;
                                  ctx.fillStyle = `hsl(${pickerHue}, ${s}%, ${l}%)`;
                                  ctx.fillRect(0, 0, 1, 1);
                                  const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data;
                                  const hex =
                                    `#${r.toString(16).padStart(2, "0")}${g.toString(16).padStart(2, "0")}${b.toString(16).padStart(2, "0")}`.toUpperCase();
                                  setNewLabelColor(hex);
                                  setHexInput(hex);
                                }}
                              >
                                <div
                                  className="pointer-events-none absolute inset-0 rounded"
                                  style={{
                                    background: "linear-gradient(to bottom, transparent, #000)",
                                  }}
                                />
                              </div>
                              <input
                                type="range"
                                min={0}
                                max={360}
                                value={pickerHue}
                                onChange={(e) => setPickerHue(Number(e.target.value))}
                                className="h-2 w-full cursor-pointer appearance-none rounded-full [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:shadow-md"
                                style={{
                                  background:
                                    "linear-gradient(to right, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00)",
                                }}
                              />
                              <div className="flex items-center gap-1.5">
                                <div
                                  className="h-5 w-5 shrink-0 rounded border border-border/40"
                                  style={{ backgroundColor: newLabelColor }}
                                />
                                <input
                                  type="text"
                                  value={hexInput}
                                  onChange={(e) => {
                                    const v = e.target.value;
                                    setHexInput(v);
                                    if (/^#[0-9A-Fa-f]{6}$/.test(v))
                                      setNewLabelColor(v.toUpperCase());
                                  }}
                                  placeholder="#000000"
                                  className="w-full rounded border border-border/40 bg-transparent px-1.5 py-0.5 font-mono text-[10px] text-foreground focus:border-primary/50 focus:outline-none"
                                />
                              </div>
                            </div>
                          )}
                          <div className="flex gap-1">
                            <button
                              onClick={() => handleCreateAndAssignLabel()}
                              disabled={!newLabelName.trim()}
                              className="cursor-pointer flex-1 rounded-md bg-primary px-2 py-1 text-[10px] font-medium text-primary-foreground disabled:opacity-40 transition-colors"
                            >
                              {t("common.create")}
                            </button>
                            <button
                              onClick={() => {
                                setCreatingLabel(false);
                                setNewLabelName("");
                              }}
                              className="cursor-pointer rounded-md px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
                            >
                              {t("common.cancel")}
                            </button>
                          </div>
                        </div>
                      ) : (
                        <button
                          onClick={() => setCreatingLabel(true)}
                          className="flex w-full cursor-pointer items-center gap-2 rounded-md px-3 py-1.5 text-left text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                        >
                          <Plus className="size-3" />
                          {t("change.newLabel")}
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Tab Navigation — sticky */}
          <div className="sticky top-0 z-20 -mx-6 mb-4 border-b border-border/40 bg-background/95 px-6 backdrop-blur supports-[backdrop-filter]:bg-background/80">
            <div className="flex gap-1 overflow-x-auto">
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
          </div>

          {/* Tab Content */}
          {activeTab === "proposal" && projectId > 0 && (
            <StructuredProposal
              changeId={changeId}
              projectId={projectId}
              currentStage={stage}
              members={members}
            />
          )}
          {activeTab === "spec" && projectId > 0 && (
            <DocumentTab
              changeId={changeId}
              projectId={projectId}
              docType="spec"
              members={members}
            />
          )}
          {activeTab === "tasks" && projectId > 0 && (
            <div className="min-h-0 max-h-[calc(100svh-20rem)] overflow-y-auto pr-1 md:max-h-[calc(100svh-18rem)]">
              <TaskBoard changeId={changeId} projectId={projectId} members={members} />
            </div>
          )}
          {activeTab === "history" && (
            <div className="py-2">
              {activities.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("change.noEvents")}</p>
              ) : (
                <ul className="space-y-0">
                  {activities.map((item, i) => {
                    const Icon = activityIcon(item.type);
                    const color = activityColor(item.type);
                    const label = activityLabel(item.type, t);
                    const time = item.createdAt ? new Date(item.createdAt).toLocaleString() : "";
                    return (
                      <li key={`${item.type}-${i}`} className="relative pl-10 pb-6 last:pb-0">
                        {/* Timeline line */}
                        {i < activities.length - 1 && (
                          <div className="absolute left-[15px] top-8 bottom-0 w-px bg-border/50" />
                        )}
                        {/* Icon */}
                        <div className={`absolute left-0 top-0.5 flex size-8 items-center justify-center rounded-full ${color}`}>
                          <Icon className="size-4" />
                        </div>
                        {/* Content */}
                        <div className="pt-0.5">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-sm font-medium text-foreground">{label}</span>
                            {item.userName && (
                              <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{item.userName}</span>
                            )}
                          </div>
                          <p className="mt-1 text-sm text-foreground/80">{item.title}</p>
                          {item.detail && (
                            <p className="mt-0.5 text-sm text-muted-foreground">{item.detail}</p>
                          )}
                          {time && (
                            <p className="mt-1 text-xs text-muted-foreground/50">{time}</p>
                          )}
                        </div>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
          )}
        </div>
        </div>
        <AISidePanel changeId={changeId} projectId={projectId} />
        <AIPanelToggle />
      </div>
    </div>
    </AIPanelProvider>
  );
}
