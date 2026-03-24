"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useI18n } from "@/lib/i18n";
import { projectClient } from "@/lib/project";
import { Plus, Folder, Calendar, icons, type LucideIcon } from "lucide-react";
import { showError } from "@/lib/toast";

const statusConfig: Record<string, { label: string; dotColor: string }> = {
  backlog: { label: "Backlog", dotColor: "bg-muted-foreground" },
  active: { label: "Active", dotColor: "bg-yellow-400" },
  paused: { label: "Paused", dotColor: "bg-orange-400" },
  completed: { label: "Completed", dotColor: "bg-emerald-400" },
  cancelled: { label: "Cancelled", dotColor: "bg-red-400" },
};

const priorityConfig: Record<string, { label: string; color: string }> = {
  urgent: { label: "Urgent", color: "text-red-400" },
  high: { label: "High", color: "text-orange-400" },
  medium: { label: "Medium", color: "text-yellow-400" },
  low: { label: "Low", color: "text-muted-foreground" },
  none: { label: "", color: "text-muted-foreground" },
};

const healthConfig: Record<string, { label: string; dotColor: string }> = {
  on_track: { label: "On Track", dotColor: "bg-emerald-400" },
  at_risk: { label: "At Risk", dotColor: "bg-yellow-400" },
  off_track: { label: "Off Track", dotColor: "bg-red-400" },
};

interface Project {
  id: bigint;
  name: string;
  slug: string;
  description: string;
  status: string;
  priority: string;
  health: string;
  leadName: string;
  leadAvatarUrl: string;
  targetDate?: string;
  icon: string;
  color: string;
}

export default function ProjectsPage() {
  const { t } = useI18n();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchProjects() {
      try {
        const res = await projectClient.listProjects({});
        setProjects(
          res.projects.map((p) => ({
            id: p.id,
            name: p.name,
            slug: p.slug,
            description: p.description,
            status: p.status || "backlog",
            priority: p.priority || "none",
            health: p.health || "",
            leadName: p.leadName || "",
            leadAvatarUrl: p.leadAvatarUrl || "",
            targetDate: p.targetDate || "",
            icon: p.icon || "",
            color: p.color || "",
          })),
        );
      } catch (err) {
        showError(t("toast.projectLoadFailed"), err);
      } finally {
        setLoading(false);
      }
    }
    fetchProjects();
  }, []);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <main className="mx-auto max-w-6xl px-6 py-10">
        <div className="mb-8 flex items-center justify-between">
          <h1 className="text-2xl font-semibold tracking-tight">{t("projects.title")}</h1>
          <Link href="/projects/new">
            <Button size="sm" className="cursor-pointer">
              <Plus className="mr-1.5 h-4 w-4" />
              {t("projects.newProject")}
            </Button>
          </Link>
        </div>

        {projects.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/40 bg-card/30 py-20">
            <div className="rounded-2xl bg-primary/5 p-5 mb-5">
              <Folder className="h-10 w-10 text-primary/40" strokeWidth={1} />
            </div>
            <p className="text-sm font-medium text-foreground/70">{t("projects.noProjects")}</p>
            <p className="mt-1 text-xs text-muted-foreground">{t("projects.createFirst")}</p>
            <Link href="/projects/new" className="mt-6">
              <Button className="cursor-pointer">{t("projects.createProject")}</Button>
            </Link>
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {projects.map((project) => {
              const status = statusConfig[project.status] ?? statusConfig.backlog;
              const priority = priorityConfig[project.priority] ?? priorityConfig.none;
              const health = project.health ? healthConfig[project.health] : null;

              return (
                <Link key={String(project.id)} href={`/projects/${project.slug}`}>
                  <div className="group relative flex flex-col rounded-xl border border-border/50 bg-card p-4 transition-all duration-200 hover:border-primary/30 hover:bg-card/80 cursor-pointer">
                    {/* Header: Icon + Name + Status */}
                    <div className="flex items-start gap-3 mb-2">
                      <div
                        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-sm font-medium"
                        style={{
                          backgroundColor: project.color ? `${project.color}15` : "hsl(var(--primary) / 0.08)",
                          color: project.color || "hsl(var(--primary))",
                        }}
                      >
                        <ProjectIcon name={project.icon} fallback={project.name.charAt(0).toUpperCase()} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <h3 className="text-sm font-semibold leading-tight truncate">
                          {project.name}
                        </h3>
                        {project.description && (
                          <p className="mt-0.5 text-xs text-muted-foreground line-clamp-1">
                            {project.description}
                          </p>
                        )}
                      </div>
                    </div>

                    {/* Metadata row */}
                    <div className="mt-auto flex items-center gap-3 pt-3 border-t border-border/30">
                      {/* Status */}
                      <div className="flex items-center gap-1.5">
                        <span className={`h-2 w-2 rounded-full ${status.dotColor}`} />
                        <span className="text-xs text-muted-foreground">{status.label}</span>
                      </div>

                      {/* Priority */}
                      {priority.label && (
                        <span className={`text-xs ${priority.color}`}>
                          {priority.label}
                        </span>
                      )}

                      {/* Health */}
                      {health && (
                        <div className="flex items-center gap-1.5">
                          <span className={`h-1.5 w-1.5 rounded-full ${health.dotColor}`} />
                          <span className="text-xs text-muted-foreground">{health.label}</span>
                        </div>
                      )}

                      {/* Spacer */}
                      <div className="flex-1" />

                      {/* Lead */}
                      {project.leadName && (
                        <div className="flex items-center gap-1 text-xs text-muted-foreground">
                          <Avatar size="sm" className="size-4">
                            <AvatarImage src={project.leadAvatarUrl} alt={project.leadName} />
                            <AvatarFallback className="bg-primary/10 text-[9px] font-medium text-primary">
                              {project.leadName.charAt(0).toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <span className="max-w-[60px] truncate">{project.leadName}</span>
                        </div>
                      )}

                      {/* Target date */}
                      {project.targetDate && (
                        <div className="flex items-center gap-1 text-xs text-muted-foreground">
                          <Calendar className="h-3 w-3" />
                          <span>{formatDate(project.targetDate)}</span>
                        </div>
                      )}
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </main>
    </div>
  );
}

function ProjectIcon({ name, fallback }: { name: string; fallback: string }) {
  if (!name) return <span>{fallback}</span>;
  // Convert kebab-case to PascalCase: "layers" → "Layers", "arrow-right" → "ArrowRight"
  const pascalName = name
    .split("-")
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join("") as keyof typeof icons;
  const Icon: LucideIcon | undefined = icons[pascalName];
  if (!Icon) return <span>{fallback}</span>;
  return <Icon className="h-4.5 w-4.5" />;
}

function formatDate(dateStr: string): string {
  try {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  } catch {
    return dateStr;
  }
}
