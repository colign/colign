"use client";

import { useState, useEffect, useRef } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Header } from "@/components/layout/header";
import { workflowClient } from "@/lib/workflow";
import { projectClient } from "@/lib/project";
import { orgClient } from "@/lib/organization";
import { isCanonicalProjectRef, toProjectPath } from "@/lib/project-ref";
import { useI18n } from "@/lib/i18n";
import { showError, showSuccess } from "@/lib/toast";
import { AIConfigSettings } from "@/components/settings/ai-config";

type SettingsTab = "general" | "members" | "approval" | "archive" | "ai" | "danger";

const tabs: { id: SettingsTab; labelKey: string }[] = [
  { id: "general", labelKey: "projectSettings.general" },
  { id: "members", labelKey: "projectSettings.members" },
  { id: "approval", labelKey: "projectSettings.approvalPolicy" },
  { id: "archive", labelKey: "projectSettings.archivePolicy" },
  { id: "ai", labelKey: "projectSettings.ai" },
  { id: "danger", labelKey: "projectSettings.dangerZone" },
];

const policyOptions = [
  { value: "owner_one", labelKey: "projectSettings.ownerOne" },
  { value: "editor_two", labelKey: "projectSettings.editorTwo" },
  { value: "all", labelKey: "projectSettings.all" },
  { value: "auto_pass", labelKey: "projectSettings.autoPass" },
];

