"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { ReadmeEditor } from "@/components/editor/readme-editor";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DatePicker } from "@/components/ui/date-picker";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Header } from "@/components/layout/header";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { projectClient } from "@/lib/project";
import { orgClient } from "@/lib/organization";
import { memoryClient } from "@/lib/memory";
import { useI18n } from "@/lib/i18n";
import { showError, showSuccess } from "@/lib/toast";
import { marked } from "marked";
import {
  Users,
  FileText,
  Pencil,
  UserPlus,
  Trash2,
  MoreHorizontal,
  Plus,
  ChevronRight,
  Layers,
  Brain,
  User,
  Signal,
} from "lucide-react";

const stageConfig: Record<string, { label: string; color: string; icon: string; glow: string }> = {
  draft: {
    label: "Draft",
    color: "bg-amber-500/10 text-amber-400 border-amber-500/20",
    icon: "M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10",
    glow: "shadow-amber-500/5",
  },
  design: {
    label: "Design",
    color: "bg-blue-500/10 text-blue-400 border-blue-500/20",
    icon: "M9.53 16.122a3 3 0 00-5.78 1.128 2.25 2.25 0 01-2.4 2.245 4.5 4.5 0 008.4-2.245c0-.399-.078-.78-.22-1.128zm0 0a15.998 15.998 0 003.388-1.62m-5.043-.025a15.994 15.994 0 011.622-3.395m3.42 3.42a15.995 15.995 0 004.764-4.648l3.876-5.814a1.151 1.151 0 00-1.597-1.597L14.146 6.32a15.996 15.996 0 00-4.649 4.763m3.42 3.42a6.776 6.776 0 00-3.42-3.42",
    glow: "shadow-blue-500/5",
  },
  review: {
    label: "Review",
    color: "bg-violet-500/10 text-violet-400 border-violet-500/20",
    icon: "M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178zM15 12a3 3 0 11-6 0 3 3 0 016 0z",
    glow: "shadow-violet-500/5",
  },
  ready: {
    label: "Ready",
    color: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
    icon: "M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z",
    glow: "shadow-emerald-500/5",
  },
};

const statusConfig: Record<string, { label: string; color: string; dotColor: string }> = {
  backlog: { label: "Backlog", color: "text-muted-foreground", dotColor: "bg-muted-foreground" },
  active: { label: "Active", color: "text-yellow-400", dotColor: "bg-yellow-400" },
  paused: { label: "Paused", color: "text-orange-400", dotColor: "bg-orange-400" },
  completed: { label: "Completed", color: "text-emerald-400", dotColor: "bg-emerald-400" },
  cancelled: { label: "Cancelled", color: "text-red-400", dotColor: "bg-red-400" },
};

const priorityConfig: Record<string, { label: string; icon: string }> = {
  urgent: { label: "Urgent", icon: "!!!" },
  high: { label: "High", icon: "!!" },
  medium: { label: "Medium", icon: "!" },
  low: { label: "Low", icon: "\u2014" },
  none: { label: "No priority", icon: "\u00B7\u00B7\u00B7" },
};

const healthConfig: Record<string, { label: string; dotColor: string }> = {
  on_track: { label: "On Track", dotColor: "bg-emerald-400" },
  at_risk: { label: "At Risk", dotColor: "bg-yellow-400" },
  off_track: { label: "Off Track", dotColor: "bg-red-400" },
};

interface Change {
  id: bigint;
  name: string;
  stage: string;
  archivedAt?: { seconds: bigint; nanos: number };
}

interface ProjectDetail {
  id: bigint;
  name: string;
  slug: string;
  description: string;
  readme: string;
  status: string;
  priority: string;
  health: string;
  leadId?: bigint;
  leadName: string;
  startDate?: string;
  targetDate?: string;
  icon: string;
  color: string;
}

function mapProjectDetail(
  project: NonNullable<Awaited<ReturnType<typeof projectClient.getProject>>["project"]>,
): ProjectDetail {
  return {
    id: project.id,
    name: project.name,
    slug: project.slug,
    description: project.description,
    readme: project.readme,
    status: project.status,
    priority: project.priority,
    health: project.health,
    leadId: project.leadId,
    leadName: project.leadName,
    startDate: project.startDate,
    targetDate: project.targetDate,
    icon: project.icon,
    color: project.color,
  };
}

type TabId = "overview" | "changes" | "members" | "memory";
const validProjectTabs: TabId[] = ["overview", "changes", "members", "memory"];

