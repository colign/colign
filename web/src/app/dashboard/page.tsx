"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Header } from "@/components/layout/header";
import { useI18n } from "@/lib/i18n";
import { useOrg } from "@/lib/org-context";
import { projectClient } from "@/lib/project";
import { FolderKanban, GitBranch, Eye, CheckCircle2, ChevronRight } from "lucide-react";
import { showError } from "@/lib/toast";

interface Project {
  id: bigint;
  name: string;
  slug: string;
  description: string;
  createdAt?: { seconds: bigint };
}

interface Change {
  id: bigint;
  projectId: bigint;
  name: string;
  stage: string;
  updatedAt?: { seconds: bigint };
}

const stageConfig: Record<string, { label: string; color: string; dot: string }> = {
  draft: { label: "Draft", color: "text-amber-400", dot: "bg-amber-400" },
  design: { label: "Design", color: "text-blue-400", dot: "bg-blue-400" },
  review: { label: "Review", color: "text-violet-400", dot: "bg-violet-400" },
  ready: { label: "Ready", color: "text-emerald-400", dot: "bg-emerald-400" },
};

function timeAgo(seconds: bigint | undefined): string {
  if (!seconds) return "";
  const now = Math.floor(Date.now() / 1000);
  const diff = now - Number(seconds);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default function DashboardPage() {
  const { t } = useI18n();
  const { currentOrg } = useOrg();
  const [projects, setProjects] = useState<Project[]>([]);
  const [allChanges, setAllChanges] = useState<
    { change: Change; projectSlug: string; projectName: string }[]
  >([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const res = await projectClient.listProjects({});
        const projectList = res.projects.map((p) => ({
          id: p.id,
          name: p.name,
          slug: p.slug,
          description: p.description,
          createdAt: p.createdAt ? { seconds: p.createdAt.seconds } : undefined,
        }));
        setProjects(projectList);

        // Load changes for all projects
        const changeResults = await Promise.all(
          projectList.map(async (project) => {
            try {
              const changesRes = await projectClient.listChanges({ projectId: project.id });
              return changesRes.changes.map((c) => ({
                change: {
                  id: c.id,
                  projectId: c.projectId,
                  name: c.name,
                  stage: c.stage,
                  updatedAt: c.updatedAt ? { seconds: c.updatedAt.seconds } : undefined,
                },
                projectSlug: project.slug,
                projectName: project.name,
              }));
            } catch {
              return [];
            }
          }),
        );
        setAllChanges(changeResults.flat());
      } catch (err) {
        showError(t("toast.projectLoadFailed"), err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [currentOrg]);

  const activeChanges = allChanges.filter(
    (c) => c.change.stage === "draft" || c.change.stage === "design",
  );
  const reviewChanges = allChanges.filter((c) => c.change.stage === "review");
  const readyChanges = allChanges.filter((c) => c.change.stage === "ready");

  // Sort by most recently updated
  const recentChanges = [...allChanges]
    .sort(
      (a, b) => Number(b.change.updatedAt?.seconds ?? 0) - Number(a.change.updatedAt?.seconds ?? 0),
    )
    .slice(0, 10);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const statCards = [
    {
      value: projects.length,
      label: t("dashboard.projects"),
      sub: `${allChanges.length} changes`,
      icon: FolderKanban,
      iconColor: "text-primary",
    },
    {
      value: activeChanges.length,
      label: t("dashboard.activeChanges"),
      sub: `${activeChanges.filter((c) => c.change.stage === "draft").length} draft, ${activeChanges.filter((c) => c.change.stage === "design").length} design`,
      icon: GitBranch,
      iconColor: "text-blue-400",
    },
    {
      value: reviewChanges.length,
      label: t("dashboard.inReview"),
      sub:
        reviewChanges.length > 0
          ? `${reviewChanges.map((c) => c.projectName).filter((v, i, a) => a.indexOf(v) === i).length} projects`
          : "No pending reviews",
      icon: Eye,
      iconColor: "text-violet-400",
    },
    {
      value: readyChanges.length,
      label: t("dashboard.ready"),
      sub:
        readyChanges.length > 0
          ? `${readyChanges.map((c) => c.projectName).filter((v, i, a) => a.indexOf(v) === i).length} projects`
          : "All clear",
      icon: CheckCircle2,
      iconColor: "text-emerald-400",
    },
  ];

  return (
    <div className="min-h-screen bg-background">
      <Header breadcrumbs={[{ label: t("dashboard.title") }]} />

      <main className="mx-auto max-w-6xl px-6 pt-8 pb-16">
        {/* Org Name */}
        {currentOrg && <p className="mb-6 text-sm text-muted-foreground">{currentOrg.name}</p>}

        {/* Stat Cards — Paperclip style */}
        <div className="mb-8 grid grid-cols-2 gap-4 lg:grid-cols-4">
          {statCards.map((card) => (
            <div
              key={card.label}
              className="rounded-xl border border-border/40 bg-card/50 px-5 py-4"
            >
              <div className="flex items-start justify-between">
                <span className="text-3xl font-bold tracking-tight">{card.value}</span>
                <card.icon className={`size-5 ${card.iconColor} opacity-60`} />
              </div>
              <p className="mt-1 text-sm font-medium text-foreground/80">{card.label}</p>
              <p className="mt-0.5 text-xs text-muted-foreground">{card.sub}</p>
            </div>
          ))}
        </div>

        {/* Two-column: Recent Activity + Recent Changes */}
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Recent Activity */}
          <div className="rounded-xl border border-border/40 bg-card/50">
            <div className="flex items-center justify-between border-b border-border/30 px-5 py-3">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t("dashboard.recentActivity")}
              </span>
            </div>
            <div className="divide-y divide-border/20">
              {recentChanges.length === 0 ? (
                <div className="px-5 py-10 text-center text-sm text-muted-foreground">
                  {t("dashboard.noActivity")}
                </div>
              ) : (
                recentChanges.map((item) => {
                  const config = stageConfig[item.change.stage] ?? stageConfig.draft;
                  return (
                    <Link
                      key={String(item.change.id)}
                      href={`/projects/${item.projectSlug}/changes/${item.change.id}`}
                    >
                      <div className="flex cursor-pointer items-center justify-between px-5 py-3 transition-colors hover:bg-accent/30">
                        <div className="flex items-center gap-3 min-w-0">
                          <div className={`h-2 w-2 shrink-0 rounded-full ${config.dot}`} />
                          <div className="min-w-0">
                            <p className="truncate text-sm">
                              <span className="font-medium">{item.change.name}</span>
                              <span className="text-muted-foreground">
                                {" "}
                                {t("dashboard.in")} {item.projectName}
                              </span>
                            </p>
                            <p className={`text-xs ${config.color}`}>{config.label}</p>
                          </div>
                        </div>
                        <span className="shrink-0 text-xs text-muted-foreground/60">
                          {timeAgo(item.change.updatedAt?.seconds)}
                        </span>
                      </div>
                    </Link>
                  );
                })
              )}
            </div>
          </div>

          {/* Recent Changes by Project */}
          <div className="rounded-xl border border-border/40 bg-card/50">
            <div className="flex items-center justify-between border-b border-border/30 px-5 py-3">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t("dashboard.recentChanges")}
              </span>
              <Link
                href="/projects"
                className="flex cursor-pointer items-center gap-1 text-xs text-primary hover:text-primary/80 transition-colors"
              >
                {t("dashboard.viewAll")}
                <ChevronRight className="size-3" />
              </Link>
            </div>
            <div className="divide-y divide-border/20">
              {projects.length === 0 ? (
                <div className="px-5 py-10 text-center text-sm text-muted-foreground">
                  {t("dashboard.noChanges")}
                </div>
              ) : (
                projects.map((project) => {
                  const projectChanges = allChanges
                    .filter((c) => String(c.change.projectId) === String(project.id))
                    .slice(0, 3);
                  return (
                    <div key={String(project.id)} className="px-5 py-3">
                      <Link href={`/projects/${project.slug}`}>
                        <div className="mb-2 flex cursor-pointer items-center gap-2 transition-colors hover:text-primary">
                          <FolderKanban className="size-3.5 text-muted-foreground/60" />
                          <span className="text-sm font-medium">{project.name}</span>
                          <span className="text-xs text-muted-foreground/50">
                            {projectChanges.length} changes
                          </span>
                        </div>
                      </Link>
                      {projectChanges.length === 0 ? (
                        <p className="ml-5.5 text-xs text-muted-foreground/40">
                          {t("dashboard.noChanges")}
                        </p>
                      ) : (
                        <div className="ml-1 space-y-1">
                          {projectChanges.map((item) => {
                            const config = stageConfig[item.change.stage] ?? stageConfig.draft;
                            return (
                              <Link
                                key={String(item.change.id)}
                                href={`/projects/${project.slug}/changes/${item.change.id}`}
                              >
                                <div className="flex cursor-pointer items-center justify-between rounded-lg px-3 py-1.5 transition-colors hover:bg-accent/30">
                                  <div className="flex items-center gap-2">
                                    <div className={`h-1.5 w-1.5 rounded-full ${config.dot}`} />
                                    <span className="text-sm text-foreground/80">
                                      {item.change.name}
                                    </span>
                                  </div>
                                  <span className={`text-xs font-medium ${config.color}`}>
                                    {config.label}
                                  </span>
                                </div>
                              </Link>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
