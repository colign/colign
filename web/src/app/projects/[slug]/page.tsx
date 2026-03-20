"use client";

import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { useI18n } from "@/lib/i18n";
import {
  GitBranch,
  Users,
  FileText,
  ExternalLink,
  Pencil,
  UserPlus,
  Trash2,
  MoreHorizontal,
  Plus,
  ChevronRight,
  Globe,
  Layers,
  Brain,
  Save,
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

interface Change {
  id: bigint;
  name: string;
  stage: string;
}

interface ProjectDetail {
  id: bigint;
  name: string;
  slug: string;
  description: string;
}

type TabId = "overview" | "changes" | "members" | "memory";

export default function ProjectDetailPage() {
  const params = useParams();
  const router = useRouter();
  const slug = params.slug as string;
  const { t } = useI18n();

  const [project, setProject] = useState<ProjectDetail | null>(null);
  const [changes, setChanges] = useState<Change[]>([]);
  const [newChangeName, setNewChangeName] = useState("");
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [activeTab, setActiveTab] = useState<TabId>("overview");
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
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [inviting, setInviting] = useState(false);
  const [inviteSuccess, setInviteSuccess] = useState("");

  // Mock data for overview
  const mockReadme = `## About\nA collaborative platform for spec-driven development. Teams discuss and write specs with AI before implementing.\n\n## Getting Started\n1. Create a new Change\n2. Write the Proposal with acceptance criteria\n3. Design the solution\n4. Review and approve\n5. Mark as Ready for implementation`;
  const mockRepos = [
    { name: "colign", url: "https://github.com/gobenpark/colign" },
    { name: "colign-web", url: "https://github.com/gobenpark/colign-web" },
    { name: "colign-infra", url: "https://github.com/gobenpark/colign-infra" },
  ];
  const mockTechStack = ["Go", "React", "Next.js", "PostgreSQL", "Y.js"];
  const mockMembers = [
    { name: "Ben Park", email: "ben@colign.dev", role: "Owner" },
    { name: "Jane Kim", email: "jane@colign.dev", role: "Editor" },
    { name: "Alex Lee", email: "alex@colign.dev", role: "Member" },
  ];
  const mockMemory = `## Domain Rules\n- SDD workflow must follow Draft → Design → Review → Ready order\n- A Change cannot skip stages\n- Acceptance Criteria must be defined before entering Review\n- All specs are written in English regardless of team language\n\n## Business Context\nTarget users are engineering teams (5-50 people) who want to align on specs before coding. Key differentiator: AI-assisted spec writing with real-time collaboration.\n\n## Constraints & Decisions\n- Payment integration must use Stripe only (contractual)\n- Data residency: all user data must stay in ap-northeast-2 (Seoul)\n- OAuth providers limited to GitHub and Google for MVP\n- Max document size: 1MB per Change document`;

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  useEffect(() => {
    async function load() {
      try {
        const projectRes = await projectClient.getProject({ slug });
        if (projectRes.project) {
          setProject({
            id: projectRes.project.id,
            name: projectRes.project.name,
            slug: projectRes.project.slug,
            description: projectRes.project.description,
          });
          const changesRes = await projectClient.listChanges({ projectId: projectRes.project.id });
          setChanges(changesRes.changes.map((c) => ({ id: c.id, name: c.name, stage: c.stage })));
        }
      } catch {
        // handle error
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
      const res = await projectClient.createChange({ projectId: project.id, name: newChangeName.trim() });
      if (res.change) {
        setChanges((prev) => [{ id: res.change!.id, name: res.change!.name, stage: res.change!.stage }, ...prev]);
        setNewChangeName("");
      }
    } catch {
      // handle error
    } finally {
      setCreating(false);
    }
  }

  async function handleRename() {
    if (!project || !renameName.trim()) return;
    setRenaming(true);
    try {
      const res = await projectClient.updateProject({ id: project.id, name: renameName.trim(), description: renameDesc });
      if (res.project) {
        setProject({ id: res.project.id, name: res.project.name, slug: res.project.slug, description: res.project.description });
        setRenameOpen(false);
        if (res.project.slug !== slug) router.replace(`/projects/${res.project.slug}`);
      }
    } catch {
      // handle error
    } finally {
      setRenaming(false);
    }
  }

  async function handleDelete() {
    if (!project) return;
    setDeleting(true);
    try {
      await projectClient.deleteProject({ id: project.id });
      router.push("/projects");
    } catch {
      setDeleting(false);
    }
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    if (!project || !inviteEmail.trim()) return;
    setInviting(true);
    try {
      await projectClient.inviteMember({ projectId: project.id, email: inviteEmail.trim(), role: inviteRole });
      setInviteSuccess(inviteEmail.trim());
      setInviteEmail("");
      setTimeout(() => setInviteSuccess(""), 3000);
    } catch {
      // handle error
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
          <Button variant="outline" className="cursor-pointer">{t("common.backToProjects")}</Button>
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
                <p className="mt-1 text-sm text-muted-foreground/40 cursor-pointer hover:text-muted-foreground/60 transition-colors"
                   onClick={() => { setRenameName(project.name); setRenameDesc(""); setRenameOpen(true); }}
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
                  onClick={() => { setMenuOpen(false); setRenameName(project.name); setRenameDesc(project.description); setRenameOpen(true); }}
                  className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
                >
                  <Pencil className="size-3.5 text-muted-foreground" />
                  {t("project.editProject")}
                </button>
                <button
                  onClick={() => { setMenuOpen(false); setInviteOpen(true); }}
                  className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
                >
                  <UserPlus className="size-3.5 text-muted-foreground" />
                  {t("project.inviteMember")}
                </button>
                <div className="my-1.5 border-t border-border/50" />
                <button
                  onClick={() => { setMenuOpen(false); setDeleteConfirm(""); setDeleteOpen(true); }}
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
          <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground/60">{t("project.properties")}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="h-2 w-2 rounded-full bg-emerald-400" />
              <span className="text-foreground/80">{changes.length} {t("project.changes").toLowerCase()}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Users className="size-3.5 text-muted-foreground/60" />
              <span className="text-foreground/80">{mockMembers.length} {t("project.membersCount")}</span>
            </div>
            {mockTechStack.slice(0, 4).map((tech) => (
              <span key={tech} className="rounded-md bg-muted/80 px-2 py-0.5 text-xs font-medium text-foreground/70">{tech}</span>
            ))}
            {mockTechStack.length > 4 && (
              <span className="text-xs text-muted-foreground">+{mockTechStack.length - 4}</span>
            )}
          </div>

          {/* Resources row */}
          <div className="mt-3 flex items-center gap-x-6 gap-y-2 text-sm">
            <span className="text-muted-foreground/60">{t("project.resources")}</span>
            {mockRepos.map((repo) => (
              <a
                key={repo.url}
                href={repo.url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex cursor-pointer items-center gap-1.5 text-foreground/70 hover:text-foreground transition-colors"
              >
                <GitBranch className="size-3.5 text-muted-foreground/50" />
                <span>{repo.name}</span>
                <ExternalLink className="size-2.5 text-muted-foreground/30" />
              </a>
            ))}
            <button className="flex cursor-pointer items-center gap-1 text-muted-foreground/40 hover:text-muted-foreground/60 transition-colors">
              <Plus className="size-3" />
              <span>{t("project.addResource")}</span>
            </button>
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
            readme={mockReadme}
            changes={changes}
            projectSlug={project.slug}
            onViewChanges={() => setActiveTab("changes")}
            t={t}
          />
        )}

        {activeTab === "changes" && (
          <ChangesTab
            changes={changes}
            projectSlug={project.slug}
            newChangeName={newChangeName}
            setNewChangeName={setNewChangeName}
            creating={creating}
            onCreateChange={handleCreateChange}
            t={t}
          />
        )}

        {activeTab === "members" && (
          <MembersTab
            members={mockMembers}
            onInvite={() => setInviteOpen(true)}
            t={t}
          />
        )}

        {activeTab === "memory" && (
          <MemoryTab content={mockMemory} t={t} />
        )}
      </main>

      {/* Rename Dialog */}
      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("project.editProject")}</DialogTitle>
            <DialogDescription>{t("project.editProjectDesc")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="rename-name">{t("projects.projectName")}</Label>
              <Input id="rename-name" value={renameName} onChange={(e) => setRenameName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="rename-desc">{t("projects.description")}</Label>
              <Input id="rename-desc" value={renameDesc} onChange={(e) => setRenameDesc(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button onClick={handleRename} disabled={renaming || !renameName.trim()} className="cursor-pointer">
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
              Type <span className="font-mono font-medium text-foreground">{project.name}</span> to confirm
            </Label>
            <Input id="delete-confirm" value={deleteConfirm} onChange={(e) => setDeleteConfirm(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)} className="cursor-pointer">{t("common.cancel")}</Button>
            <Button onClick={handleDelete} disabled={deleting || deleteConfirm !== project.name} className="cursor-pointer bg-destructive text-destructive-foreground hover:bg-destructive/90">
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
              <div className="space-y-2">
                <Label htmlFor="invite-email">{t("auth.email")}</Label>
                <Input id="invite-email" type="email" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label>{t("project.role")}</Label>
                <div className="flex gap-2">
                  {["member", "admin"].map((role) => (
                    <button
                      key={role}
                      type="button"
                      onClick={() => setInviteRole(role)}
                      className={`cursor-pointer rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                        inviteRole === role ? "border-primary bg-primary/10 text-primary" : "border-border/50 text-muted-foreground hover:border-border hover:text-foreground"
                      }`}
                    >
                      {role.charAt(0).toUpperCase() + role.slice(1)}
                    </button>
                  ))}
                </div>
              </div>
              {inviteSuccess && <p className="text-sm text-emerald-400">{t("project.inviteSent")} {inviteSuccess}</p>}
            </div>
            <DialogFooter>
              <Button type="submit" disabled={inviting || !inviteEmail.trim()} className="cursor-pointer">
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
  changes,
  projectSlug,
  onViewChanges,
  t,
}: {
  readme: string;
  changes: Change[];
  projectSlug: string;
  onViewChanges: () => void;
  t: (key: string) => string;
}) {
  return (
    <div className="space-y-6">
      {/* README */}
      <div className="rounded-xl border border-border/40 bg-card/50">
        <div className="flex items-center justify-between border-b border-border/40 px-5 py-3">
          <div className="flex items-center gap-2">
            <FileText className="size-4 text-muted-foreground" />
            <span className="text-sm font-medium">README</span>
          </div>
          <button className="flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors">
            <Pencil className="size-3" />
            {t("common.edit")}
          </button>
        </div>
        <div className="prose prose-invert prose-sm max-w-none px-5 py-4">
          {readme.split("\n").map((line, i) => {
            if (line.startsWith("## ")) return <h2 key={i} className="mt-4 first:mt-0 text-base font-semibold">{line.slice(3)}</h2>;
            if (line.startsWith("1. ") || line.startsWith("2. ") || line.startsWith("3. ") || line.startsWith("4. ") || line.startsWith("5. ")) {
              return <p key={i} className="ml-4 text-sm text-foreground/70">{line}</p>;
            }
            if (line.trim()) return <p key={i} className="text-sm text-foreground/70">{line}</p>;
            return null;
          })}
        </div>
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
                <Link key={String(change.id)} href={`/projects/${projectSlug}/changes/${change.id}`}>
                  <div className="flex cursor-pointer items-center justify-between px-5 py-3 transition-colors hover:bg-accent/50">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-7 w-7 items-center justify-center rounded-md border ${config.color}`}>
                        <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d={config.icon} />
                        </svg>
                      </div>
                      <span className="text-sm">{change.name}</span>
                    </div>
                    <span className={`text-xs font-medium ${config.color.split(" ")[1]}`}>{config.label}</span>
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
  changes,
  projectSlug,
  newChangeName,
  setNewChangeName,
  creating,
  onCreateChange,
  t,
}: {
  changes: Change[];
  projectSlug: string;
  newChangeName: string;
  setNewChangeName: (v: string) => void;
  creating: boolean;
  onCreateChange: (e: React.FormEvent) => void;
  t: (key: string) => string;
}) {
  return (
    <div>
      {/* Create Change */}
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
          <Button type="submit" disabled={creating || !newChangeName.trim()} className="cursor-pointer px-5">
            {creating ? <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" /> : t("common.create")}
          </Button>
        </div>
      </form>

      {/* Changes List */}
      {changes.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/40 bg-card/30 py-20">
          <div className="mb-5 rounded-2xl bg-primary/5 p-5">
            <FileText className="size-10 text-primary/40" />
          </div>
          <p className="text-sm font-medium text-foreground/70">{t("project.noChanges")}</p>
          <p className="mt-1 text-xs text-muted-foreground">{t("project.createFirstChange")}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {changes.map((change) => {
            const config = stageConfig[change.stage] ?? stageConfig.draft;
            return (
              <Link key={String(change.id)} href={`/projects/${projectSlug}/changes/${change.id}`}>
                <div className={`group flex cursor-pointer items-center justify-between rounded-xl border border-border/40 bg-card/50 px-5 py-4 transition-all duration-200 hover:border-primary/20 hover:bg-card/80 hover:shadow-lg ${config.glow}`}>
                  <div className="flex items-center gap-4">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg border ${config.color}`}>
                      <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d={config.icon} />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-medium group-hover:text-foreground transition-colors">{change.name}</p>
                      <span className={`inline-flex items-center text-xs font-medium mt-0.5 ${config.color.split(" ")[1]}`}>{config.label}</span>
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
        <span className="text-sm text-muted-foreground">{members.length} {t("project.membersCount")}</span>
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
          <div key={member.email} className="flex items-center justify-between rounded-xl border border-border/40 bg-card/50 px-5 py-3.5">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-sm font-bold uppercase text-primary">
                {member.name.charAt(0)}
              </div>
              <div>
                <p className="text-sm font-medium">{member.name}</p>
                <p className="text-xs text-muted-foreground">{member.email}</p>
              </div>
            </div>
            <span className={`rounded-md px-2 py-0.5 text-xs font-medium ${roleColor[member.role] || roleColor.Member}`}>
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
  content,
  t,
}: {
  content: string;
  t: (key: string) => string;
}) {
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(content);

  function renderMarkdown(text: string) {
    return text.split("\n").map((line, i) => {
      if (line.startsWith("## ")) {
        return <h2 key={i} className="mt-5 first:mt-0 mb-2 text-sm font-semibold text-foreground">{line.slice(3)}</h2>;
      }
      if (line.startsWith("- ")) {
        return (
          <p key={i} className="ml-3 py-0.5 text-sm text-foreground/70 before:content-['•'] before:mr-2 before:text-muted-foreground/50">
            {line.slice(2)}
          </p>
        );
      }
      if (line.trim()) {
        return <p key={i} className="py-0.5 text-sm text-foreground/70">{line}</p>;
      }
      return <div key={i} className="h-2" />;
    });
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Brain className="size-4 text-muted-foreground" />
          <span className="text-sm text-muted-foreground">{t("project.memoryDesc")}</span>
        </div>
      </div>

      <div className="rounded-xl border border-border/40 bg-card/50">
        <div className="flex items-center justify-between border-b border-border/30 px-5 py-3">
          <div className="flex items-center gap-2">
            <Brain className="size-3.5 text-primary/60" />
            <span className="text-sm font-medium">{t("project.memory")}</span>
          </div>
          <button
            onClick={() => {
              if (editing) {
                // TODO: save to server
                setEditing(false);
              } else {
                setEditContent(content);
                setEditing(true);
              }
            }}
            className="flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {editing ? (
              <><Save className="size-3" /> {t("common.save")}</>
            ) : (
              <><Pencil className="size-3" /> {t("common.edit")}</>
            )}
          </button>
        </div>

        <div className="px-5 py-4">
          {editing ? (
            <textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              className="w-full resize-none rounded-md border border-border/50 bg-transparent px-3 py-2 text-sm font-mono outline-none focus:border-primary transition-colors"
              rows={Math.max(10, editContent.split("\n").length + 2)}
              autoFocus
            />
          ) : content.trim() ? (
            renderMarkdown(content)
          ) : (
            <div className="py-8 text-center">
              <Brain className="mx-auto mb-3 size-8 text-muted-foreground/30" />
              <p className="text-sm text-muted-foreground">{t("project.noMemories")}</p>
              <button
                onClick={() => { setEditContent("## Domain Rules\n- \n\n## Business Context\n\n\n## Constraints\n- "); setEditing(true); }}
                className="mt-2 cursor-pointer text-xs text-primary hover:text-primary/80 transition-colors"
              >
                {t("project.addFirstMemory")}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