export default function ProjectDetailPage() {
  const params = useParams();
  const router = useRouter();
  const searchParams = useSearchParams();
  const slug = params.slug as string;
  const { t } = useI18n();

  const [project, setProject] = useState<ProjectDetail | null>(null);
  const [changes, setChanges] = useState<Change[]>([]);
  const [newChangeName, setNewChangeName] = useState("");
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const tabParam = searchParams.get("tab") as TabId | null;
  const initialProjectTab = tabParam && validProjectTabs.includes(tabParam) ? tabParam : "overview";
  const [activeTab, setActiveTabState] = useState<TabId>(initialProjectTab);

  const setActiveTab = useCallback(
    (tab: TabId) => {
      setActiveTabState(tab);
      const url = new URL(window.location.href);
      url.searchParams.set("tab", tab);
      router.replace(url.pathname + url.search, { scroll: false });
    },
    [router],
  );
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  // Dialogs
  const [renameOpen, setRenameOpen] = useState(false);
  const [renameName, setRenameName] = useState("");
  const [renameDesc, setRenameDesc] = useState("");
  const [renaming, setRenaming] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [renameArchiveMode, setRenameArchiveMode] = useState("manual");
  const [renameArchiveTrigger, setRenameArchiveTrigger] = useState("tasks_done");
  const [renameArchiveDaysDelay, setRenameArchiveDaysDelay] = useState(0);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("editor");
  const [inviting, setInviting] = useState(false);
  const [inviteSuccess, setInviteSuccess] = useState("");
  const [orgMembers, setOrgMembers] = useState<{ userId: bigint; name: string; email: string }[]>(
    [],
  );
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [members, setMembers] = useState<{ name: string; email: string; role: string }[]>([]);
  const [memoryContent, setMemoryContent] = useState("");
  const [activeProperty, setActiveProperty] = useState<string | null>(null);
  const propertyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
      if (propertyRef.current && !propertyRef.current.contains(e.target as Node)) {
        setActiveProperty(null);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  useEffect(() => {
    async function load() {
      try {
        const projectRes = await projectClient.getProject({ slug });
        if (!projectRes.project) {
          router.replace("/projects");
          return;
        }
        if (projectRes.project) {
          setProject(mapProjectDetail(projectRes.project));
          // Members from API
          setMembers(
            (projectRes.members || []).map((m) => ({
              name: m.userName,
              email: m.userEmail,
              role: m.role,
            })),
          );
          const [changesRes, memoryRes] = await Promise.all([
            projectClient.listChanges({ projectId: projectRes.project.id, filter: "active" }),
            memoryClient
              .getMemory({ projectId: projectRes.project.id })
              .catch(() => ({ memory: undefined })),
          ]);
          setChanges(
            changesRes.changes.map((c) => ({
              id: c.id,
              name: c.name,
              stage: c.stage,
              archivedAt: c.archivedAt,
            })),
          );
          setMemoryContent(memoryRes.memory?.content ?? "");
        }
      } catch (err) {
        showError(t("toast.projectLoadFailed"), err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [slug]);

  async function handleCreateChange(e: React.FormEvent) {
    e.preventDefault();
    if (!newChangeName.trim() || !project) return;
    setCreating(true);
    try {
      const res = await projectClient.createChange({
        projectId: project.id,
        name: newChangeName.trim(),
      });
      if (res.change) {
        setChanges((prev) => [
          {
            id: res.change!.id,
            name: res.change!.name,
            stage: res.change!.stage,
            archivedAt: res.change!.archivedAt,
          },
          ...prev,
        ]);
        setNewChangeName("");
      }
    } catch (err) {
      showError(t("toast.createFailed"), err);
    } finally {
      setCreating(false);
    }
  }

  async function handleRename() {
    if (!project || !renameName.trim()) return;
    setRenaming(true);
    try {
      const [res] = await Promise.all([
        projectClient.updateProject({
          id: project.id,
          name: renameName.trim(),
          description: renameDesc,
          projectId: project.id,
        }),
        projectClient.updateArchivePolicy({
          projectId: project.id,
          mode: renameArchiveMode,
          trigger: renameArchiveTrigger,
          daysDelay: renameArchiveDaysDelay,
        }),
      ]);
      if (res.project) {
        setProject(mapProjectDetail(res.project));
        setRenameOpen(false);
        if (res.project.slug !== slug) router.replace(`/projects/${res.project.slug}`);
      }
      showSuccess(t("toast.saveSuccess"));
    } catch (err) {
      showError(t("toast.updateFailed"), err);
    } finally {
      setRenaming(false);
    }
  }

  async function handlePropertyUpdate(field: string, value: string | bigint) {
    if (!project) return;
    const prev = { ...project };
    // Optimistic update
    setProject({
      ...project,
      [field]: value,
      ...(field === "leadId"
        ? {
            leadId: value === BigInt(0) ? undefined : (value as bigint),
            leadName:
              value === BigInt(0)
                ? ""
                : (orgMembers.find((member) => member.userId === value)?.name ?? project.leadName),
          }
        : {}),
    } as ProjectDetail);
    setActiveProperty(null);
    try {
      const updatePayload: Record<string, unknown> = { id: project.id, projectId: project.id };
      if (field === "status") updatePayload.status = value as string;
      else if (field === "priority") updatePayload.priority = value as string;
      else if (field === "health") updatePayload.health = value as string;
      else if (field === "leadId") {
        updatePayload.leadId = value as bigint;
      } else if (field === "startDate") updatePayload.startDate = value as string;
      else if (field === "targetDate") updatePayload.targetDate = value as string;
      const res = await projectClient.updateProject(
        updatePayload as Parameters<typeof projectClient.updateProject>[0],
      );
      if (res.project) {
        setProject(mapProjectDetail(res.project));
      }
    } catch (err) {
      setProject(prev); // rollback
      showError(t("toast.updateFailed"), err);
    }
  }

  async function handleDelete() {
    if (!project) return;
    setDeleting(true);
    try {
      await projectClient.deleteProject({ id: project.id, projectId: project.id });
      router.push("/projects");
    } catch (err) {
      showError(t("toast.deleteFailed"), err);
      setDeleting(false);
    }
  }

  useEffect(() => {
    if (!inviteOpen && activeProperty !== "lead") return;
    if (orgMembers.length > 0) return;
    orgClient
      .listMembers({})
      .then((res) => {
        setOrgMembers(
          res.members.map((m) => ({ userId: m.userId, name: m.userName, email: m.userEmail })),
        );
      })
      .catch((err: unknown) => { showError(t("toast.loadFailed"), err); });
  }, [inviteOpen, activeProperty, orgMembers.length]);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const filteredOrgMembers = orgMembers
    .filter((m) => {
      if (!inviteEmail.trim()) return true;
      const q = inviteEmail.toLowerCase();
      return m.name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q);
    })
    .filter((m) => !members.some((pm) => pm.email === m.email));

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    if (!project || !inviteEmail.trim()) return;
    setInviting(true);
    try {
      await projectClient.inviteMember({
        projectId: project.id,
        email: inviteEmail.trim(),
        role: inviteRole,
      });
      setInviteSuccess(inviteEmail.trim());
      setInviteEmail("");
      setTimeout(() => setInviteSuccess(""), 3000);
      showSuccess(t("toast.inviteSuccess"));
    } catch (err) {
      showError(t("toast.inviteFailed"), err);
    } finally {
      setInviting(false);
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4">
        <h1 className="text-2xl font-bold">{t("common.notFound")}</h1>
        <Link href="/projects">
          <Button variant="outline" className="cursor-pointer">
            {t("common.backToProjects")}
          </Button>
        </Link>
      </div>
    );
  }

  const tabs: { id: TabId; label: string }[] = [
    { id: "overview", label: t("project.overview") },
    { id: "changes", label: t("project.changes") },
    { id: "members", label: t("project.members") },
    { id: "memory", label: t("project.memory") },
  ];

  return (
    <div className="min-h-screen bg-background">
      <Header breadcrumbs={[{ label: project.name }]} />

      <main className="mx-auto max-w-5xl px-6 pt-10 pb-16">
        {/* Project Hero — Linear style */}
        <div className="mb-8">
          {/* Icon */}
          <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Layers className="size-6" />
          </div>

          {/* Title + Menu */}
          <div className="flex items-start justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight">{project.name}</h1>
              {project.description ? (
                <p className="mt-1 text-sm text-muted-foreground">{project.description}</p>
              ) : (
                <p
                  className="mt-1 text-sm text-muted-foreground/40 cursor-pointer hover:text-muted-foreground/60 transition-colors"
                  onClick={() => {
                    setRenameName(project.name);
                    setRenameDesc("");
                    projectClient
                      .getArchivePolicy({ projectId: project.id })
                      .then((res) => {
                        if (res.policy) {
                          setRenameArchiveMode(res.policy.mode || "manual");
                          setRenameArchiveTrigger(res.policy.trigger || "tasks_done");
                          setRenameArchiveDaysDelay(res.policy.daysDelay ?? 0);
                        }
                      })
                      .catch((err: unknown) => { showError(t("toast.loadFailed"), err); });
                    setRenameOpen(true);
                  }}
                >
                  {t("project.addSummary")}
                </p>
              )}
            </div>
            <div className="relative" ref={menuRef}>
              <button
                onClick={() => setMenuOpen(!menuOpen)}
                className="flex h-8 w-8 cursor-pointer items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                <MoreHorizontal className="size-4" />
              </button>
              {menuOpen && (
                <div className="absolute right-0 top-10 z-50 w-48 rounded-xl border border-border/50 bg-popover p-1.5 shadow-xl animate-in fade-in slide-in-from-top-2 duration-150">
                  <button
                    onClick={() => {
                      setMenuOpen(false);
                      setRenameName(project.name);
                      setRenameDesc(project.description);
                      projectClient
                        .getArchivePolicy({ projectId: project.id })
                        .then((res) => {
                          if (res.policy) {
                            setRenameArchiveMode(res.policy.mode || "manual");
                            setRenameArchiveTrigger(res.policy.trigger || "tasks_done");
                            setRenameArchiveDaysDelay(res.policy.daysDelay ?? 0);
                          }
                        })
                        .catch((err: unknown) => { showError(t("toast.loadFailed"), err); });
                      setRenameOpen(true);
                    }}
                    className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
                  >
                    <Pencil className="size-3.5 text-muted-foreground" />
                    {t("project.editProject")}
                  </button>
                  <button
                    onClick={() => {
                      setMenuOpen(false);
                      setInviteOpen(true);
                    }}
                    className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
                  >
                    <UserPlus className="size-3.5 text-muted-foreground" />
                    {t("project.inviteMember")}
                  </button>
                  <div className="my-1.5 border-t border-border/50" />
                  <button
                    onClick={() => {
                      setMenuOpen(false);
                      setDeleteConfirm("");
                      setDeleteOpen(true);
                    }}
                    className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-destructive transition-colors hover:bg-destructive/10"
                  >
                    <Trash2 className="size-3.5" />
                    {t("project.deleteProject")}
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Properties row — Linear style */}
          <div
            className="mt-5 flex flex-wrap items-center gap-x-1.5 gap-y-1.5 text-sm"
            ref={propertyRef}
          >
            <span className="mr-1.5 text-xs text-muted-foreground/50">
              {t("project.properties")}
            </span>

            {/* Status */}
            <div className="relative">
              <button
                onClick={() => setActiveProperty(activeProperty === "status" ? null : "status")}
                className="flex cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-foreground/80 transition-colors hover:bg-accent"
              >
                <div
                  className={`h-2 w-2 rounded-full ${statusConfig[project.status]?.dotColor ?? "bg-muted-foreground"}`}
                />
                <span>{statusConfig[project.status]?.label ?? project.status}</span>
              </button>
              {activeProperty === "status" && (
                <div className="absolute left-0 top-full z-50 mt-1 w-40 rounded-lg border border-border/50 bg-popover p-1 shadow-xl animate-in fade-in slide-in-from-top-1 duration-100">
                  {Object.entries(statusConfig).map(([key, cfg]) => (
                    <button
                      key={key}
                      onClick={() => handlePropertyUpdate("status", key)}
                      className={`flex w-full cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors hover:bg-accent ${project.status === key ? "bg-accent/50" : ""}`}
                    >
                      <div className={`h-2 w-2 rounded-full ${cfg.dotColor}`} />
                      {cfg.label}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Priority */}
            <div className="relative">
              <button
                onClick={() => setActiveProperty(activeProperty === "priority" ? null : "priority")}
                className="flex cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-foreground/80 transition-colors hover:bg-accent"
              >
                <span className="text-xs font-mono text-muted-foreground">
                  {priorityConfig[project.priority]?.icon ?? "···"}
                </span>
                <span>{priorityConfig[project.priority]?.label ?? "No priority"}</span>
              </button>
              {activeProperty === "priority" && (
                <div className="absolute left-0 top-full z-50 mt-1 w-40 rounded-lg border border-border/50 bg-popover p-1 shadow-xl animate-in fade-in slide-in-from-top-1 duration-100">
                  {Object.entries(priorityConfig).map(([key, cfg]) => (
                    <button
                      key={key}
                      onClick={() => handlePropertyUpdate("priority", key)}
                      className={`flex w-full cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors hover:bg-accent ${project.priority === key ? "bg-accent/50" : ""}`}
                    >
                      <span className="w-5 text-xs font-mono text-muted-foreground">
                        {cfg.icon}
                      </span>
                      {cfg.label}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Health */}
            <div className="relative">
              <button
                onClick={() => setActiveProperty(activeProperty === "health" ? null : "health")}
                className="flex cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-foreground/80 transition-colors hover:bg-accent"
              >
                <Signal className="size-3.5 text-muted-foreground/60" />
                <span>{healthConfig[project.health]?.label ?? "On Track"}</span>
              </button>
              {activeProperty === "health" && (
                <div className="absolute left-0 top-full z-50 mt-1 w-40 rounded-lg border border-border/50 bg-popover p-1 shadow-xl animate-in fade-in slide-in-from-top-1 duration-100">
                  {Object.entries(healthConfig).map(([key, cfg]) => (
                    <button
                      key={key}
                      onClick={() => handlePropertyUpdate("health", key)}
                      className={`flex w-full cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors hover:bg-accent ${project.health === key ? "bg-accent/50" : ""}`}
                    >
                      <div className={`h-2 w-2 rounded-full ${cfg.dotColor}`} />
                      {cfg.label}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Lead */}
            <div className="relative">
              <button
                onClick={() => setActiveProperty(activeProperty === "lead" ? null : "lead")}
                className="flex cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 transition-colors hover:bg-accent"
              >
                <User className="size-3.5 text-muted-foreground/60" />
                <span
                  className={project.leadName ? "text-foreground/80" : "text-muted-foreground/40"}
                >
                  {project.leadName || "Lead"}
                </span>
              </button>
              {activeProperty === "lead" && (
                <div className="absolute left-0 top-full z-50 mt-1 w-52 rounded-lg border border-border/50 bg-popover p-1 shadow-xl animate-in fade-in slide-in-from-top-1 duration-100">
                  <button
                    onClick={() => handlePropertyUpdate("leadId", BigInt(0))}
                    className="flex w-full cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent"
                  >
                    No lead
                  </button>
                  {members.map((m) => (
                    <button
                      key={m.email}
                      onClick={() => {
                        const orgMember = orgMembers.find((om) => om.email === m.email);
                        if (orgMember) handlePropertyUpdate("leadId", orgMember.userId);
                      }}
                      className="flex w-full cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors hover:bg-accent"
                    >
                      <div className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                        {m.name?.[0]?.toUpperCase() ?? "?"}
                      </div>
                      {m.name || m.email}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <DatePicker
              value={project.startDate}
              placeholder="Start date"
              onChange={(value) => handlePropertyUpdate("startDate", value)}
            />

            <DatePicker
              value={project.targetDate}
              placeholder="Target date"
              onChange={(value) => handlePropertyUpdate("targetDate", value)}
            />

            {/* Members count */}
            <div className="flex items-center gap-1.5 rounded-md px-2 py-1 text-foreground/80">
              <Users className="size-3.5 text-muted-foreground/60" />
              <span>
                {members.length} {t("project.membersCount")}
              </span>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="mb-6 flex gap-1 border-b border-border/50">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`cursor-pointer whitespace-nowrap px-4 py-2.5 text-sm font-medium transition-colors duration-200 ${
                activeTab === tab.id
                  ? "border-b-2 border-primary text-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Tab Content */}
        {activeTab === "overview" && (
          <OverviewTab
            readme={project.readme}
            projectId={project.id}
            changes={changes}
            projectSlug={project.slug}
            onViewChanges={() => setActiveTab("changes")}
            onReadmeUpdate={(readme) => setProject({ ...project, readme })}
            t={t}
          />
        )}

        {activeTab === "changes" && (
          <ChangesTab
            projectId={project.id}
            initialChanges={changes}
            projectSlug={project.slug}
            newChangeName={newChangeName}
            setNewChangeName={setNewChangeName}
            creating={creating}
            onCreateChange={handleCreateChange}
            t={t}
          />
        )}

        {activeTab === "members" && (
          <MembersTab members={members} onInvite={() => setInviteOpen(true)} t={t} />
        )}

        {activeTab === "memory" && (
          <MemoryTab projectId={project.id} content={memoryContent} t={t} />
        )}
      </main>

      {/* Edit Project Dialog */}
      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("project.editProject")}</DialogTitle>
            <DialogDescription>{t("project.editProjectDesc")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-5 py-2">
            <div className="space-y-2">
              <Label htmlFor="rename-name">{t("projects.projectName")}</Label>
              <Input
                id="rename-name"
                value={renameName}
                onChange={(e) => setRenameName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="rename-desc">{t("projects.description")}</Label>
              <Input
                id="rename-desc"
                value={renameDesc}
                onChange={(e) => setRenameDesc(e.target.value)}
              />
            </div>

            <div className="border-t border-border/50 pt-4">
              <p className="mb-3 text-sm font-medium">{t("projectSettings.archivePolicy")}</p>
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">
                    {t("projectSettings.archiveMode")}
                  </Label>
                  <Select
                    value={renameArchiveMode}
                    onValueChange={(v) => v && setRenameArchiveMode(v)}
                  >
                    <SelectTrigger className="cursor-pointer">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="manual" className="cursor-pointer">
                        {t("projectSettings.archiveModeManual")}
                      </SelectItem>
                      <SelectItem value="auto" className="cursor-pointer">
                        {t("projectSettings.archiveModeAuto")}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {renameArchiveMode === "auto" && (
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">
                      {t("projectSettings.archiveTrigger")}
                    </Label>
                    <Select
                      value={renameArchiveTrigger}
                      onValueChange={(v) => v && setRenameArchiveTrigger(v)}
                    >
                      <SelectTrigger className="w-full cursor-pointer">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="min-w-[280px]">
                        <SelectItem value="tasks_done" className="cursor-pointer">
                          {t("projectSettings.archiveTriggerTasksDone")}
                        </SelectItem>
                        <SelectItem value="days_after_ready" className="cursor-pointer">
                          {t("projectSettings.archiveTriggerDaysAfterReady")}
                        </SelectItem>
                        <SelectItem value="tasks_done_and_days" className="cursor-pointer">
                          {t("projectSettings.archiveTriggerTasksDoneAndDays")}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}

                {renameArchiveMode === "auto" &&
                  (renameArchiveTrigger === "days_after_ready" ||
                    renameArchiveTrigger === "tasks_done_and_days") && (
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">
                        {t("projectSettings.archiveDaysDelay")}
                      </Label>
                      <Input
                        type="number"
                        min={0}
                        max={365}
                        value={renameArchiveDaysDelay}
                        onChange={(e) => setRenameArchiveDaysDelay(Number(e.target.value))}
                        className="w-32"
                      />
                    </div>
                  )}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={handleRename}
              disabled={renaming || !renameName.trim()}
              className="cursor-pointer"
            >
              {renaming ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-destructive">{t("project.deleteProject")}</DialogTitle>
            <DialogDescription>{t("project.deleteProjectDesc")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2 py-2">
            <Label htmlFor="delete-confirm">
              Type <span className="font-mono font-medium text-foreground">{project.name}</span> to
              confirm
            </Label>
            <Input
              id="delete-confirm"
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteOpen(false)}
              className="cursor-pointer"
            >
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handleDelete}
              disabled={deleting || deleteConfirm !== project.name}
              className="cursor-pointer bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleting ? t("common.loading") : t("project.deleteProject")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Invite Dialog */}
      <Dialog open={inviteOpen} onOpenChange={setInviteOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("project.inviteMember")}</DialogTitle>
            <DialogDescription>{t("project.inviteMemberDesc")}</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleInvite}>
            <div className="space-y-4 py-2">
              <div className="relative space-y-2" ref={dropdownRef}>
                <Label htmlFor="invite-email">{t("auth.email")}</Label>
                <Input
                  id="invite-email"
                  type="text"
                  placeholder="Search by name or email..."
                  value={inviteEmail}
                  onChange={(e) => {
                    setInviteEmail(e.target.value);
                    setShowDropdown(true);
                  }}
                  onFocus={() => setShowDropdown(true)}
                  autoComplete="off"
                  required
                />
                {showDropdown && filteredOrgMembers.length > 0 && (
                  <div className="absolute left-0 right-0 top-full z-50 mt-1 max-h-48 overflow-y-auto rounded-lg border border-border/50 bg-popover shadow-xl">
                    {filteredOrgMembers.map((m) => (
                      <button
                        key={String(m.userId)}
                        type="button"
                        onClick={() => {
                          setInviteEmail(m.email);
                          setShowDropdown(false);
                        }}
                        className="flex w-full cursor-pointer items-center gap-3 px-3 py-2 text-left transition-colors hover:bg-accent"
                      >
                        <div className="flex size-7 items-center justify-center rounded-full bg-accent text-xs font-medium">
                          {m.name?.[0]?.toUpperCase() ?? m.email[0].toUpperCase()}
                        </div>
                        <div>
                          <p className="text-sm font-medium">{m.name || "—"}</p>
                          <p className="text-xs text-muted-foreground">{m.email}</p>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("project.role")}</Label>
                <div className="flex gap-2">
                  {["editor", "viewer"].map((role) => (
                    <button
                      key={role}
                      type="button"
                      onClick={() => setInviteRole(role)}
                      className={`cursor-pointer rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                        inviteRole === role
                          ? "border-primary bg-primary/10 text-primary"
                          : "border-border/50 text-muted-foreground hover:border-border hover:text-foreground"
                      }`}
                    >
                      {role.charAt(0).toUpperCase() + role.slice(1)}
                    </button>
                  ))}
                </div>
              </div>
              {inviteSuccess && (
                <p className="text-sm text-emerald-400">
                  {t("project.inviteSent")} {inviteSuccess}
                </p>
              )}
            </div>
            <DialogFooter>
              <Button
                type="submit"
                disabled={inviting || !inviteEmail.trim()}
                className="cursor-pointer"
              >
                {inviting ? t("common.loading") : t("common.invite")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── Overview Tab ───────────────────────────────────────

function OverviewTab({
  readme,
  projectId,
  changes,
  projectSlug,
  onViewChanges,
  onReadmeUpdate,
  t,
}: {
  readme: string;
  projectId: bigint;
  changes: Change[];
  projectSlug: string;
  onViewChanges: () => void;
  onReadmeUpdate: (desc: string) => void;
  t: (key: string) => string;
}) {
  const handleReadmeSave = async (html: string) => {
    try {
      await projectClient.updateProject({ id: projectId, readme: html, projectId });
      onReadmeUpdate(html);
    } catch (err) {
      showError(t("toast.saveFailed"), err);
    }
  };

  return (
    <div className="space-y-6">
      {/* README */}
      <div className="rounded-xl border border-border/40 bg-card/50">
        <div className="flex items-center gap-2 border-b border-border/40 px-5 py-3">
          <FileText className="size-4 text-muted-foreground" />
          <span className="text-sm font-medium">README</span>
        </div>
        <ReadmeEditor
          initialContent={readme}
          onSave={handleReadmeSave}
          placeholder="Write your project README..."
        />
      </div>

      {/* Recent Changes */}
      <div className="rounded-xl border border-border/40 bg-card/50">
        <div className="flex items-center justify-between border-b border-border/40 px-5 py-3">
          <span className="text-sm font-medium">{t("project.recentChanges")}</span>
          {changes.length > 3 && (
            <button
              onClick={onViewChanges}
              className="flex cursor-pointer items-center gap-1 text-xs text-primary hover:text-primary/80 transition-colors"
            >
              {t("project.viewAll")}
              <ChevronRight className="size-3" />
            </button>
          )}
        </div>
        {changes.length === 0 ? (
          <div className="py-8 text-center">
            <p className="text-sm text-muted-foreground">{t("project.noChanges")}</p>
          </div>
        ) : (
          <div className="divide-y divide-border/30">
            {changes.slice(0, 3).map((change) => {
              const config = stageConfig[change.stage] ?? stageConfig.draft;
              return (
                <Link
                  key={String(change.id)}
                  href={`/projects/${projectSlug}/changes/${change.id}`}
                >
                  <div className="flex cursor-pointer items-center justify-between px-5 py-3 transition-colors hover:bg-accent/50">
                    <div className="flex items-center gap-3">
                      <div
                        className={`flex h-7 w-7 items-center justify-center rounded-md border ${config.color}`}
                      >
                        <svg
                          className="h-3 w-3"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={1.5}
                            d={config.icon}
                          />
                        </svg>
                      </div>
                      <span className="text-sm">{change.name}</span>
                    </div>
                    <span className={`text-xs font-medium ${config.color.split(" ")[1]}`}>
                      {config.label}
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Changes Tab ────────────────────────────────────────

function ChangesTab({
  projectId,
  initialChanges,
  projectSlug,
  newChangeName,
  setNewChangeName,
  creating,
  onCreateChange,
  t,
}: {
  projectId: bigint;
  initialChanges: Change[];
  projectSlug: string;
  newChangeName: string;
  setNewChangeName: (v: string) => void;
  creating: boolean;
  onCreateChange: (e: React.FormEvent) => void;
  t: (key: string) => string;
}) {
  const [archiveFilter, setArchiveFilter] = useState<"active" | "archived">("active");
  const [stageFilters, setStageFilters] = useState<Set<string>>(new Set());
  const [activeChanges, setActiveChanges] = useState<Change[]>(initialChanges);
  const [archivedChanges, setArchivedChanges] = useState<Change[]>([]);
  const [archivedCount, setArchivedCount] = useState<number | null>(null);
  const [loadingArchived, setLoadingArchived] = useState(false);

  const stages = ["draft", "design", "review", "ready"] as const;

  function toggleStageFilter(stage: string) {
    setStageFilters((prev) => {
      const next = new Set(prev);
      if (next.has(stage)) {
        next.delete(stage);
      } else {
        next.add(stage);
      }
      return next;
    });
  }

  // Fetch archived count once on mount
  useEffect(() => {
    projectClient
      .listChanges({ projectId, filter: "archived" })
      .then((res) => {
        setArchivedCount(res.changes.length);
        setArchivedChanges(
          res.changes.map((c) => ({
            id: c.id,
            name: c.name,
            stage: c.stage,
            archivedAt: c.archivedAt,
          })),
        );
      })
      .catch((err: unknown) => {
        setArchivedCount(0);
        showError(t("toast.changesLoadFailed"), err);
      });
  }, [projectId]);

  async function switchToFilter(filter: "active" | "archived") {
    setArchiveFilter(filter);
    if (filter === "archived" && archivedChanges.length === 0 && archivedCount !== 0) {
      setLoadingArchived(true);
      try {
        const res = await projectClient.listChanges({ projectId, filter: "archived" });
        setArchivedChanges(
          res.changes.map((c) => ({
            id: c.id,
            name: c.name,
            stage: c.stage,
            archivedAt: c.archivedAt,
          })),
        );
        setArchivedCount(res.changes.length);
      } catch (err) {
        showError(t("toast.changesLoadFailed"), err);
      } finally {
        setLoadingArchived(false);
      }
    }
    if (filter === "active") {
      try {
        const res = await projectClient.listChanges({ projectId, filter: "active" });
        setActiveChanges(
          res.changes.map((c) => ({
            id: c.id,
            name: c.name,
            stage: c.stage,
            archivedAt: c.archivedAt,
          })),
        );
      } catch (err) {
        showError(t("toast.changesLoadFailed"), err);
      }
    }
  }

  const baseChanges = archiveFilter === "active" ? activeChanges : archivedChanges;
  const displayChanges =
    stageFilters.size === 0 || archiveFilter === "archived"
      ? baseChanges
      : baseChanges.filter((c) => stageFilters.has(c.stage));

  return (
    <div>
      {/* Archive Filter Tabs */}
      <div className="mb-4 flex gap-1 border-b border-border/50">
        <button
          onClick={() => switchToFilter("active")}
          className={`cursor-pointer whitespace-nowrap px-4 py-2.5 text-sm font-medium transition-colors duration-200 ${
            archiveFilter === "active"
              ? "border-b-2 border-primary text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          {t("project.activeChanges")}
          <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
            {activeChanges.length}
          </span>
        </button>
        <button
          onClick={() => switchToFilter("archived")}
          className={`cursor-pointer whitespace-nowrap px-4 py-2.5 text-sm font-medium transition-colors duration-200 ${
            archiveFilter === "archived"
              ? "border-b-2 border-primary text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          {t("project.archivedChanges")}
          {archivedCount !== null && (
            <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              {archivedCount}
            </span>
          )}
        </button>
      </div>

      {/* Stage Filter Chips — only for active */}
      {archiveFilter === "active" && activeChanges.length > 0 && (
        <div className="mb-4 flex flex-wrap gap-1.5">
          <button
            onClick={() => setStageFilters(new Set())}
            className={`cursor-pointer inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
              stageFilters.size === 0
                ? "bg-foreground/10 text-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-foreground/5"
            }`}
          >
            {t("project.allStages")}
            <span className="text-[10px] tabular-nums opacity-60">{activeChanges.length}</span>
          </button>
          {stages.map((stage) => {
            const config = stageConfig[stage];
            const count = activeChanges.filter((c) => c.stage === stage).length;
            const isSelected = stageFilters.has(stage);
            return (
              <button
                key={stage}
                onClick={() => toggleStageFilter(stage)}
                className={`cursor-pointer inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                  isSelected
                    ? "bg-foreground/10 text-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-foreground/5"
                }`}
              >
                <span
                  className={`h-1.5 w-1.5 rounded-full ${config.color.split(" ")[0].replace("/10", "")}`}
                />
                {config.label}
                <span className="text-[10px] tabular-nums opacity-60">{count}</span>
              </button>
            );
          })}
        </div>
      )}

      {/* Create Change — only for active */}
      {archiveFilter === "active" && (
        <form onSubmit={onCreateChange} className="mb-6">
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Plus className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground/50" />
              <Input
                value={newChangeName}
                onChange={(e) => setNewChangeName(e.target.value)}
                placeholder={t("project.newChangePlaceholder")}
                className="pl-9 bg-card/50 border-border/40 focus:border-primary/50 transition-colors"
              />
            </div>
            <Button
              type="submit"
              disabled={creating || !newChangeName.trim()}
              className="cursor-pointer px-5"
            >
              {creating ? (
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
              ) : (
                t("common.create")
              )}
            </Button>
          </div>
        </form>
      )}

      {/* Changes List */}
      {loadingArchived ? (
        <div className="flex items-center justify-center py-20">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      ) : displayChanges.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/40 bg-card/30 py-20">
          <div className="mb-5 rounded-2xl bg-primary/5 p-5">
            <FileText className="size-10 text-primary/40" />
          </div>
          {archiveFilter === "active" && stageFilters.size > 0 && baseChanges.length > 0 ? (
            <p className="text-sm font-medium text-foreground/70">
              {t("project.noMatchingChanges")}
            </p>
          ) : (
            <>
              <p className="text-sm font-medium text-foreground/70">{t("project.noChanges")}</p>
              {archiveFilter === "active" && (
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("project.createFirstChange")}
                </p>
              )}
            </>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {displayChanges.map((change) => {
            const config = stageConfig[change.stage] ?? stageConfig.draft;
            return (
              <Link key={String(change.id)} href={`/projects/${projectSlug}/changes/${change.id}`}>
                <div
                  className={`group flex cursor-pointer items-center justify-between rounded-xl border border-border/40 bg-card/50 px-5 py-4 transition-all duration-200 hover:border-primary/20 hover:bg-card/80 hover:shadow-lg ${config.glow}`}
                >
                  <div className="flex items-center gap-4">
                    <div
                      className={`flex h-8 w-8 items-center justify-center rounded-lg border ${config.color}`}
                    >
                      <svg
                        className="h-3.5 w-3.5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={1.5}
                          d={config.icon}
                        />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-medium group-hover:text-foreground transition-colors">
                        {change.name}
                      </p>
                      <span
                        className={`inline-flex items-center text-xs font-medium mt-0.5 ${config.color.split(" ")[1]}`}
                      >
                        {config.label}
                      </span>
                    </div>
                  </div>
                  <ChevronRight className="size-4 text-muted-foreground/30 transition-all duration-200 group-hover:text-muted-foreground group-hover:translate-x-0.5" />
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ─── Members Tab ────────────────────────────────────────

function MembersTab({
  members,
  onInvite,
  t,
}: {
  members: { name: string; email: string; role: string }[];
  onInvite: () => void;
  t: (key: string) => string;
}) {
  const roleColor: Record<string, string> = {
    Owner: "text-amber-400 bg-amber-400/10",
    Editor: "text-blue-400 bg-blue-400/10",
    Member: "text-muted-foreground bg-muted",
  };

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <span className="text-sm text-muted-foreground">
          {members.length} {t("project.membersCount")}
        </span>
        <button
          onClick={onInvite}
          className="flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-xs text-primary hover:bg-primary/10 transition-colors"
        >
          <UserPlus className="size-3.5" />
          {t("project.inviteMember")}
        </button>
      </div>
      <div className="space-y-2">
        {members.map((member) => (
          <div
            key={member.email}
            className="flex items-center justify-between rounded-xl border border-border/40 bg-card/50 px-5 py-3.5"
          >
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-sm font-bold uppercase text-primary">
                {member.name.charAt(0)}
              </div>
              <div>
                <p className="text-sm font-medium">{member.name}</p>
                <p className="text-xs text-muted-foreground">{member.email}</p>
              </div>
            </div>
            <span
              className={`rounded-md px-2 py-0.5 text-xs font-medium ${roleColor[member.role] || roleColor.Member}`}
            >
              {member.role}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Memory Tab ─────────────────────────────────────────

function MemoryTab({
  projectId,
  content,
  t,
}: {
  projectId: bigint;
  content: string;
  t: (key: string) => string;
}) {
  const handleMemorySave = async (html: string) => {
    try {
      await memoryClient.saveMemory({ projectId, content: html });
    } catch (err) {
      showError(t("toast.saveFailed"), err);
    }
  };

  return (
    <div>
      <div className="mb-4 flex items-center gap-2">
        <Brain className="size-4 text-muted-foreground" />
        <span className="text-sm text-muted-foreground">{t("project.memoryDesc")}</span>
      </div>

      <div className="rounded-xl border border-border/40 bg-card/50">
        <div className="flex items-center gap-2 border-b border-border/30 px-5 py-3">
          <Brain className="size-3.5 text-primary/60" />
          <span className="text-sm font-medium">{t("project.memory")}</span>
        </div>
        <ReadmeEditor
          initialContent={content.startsWith("<") ? content : marked.parse(content, { async: false }) as string}
          onSave={handleMemorySave}
          placeholder="Write project memory — domain rules, business context, constraints..."
        />
      </div>
    </div>
  );
}