export default function ProjectSettingsPage() {
  const params = useParams();
  const projectRef = params.slug as string;
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useI18n();

  const initialTab = tabs.some((tab) => tab.id === searchParams.get("tab"))
    ? (searchParams.get("tab") as SettingsTab)
    : "general";
  const [activeTab, setActiveTab] = useState<SettingsTab>(initialTab);
  const [projectId, setProjectId] = useState<bigint | null>(null);
  const [projectSlug, setProjectSlug] = useState("");

  // General
  const [projectName, setProjectName] = useState("");
  const [projectDescription, setProjectDescription] = useState("");
  const [projectIdentifier, setProjectIdentifier] = useState("");

  // Members
  const [inviteQuery, setInviteQuery] = useState("");
  const [inviteRole, setInviteRole] = useState("editor");
  const [inviting, setInviting] = useState(false);
  const [orgMembers, setOrgMembers] = useState<{ userId: bigint; name: string; email: string }[]>([]);
  const [projectMembers, setProjectMembers] = useState<{ name: string; email: string; role: string }[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Approval
  const [policy, setPolicy] = useState("owner_one");
  const [minCount, setMinCount] = useState(1);

  // Archive Policy
  const [archiveMode, setArchiveMode] = useState("manual");
  const [archiveTrigger, setArchiveTrigger] = useState("tasks_done");
  const [archiveDaysDelay, setArchiveDaysDelay] = useState(0);
  const [loadingArchivePolicy, setLoadingArchivePolicy] = useState(false);
  const [savingArchive, setSavingArchive] = useState(false);
  const [savedArchive, setSavedArchive] = useState(false);

  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState("");

  // Load project ID on mount
  useEffect(() => {
    projectClient
      .getProject({ slug: projectRef })
      .then((res) => {
        if (!res.project) {
          router.replace("/projects");
          return;
        }
        if (!isCanonicalProjectRef(projectRef, res.project)) {
          const qs = searchParams.toString();
          router.replace(`${toProjectPath(res.project)}/settings${qs ? `?${qs}` : ""}`);
          return;
        }
        setProjectId(res.project.id);
        setProjectSlug(res.project.slug);
        setProjectName(res.project.name);
        setProjectDescription(res.project.description);
        setProjectIdentifier(res.project.identifier);
        setProjectMembers(
          (res.members || []).map((m) => ({
            name: m.userName,
            email: m.userEmail,
            role: m.role,
          })),
        );
      })
      .catch((err: unknown) => {
        showError(t("toast.projectLoadFailed"), err);
      });
  }, [projectRef, router, t]);

  // Load org members when members tab is active
  useEffect(() => {
    if (activeTab !== "members") return;
    if (orgMembers.length > 0) return;
    orgClient
      .listMembers({})
      .then((res) => {
        setOrgMembers(
          res.members.map((m) => ({ userId: m.userId, name: m.userName, email: m.userEmail })),
        );
      })
      .catch((err: unknown) => {
        showError(t("toast.loadFailed"), err);
      });
  }, [activeTab, orgMembers.length, t]);

  // Close dropdown on outside click
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
      if (!inviteQuery.trim()) return true;
      const q = inviteQuery.toLowerCase();
      return m.name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q);
    })
    .filter((m) => !projectMembers.some((pm) => pm.email === m.email));

  async function handleInviteMember(email: string) {
    if (!projectId) return;
    setInviting(true);
    try {
      await projectClient.inviteMember({
        projectId,
        email,
        role: inviteRole,
      });
      setProjectMembers((prev) => [
        ...prev,
        { name: orgMembers.find((m) => m.email === email)?.name ?? "", email, role: inviteRole },
      ]);
      setInviteQuery("");
      setShowDropdown(false);
      showSuccess(t("projectSettings.invitationSent"));
    } catch (err) {
      showError(t("toast.inviteFailed"), err);
    } finally {
      setInviting(false);
    }
  }

  // Load archive policy when archive tab becomes active
  useEffect(() => {
    if (activeTab !== "archive" || !projectId) return;
    setLoadingArchivePolicy(true);
    projectClient
      .getArchivePolicy({ projectId })
      .then((res) => {
        if (res.policy) {
          setArchiveMode(res.policy.mode || "manual");
          setArchiveTrigger(res.policy.trigger || "tasks_done");
          setArchiveDaysDelay(res.policy.daysDelay ?? 0);
        }
      })
      .catch((err: unknown) => {
        showError(t("toast.loadFailed"), err);
      })
      .finally(() => {
        setLoadingArchivePolicy(false);
      });
  }, [activeTab, projectId]);

  function showSaved(section: string) {
    setSaved(section);
    setTimeout(() => setSaved(""), 2000);
  }

  async function handleSave(section: string) {
    setSaving(true);
    if (section === "general") {
      try {
        await projectClient.updateProject({
          id: projectId ?? BigInt(0),
          projectId: projectId ?? BigInt(0),
          name: projectName,
          description: projectDescription,
          identifier: projectIdentifier.toUpperCase(),
        });
        showSuccess(t("toast.saveSuccess"));
      } catch (err) {
        showError(t("toast.saveFailed"), err);
      }
    }
    if (section === "approval") {
      try {
        await workflowClient.setApprovalPolicy({
          projectId: projectId ?? BigInt(1),
          policy,
          minCount,
        });
        showSuccess(t("toast.saveSuccess"));
      } catch (err) {
        showError(t("toast.saveFailed"), err);
      }
    }
    // TODO: other API calls
    await new Promise((r) => setTimeout(r, 300));
    setSaving(false);
    showSaved(section);
  }

  async function handleSaveArchivePolicy() {
    if (!projectId) return;
    setSavingArchive(true);
    try {
      await projectClient.updateArchivePolicy({
        projectId,
        mode: archiveMode,
        trigger: archiveTrigger,
        daysDelay: archiveDaysDelay,
      });
      setSavedArchive(true);
      setTimeout(() => setSavedArchive(false), 2000);
      showSuccess(t("toast.saveSuccess"));
    } catch (err) {
      showError(t("toast.saveFailed"), err);
    } finally {
      setSavingArchive(false);
    }
  }

  const showDaysInput =
    archiveTrigger === "days_after_approved" || archiveTrigger === "tasks_done_and_days";

  return (
    <div className="min-h-screen">
      <Header
        breadcrumbs={[
          {
            label: "Project",
            href: projectId
              ? toProjectPath({ id: projectId, slug: projectSlug })
              : `/projects/${projectRef}`,
          },
          { label: "Settings" },
        ]}
      />

      <div className="mx-auto flex max-w-5xl gap-8 px-6 py-8">
        {/* Sidebar */}
        <nav className="w-48 shrink-0">
          <ul className="space-y-1">
            {tabs.map((tab) => (
              <li key={tab.id}>
                <button
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full cursor-pointer rounded-lg px-3 py-2 text-left text-sm transition-colors duration-200 ${
                    activeTab === tab.id
                      ? "bg-accent text-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  } ${tab.id === "danger" ? "text-destructive" : ""}`}
                >
                  {t(tab.labelKey)}
                </button>
              </li>
            ))}
          </ul>
        </nav>

        {/* Content */}
        <div className="flex-1 space-y-6">
          {/* General */}
          {activeTab === "general" && (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{t("projectSettings.general")}</CardTitle>
                <CardDescription>{t("projectSettings.projectNameDesc")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="project-name">{t("projects.projectName")}</Label>
                  <Input
                    id="project-name"
                    value={projectName}
                    onChange={(e) => setProjectName(e.target.value)}
                    placeholder="My App"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="project-identifier">Identifier</Label>
                  <p className="text-xs text-muted-foreground">
                    Used for change numbering (e.g. COL-1, COL-2). 1-5 uppercase letters.
                  </p>
                  <Input
                    id="project-identifier"
                    value={projectIdentifier}
                    onChange={(e) =>
                      setProjectIdentifier(
                        e.target.value
                          .toUpperCase()
                          .replace(/[^A-Z0-9]/g, "")
                          .slice(0, 5),
                      )
                    }
                    placeholder="COL"
                    className="w-32 font-mono uppercase"
                    maxLength={5}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="project-desc">{t("projects.description")}</Label>
                  <Input
                    id="project-desc"
                    value={projectDescription}
                    onChange={(e) => setProjectDescription(e.target.value)}
                    placeholder="A brief description"
                  />
                </div>
                <div className="flex items-center gap-3 pt-2">
                  <Button
                    onClick={() => handleSave("general")}
                    disabled={saving}
                    className="cursor-pointer"
                  >
                    {saving ? t("common.saving") : t("common.save")}
                  </Button>
                  {saved === "general" && (
                    <span className="text-sm text-emerald-400">{t("common.saved")}</span>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Members */}
          {activeTab === "members" && (
            <>
              <Card className="border-border/50 overflow-visible">
                <CardHeader>
                  <CardTitle>{t("projectSettings.inviteMember")}</CardTitle>
                  <CardDescription>{t("projectSettings.inviteMemberDesc")}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-2">
                    <div className="relative flex-1" ref={dropdownRef}>
                      <Input
                        value={inviteQuery}
                        onChange={(e) => {
                          setInviteQuery(e.target.value);
                          setShowDropdown(true);
                        }}
                        onFocus={() => setShowDropdown(true)}
                        placeholder={t("projectSettings.searchMemberPlaceholder")}
                        autoComplete="off"
                      />
                      {showDropdown && (
                        <div className="absolute top-full left-0 z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-border bg-popover shadow-lg">
                          {filteredOrgMembers.length === 0 ? (
                            <div className="px-4 py-3 text-center text-sm text-muted-foreground">
                              {orgMembers.length > 0 && orgMembers.length === projectMembers.length
                                ? t("projectSettings.allMembersAdded")
                                : t("projectSettings.noMembersFound")}
                            </div>
                          ) : (
                            filteredOrgMembers.map((m) => (
                              <button
                                key={String(m.userId)}
                                onClick={() => handleInviteMember(m.email)}
                                disabled={inviting}
                                className="flex w-full cursor-pointer items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-accent disabled:opacity-50"
                              >
                                <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-medium">
                                  {(m.name?.[0] ?? m.email[0]).toUpperCase()}
                                </div>
                                <div className="min-w-0 flex-1">
                                  <p className="truncate text-sm font-medium">{m.name || "\u2014"}</p>
                                  <p className="truncate text-xs text-muted-foreground">{m.email}</p>
                                </div>
                                <UserPlus className="size-4 shrink-0 text-muted-foreground" />
                              </button>
                            ))
                          )}
                        </div>
                      )}
                    </div>
                    <Select value={inviteRole} onValueChange={(v) => v && setInviteRole(v)}>
                      <SelectTrigger className="w-32 cursor-pointer">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="editor" className="cursor-pointer">
                          Editor
                        </SelectItem>
                        <SelectItem value="viewer" className="cursor-pointer">
                          Viewer
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-border/50">
                <CardHeader>
                  <CardTitle>{t("projectSettings.members")}</CardTitle>
                  <CardDescription>{t("projectSettings.membersDesc")}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {projectMembers.map((member) => (
                      <div
                        key={member.email}
                        className="flex items-center justify-between rounded-lg border border-border/50 p-3"
                      >
                        <div className="flex items-center gap-3">
                          <div className="flex size-8 items-center justify-center rounded-full bg-accent text-xs font-medium">
                            {(member.name?.[0] ?? member.email[0]).toUpperCase()}
                          </div>
                          <div>
                            <p className="text-sm font-medium">{member.name || "\u2014"}</p>
                            <p className="text-xs text-muted-foreground">{member.email}</p>
                          </div>
                        </div>
                        <span className="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary capitalize">
                          {member.role}
                        </span>
                      </div>
                    ))}
                    {projectMembers.length === 0 && (
                      <div className="px-4 py-8 text-center text-sm text-muted-foreground">
                        {t("projectSettings.membersDesc")}
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </>
          )}

          {/* Approval Policy */}
          {activeTab === "approval" && (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{t("projectSettings.approvalPolicy")}</CardTitle>
                <CardDescription>{t("projectSettings.approvalPolicyDesc")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-5">
                <div className="space-y-2">
                  <Label>{t("projectSettings.policyType")}</Label>
                  <Select value={policy} onValueChange={(v) => v && setPolicy(v)}>
                    <SelectTrigger className="cursor-pointer">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {policyOptions.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value} className="cursor-pointer">
                          {t(opt.labelKey)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                {policy !== "auto_pass" && policy !== "all" && (
                  <div className="space-y-2">
                    <Label>{t("projectSettings.minApprovals")}</Label>
                    <Input
                      type="number"
                      min={1}
                      max={10}
                      value={minCount}
                      onChange={(e) => setMinCount(Number(e.target.value))}
                    />
                  </div>
                )}
                <div className="flex items-center gap-3 pt-2">
                  <Button
                    onClick={() => handleSave("approval")}
                    disabled={saving}
                    className="cursor-pointer"
                  >
                    {saving ? t("common.saving") : t("projectSettings.savePolicy")}
                  </Button>
                  {saved === "approval" && (
                    <span className="text-sm text-emerald-400">{t("common.saved")}</span>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Archive Policy */}
          {activeTab === "archive" && (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{t("projectSettings.archivePolicy")}</CardTitle>
                <CardDescription>{t("projectSettings.archivePolicyDesc")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-5">
                {loadingArchivePolicy ? (
                  <div className="flex items-center justify-center py-8">
                    <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                  </div>
                ) : (
                  <>
                    <div className="space-y-2">
                      <Label>{t("projectSettings.archiveMode")}</Label>
                      <Select value={archiveMode} onValueChange={(v) => v && setArchiveMode(v)}>
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

                    {archiveMode === "auto" && (
                      <div className="space-y-2">
                        <Label>{t("projectSettings.archiveTrigger")}</Label>
                        <Select
                          value={archiveTrigger}
                          onValueChange={(v) => v && setArchiveTrigger(v)}
                        >
                          <SelectTrigger className="w-full cursor-pointer">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent className="min-w-[280px]">
                            <SelectItem value="tasks_done" className="cursor-pointer">
                              {t("projectSettings.archiveTriggerTasksDone")}
                            </SelectItem>
                            <SelectItem value="days_after_approved" className="cursor-pointer">
                              {t("projectSettings.archiveTriggerDaysAfterApproved")}
                            </SelectItem>
                            <SelectItem value="tasks_done_and_days" className="cursor-pointer">
                              {t("projectSettings.archiveTriggerTasksDoneAndDays")}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    )}

                    {archiveMode === "auto" && showDaysInput && (
                      <div className="space-y-2">
                        <Label>{t("projectSettings.archiveDaysDelay")}</Label>
                        <Input
                          type="number"
                          min={0}
                          max={365}
                          value={archiveDaysDelay}
                          onChange={(e) => setArchiveDaysDelay(Number(e.target.value))}
                          className="w-32"
                        />
                      </div>
                    )}

                    <div className="flex items-center gap-3 pt-2">
                      <Button
                        onClick={handleSaveArchivePolicy}
                        disabled={savingArchive || !projectId}
                        className="cursor-pointer"
                      >
                        {savingArchive
                          ? t("common.saving")
                          : t("projectSettings.saveArchivePolicy")}
                      </Button>
                      {savedArchive && (
                        <span className="text-sm text-emerald-400">{t("common.saved")}</span>
                      )}
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          )}

          {/* AI Configuration */}
          {activeTab === "ai" && projectId && <AIConfigSettings projectId={projectId} />}

          {/* Danger Zone */}
          {activeTab === "danger" && (
            <Card className="border-destructive/30">
              <CardHeader>
                <CardTitle className="text-destructive">
                  {t("projectSettings.dangerZone")}
                </CardTitle>
                <CardDescription>{t("projectSettings.deleteConfirm")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between rounded-lg border border-destructive/20 p-4">
                  <div>
                    <p className="text-sm font-medium">{t("projectSettings.deleteProject")}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("projectSettings.deleteProjectDesc")}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    className="cursor-pointer border-destructive/50 text-destructive hover:bg-destructive/10"
                    onClick={() => {
                      if (confirm(t("projectSettings.deleteConfirm"))) {
                        // TODO: API call
                        router.push("/projects");
                      }
                    }}
                  >
                    {t("projectSettings.deleteProject")}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
