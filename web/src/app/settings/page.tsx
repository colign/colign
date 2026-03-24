"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useRouter } from "next/navigation";
import { createClient } from "@connectrpc/connect";
import { useI18n } from "@/lib/i18n";
import { apiTokenClient } from "@/lib/apitoken";
import type { ApiToken } from "@/gen/proto/apitoken/v1/apitoken_pb";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Header } from "@/components/layout/header";
import { clearTokens } from "@/lib/auth";
import { transport } from "@/lib/connect";
import { OrgMembers } from "@/components/settings/org-members";
import { showError, showSuccess } from "@/lib/toast";

type SettingsTab = "profile" | "account" | "organization" | "ai" | "appearance" | "notifications";

const tabIcons: Record<SettingsTab, string> = {
  profile:
    "M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z",
  account:
    "M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z",
  organization:
    "M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z",
  ai: "M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z",
  appearance:
    "M4.098 19.902a3.75 3.75 0 005.304 0l6.401-6.402M6.75 21A3.75 3.75 0 013 17.25V4.125C3 3.504 3.504 3 4.125 3h5.25c.621 0 1.125.504 1.125 1.125v4.072M6.75 21a3.75 3.75 0 003.75-3.75V8.197M6.75 21h13.125c.621 0 1.125-.504 1.125-1.125v-5.25c0-.621-.504-1.125-1.125-1.125h-4.072M10.5 8.197l2.88-2.88c.438-.439 1.15-.439 1.59 0l3.712 3.713c.44.44.44 1.152 0 1.59l-2.879 2.88M6.75 17.25h.008v.008H6.75v-.008z",
  notifications:
    "M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0",
};

const tabI18nKeys: Record<SettingsTab, string> = {
  profile: "settings.profile",
  account: "settings.account",
  organization: "settings.organization",
  ai: "settings.aiApiKeys",
  appearance: "settings.appearance",
  notifications: "settings.notifications",
};

const meClient = createClient(AuthService, transport);

export default function SettingsPage() {
  const router = useRouter();
  const { locale, setLocale, t } = useI18n();
  const [activeTab, setActiveTab] = useState<SettingsTab>("profile");

  // Profile state
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [avatarError, setAvatarError] = useState("");
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement>(null);

  // Account state
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");

  // AI state
  const [claudeApiKey, setClaudeApiKey] = useState("");
  const [showApiKey, setShowApiKey] = useState(false);

  // API Token state
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [newTokenName, setNewTokenName] = useState("");
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [loadingTokens, setLoadingTokens] = useState(false);
  const [creatingToken, setCreatingToken] = useState(false);
  const [copied, setCopied] = useState(false);

  // Appearance state
  const [theme, setTheme] = useState("dark");

  // Notifications state
  const [emailNotifications, setEmailNotifications] = useState(true);
  const [commentNotifications, setCommentNotifications] = useState(true);
  const [reviewNotifications, setReviewNotifications] = useState(true);
  const [stageNotifications, setStageNotifications] = useState(true);

  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState("");

  function showSaved(section: string) {
    setSaved(section);
    setTimeout(() => setSaved(""), 2000);
  }

  const loadTokens = useCallback(async () => {
    setLoadingTokens(true);
    try {
      const res = await apiTokenClient.listApiTokens({});
      setTokens(res.tokens);
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    } finally {
      setLoadingTokens(false);
    }
  }, []);

  useEffect(() => {
    meClient
      .me({})
      .then((res) => {
        setName(res.name);
        setEmail(res.email);
        setAvatarUrl(res.avatarUrl || "");
      })
      .catch((err: unknown) => { showError(t("toast.loadFailed"), err); });
  }, []);

  useEffect(() => {
    if (activeTab === "ai") {
      loadTokens();
    }
  }, [activeTab, loadTokens]);

  async function handleCreateToken() {
    if (!newTokenName.trim()) return;
    setCreatingToken(true);
    try {
      const res = await apiTokenClient.createApiToken({ name: newTokenName });
      setCreatedToken(res.rawToken);
      setNewTokenName("");
      setCopied(false);
      loadTokens();
    } catch (err) {
      showError(t("toast.createFailed"), err);
    } finally {
      setCreatingToken(false);
    }
  }

  async function handleDeleteToken(id: bigint) {
    try {
      await apiTokenClient.deleteApiToken({ id });
      loadTokens();
    } catch (err) {
      showError(t("toast.deleteFailed"), err);
    }
  }

  function handleCopyToken() {
    if (createdToken) {
      navigator.clipboard.writeText(createdToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  async function handleSave(section: string) {
    setSaving(true);
    if (section === "profile") {
      try {
        const res = await meClient.updateProfile({ name, avatarUrl });
        setName(res.name);
        setEmail(res.email);
        setAvatarUrl(res.avatarUrl || "");
        window.dispatchEvent(
          new CustomEvent("colign:profile-updated", {
            detail: {
              name: res.name,
              email: res.email,
              avatarUrl: res.avatarUrl || "",
            },
          }),
        );
        showSuccess(t("toast.saveSuccess"));
      } catch (err) {
        showError(t("toast.saveFailed"), err);
      } finally {
        setSaving(false);
      }
      showSaved(section);
      return;
    }

    await new Promise((r) => setTimeout(r, 300));
    setSaving(false);
    showSaved(section);
  }

  async function handleAvatarFile(file: File) {
    if (!file.type.startsWith("image/")) {
      setAvatarError("Please select an image file.");
      return;
    }

    setAvatarError("");
    setUploadingAvatar(true);
    try {
      const compressed = await fileToAvatarDataUrl(file);
      setAvatarUrl(compressed);
    } catch (err) {
      setAvatarError("Could not process that image.");
      showError(t("toast.saveFailed"), err);
    } finally {
      setUploadingAvatar(false);
    }
  }

  return (
    <div className="min-h-screen">
      <Header breadcrumbs={[{ label: t("settings.title") }]} />

      <div className="mx-auto flex max-w-5xl gap-8 px-6 py-8">
        {/* Sidebar */}
        <nav className="w-56 shrink-0">
          <ul className="space-y-1">
            {(Object.keys(tabIcons) as SettingsTab[]).map((tabId) => (
              <li key={tabId}>
                <button
                  onClick={() => setActiveTab(tabId)}
                  className={`flex w-full cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors duration-200 ${
                    activeTab === tabId
                      ? "bg-accent text-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  }`}
                >
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={1.5}
                      d={tabIcons[tabId]}
                    />
                  </svg>
                  {t(tabI18nKeys[tabId])}
                </button>
              </li>
            ))}
          </ul>

          <Separator className="my-4" />

          <button
            onClick={() => {
              clearTokens();
              router.push("/auth");
            }}
            className="flex w-full cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-sm text-destructive transition-colors duration-200 hover:bg-destructive/10"
          >
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9"
              />
            </svg>
            {t("common.signOut")}
          </button>
        </nav>

        {/* Content */}
        <div className="flex-1 space-y-6">
          {/* Profile */}
          {activeTab === "profile" && (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{t("settings.profile")}</CardTitle>
                <CardDescription>{t("settings.profileDesc")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center gap-4">
                  <Avatar size="lg" className="size-16">
                    <AvatarImage src={avatarUrl} alt={name || email} />
                    <AvatarFallback className="bg-primary/10 text-lg font-semibold text-primary">
                      {(name || email || "?").charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 flex-1 space-y-2">
                    <Label>Profile Photo</Label>
                    <input
                      ref={avatarInputRef}
                      type="file"
                      accept="image/*"
                      className="hidden"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          void handleAvatarFile(file);
                        }
                        e.target.value = "";
                      }}
                    />
                    <div className="flex flex-wrap items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        className="cursor-pointer"
                        onClick={() => avatarInputRef.current?.click()}
                        disabled={uploadingAvatar}
                      >
                        {uploadingAvatar ? "Uploading..." : avatarUrl ? "Change photo" : "Upload photo"}
                      </Button>
                      {avatarUrl && (
                        <Button
                          type="button"
                          variant="ghost"
                          className="cursor-pointer text-muted-foreground"
                          onClick={() => {
                            setAvatarUrl("");
                            setAvatarError("");
                          }}
                        >
                          Remove
                        </Button>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      PNG, JPG, GIF, or WebP. The image is resized automatically.
                    </p>
                    {avatarError && <p className="text-xs text-destructive">{avatarError}</p>}
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="name">{t("auth.name")}</Label>
                  <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="email">{t("auth.email")}</Label>
                  <Input id="email" value={email} disabled className="text-muted-foreground" />
                  <p className="text-xs text-muted-foreground">{t("settings.emailCannotChange")}</p>
                </div>
                <div className="flex items-center gap-3 pt-2">
                  <Button
                    onClick={() => handleSave("profile")}
                    disabled={saving}
                    className="cursor-pointer"
                  >
                    {saving ? t("common.saving") : t("common.save")}
                  </Button>
                  {saved === "profile" && <span className="text-sm text-emerald-400">Saved</span>}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Account */}
          {activeTab === "account" && (
            <>
              <Card className="border-border/50">
                <CardHeader>
                  <CardTitle>Change Password</CardTitle>
                  <CardDescription>Update your password</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="current-password">Current Password</Label>
                    <Input
                      id="current-password"
                      type="password"
                      value={currentPassword}
                      onChange={(e) => setCurrentPassword(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="new-password">New Password</Label>
                    <Input
                      id="new-password"
                      type="password"
                      value={newPassword}
                      onChange={(e) => setNewPassword(e.target.value)}
                      placeholder="Min 8 characters"
                      minLength={8}
                    />
                  </div>
                  <div className="flex items-center gap-3 pt-2">
                    <Button
                      onClick={() => handleSave("password")}
                      disabled={saving || !currentPassword || !newPassword}
                      className="cursor-pointer"
                    >
                      Update Password
                    </Button>
                    {saved === "password" && (
                      <span className="text-sm text-emerald-400">Updated</span>
                    )}
                  </div>
                </CardContent>
              </Card>

              <Card className="border-border/50">
                <CardHeader>
                  <CardTitle>Connected Accounts</CardTitle>
                  <CardDescription>OAuth providers linked to your account</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex items-center justify-between rounded-lg border border-border/50 p-3">
                    <div className="flex items-center gap-3">
                      <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                      </svg>
                      <span className="text-sm font-medium">GitHub</span>
                    </div>
                    <Button variant="outline" size="sm" className="cursor-pointer">
                      Connect
                    </Button>
                  </div>
                  <div className="flex items-center justify-between rounded-lg border border-border/50 p-3">
                    <div className="flex items-center gap-3">
                      <svg className="h-5 w-5" viewBox="0 0 24 24">
                        <path
                          d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
                          fill="#4285F4"
                        />
                        <path
                          d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                          fill="#34A853"
                        />
                        <path
                          d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                          fill="#FBBC05"
                        />
                        <path
                          d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                          fill="#EA4335"
                        />
                      </svg>
                      <span className="text-sm font-medium">Google</span>
                    </div>
                    <Button variant="outline" size="sm" className="cursor-pointer">
                      Connect
                    </Button>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-destructive/30">
                <CardHeader>
                  <CardTitle className="text-destructive">Danger Zone</CardTitle>
                  <CardDescription>Irreversible actions</CardDescription>
                </CardHeader>
                <CardContent>
                  <Button
                    variant="outline"
                    className="cursor-pointer border-destructive/50 text-destructive hover:bg-destructive/10"
                  >
                    Delete Account
                  </Button>
                </CardContent>
              </Card>
            </>
          )}

          {/* Organization */}
          {activeTab === "organization" && <OrgMembers />}

          {/* AI & API Keys */}
          {activeTab === "ai" && (
            <>
              <Card className="border-border/50">
                <CardHeader>
                  <CardTitle>AI & API Keys</CardTitle>
                  <CardDescription>
                    Configure your own API key for AI features. Without a key, AI features use the
                    platform&apos;s shared quota.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="claude-key">Claude API Key</Label>
                    <div className="flex gap-2">
                      <Input
                        id="claude-key"
                        type={showApiKey ? "text" : "password"}
                        value={claudeApiKey}
                        onChange={(e) => setClaudeApiKey(e.target.value)}
                        placeholder="sk-ant-..."
                        className="flex-1"
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowApiKey(!showApiKey)}
                        className="cursor-pointer text-muted-foreground"
                      >
                        {showApiKey ? "Hide" : "Show"}
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Your key is encrypted and stored securely. It&apos;s only used for your
                      requests.
                    </p>
                  </div>
                  <div className="flex items-center gap-3 pt-2">
                    <Button
                      onClick={() => handleSave("ai")}
                      disabled={saving}
                      className="cursor-pointer"
                    >
                      {saving ? "Saving..." : "Save API Key"}
                    </Button>
                    {saved === "ai" && <span className="text-sm text-emerald-400">Saved</span>}
                  </div>
                </CardContent>
              </Card>

              <Card className="border-border/50">
                <CardHeader>
                  <CardTitle>{t("settings.apiTokens")}</CardTitle>
                  <CardDescription>{t("settings.apiTokensDesc")}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {/* Create token form */}
                  <div className="flex gap-2">
                    <Input
                      value={newTokenName}
                      onChange={(e) => setNewTokenName(e.target.value)}
                      placeholder={t("settings.tokenNamePlaceholder")}
                      onKeyDown={(e) => e.key === "Enter" && handleCreateToken()}
                      className="flex-1"
                    />
                    <Button
                      onClick={handleCreateToken}
                      disabled={!newTokenName.trim() || creatingToken}
                      className="cursor-pointer"
                    >
                      {creatingToken ? t("common.loading") : t("settings.generateToken")}
                    </Button>
                  </div>

                  {/* Created token display */}
                  {createdToken && (
                    <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4">
                      <p className="mb-2 text-sm font-medium text-emerald-400">
                        {t("settings.tokenCreatedWarning")}
                      </p>
                      <div className="flex items-center gap-2">
                        <code className="flex-1 overflow-x-auto rounded bg-muted px-3 py-2 font-mono text-xs">
                          {createdToken}
                        </code>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={handleCopyToken}
                          className="cursor-pointer"
                        >
                          {copied ? t("common.saved") : "Copy"}
                        </Button>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="mt-2 cursor-pointer text-muted-foreground"
                        onClick={() => setCreatedToken(null)}
                      >
                        {t("settings.dismissToken")}
                      </Button>
                    </div>
                  )}

                  {/* Token list */}
                  {loadingTokens ? (
                    <p className="text-sm text-muted-foreground">{t("common.loading")}</p>
                  ) : tokens.length === 0 ? (
                    <p className="text-sm text-muted-foreground">{t("settings.noTokens")}</p>
                  ) : (
                    <div className="space-y-2">
                      {tokens.map((token) => (
                        <div
                          key={String(token.id)}
                          className="flex items-center justify-between rounded-lg border border-border/50 p-3"
                        >
                          <div>
                            <p className="text-sm font-medium">{token.name}</p>
                            <p className="text-xs text-muted-foreground">
                              {token.prefix}...
                              {" \u00B7 "}
                              {t("settings.tokenCreated")}{" "}
                              {token.createdAt
                                ? new Date(
                                    Number(token.createdAt.seconds) * 1000,
                                  ).toLocaleDateString()
                                : ""}
                              {" \u00B7 "}
                              {token.lastUsedAt
                                ? `${t("settings.tokenLastUsed")} ${new Date(Number(token.lastUsedAt.seconds) * 1000).toLocaleDateString()}`
                                : t("settings.tokenNeverUsed")}
                            </p>
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="cursor-pointer text-destructive hover:bg-destructive/10"
                            onClick={() => handleDeleteToken(token.id)}
                          >
                            {t("common.delete")}
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* MCP setup guide */}
                  <Separator />
                  <div className="rounded-lg border border-border/50 bg-muted/30 p-4">
                    <p className="mb-2 text-sm font-medium">{t("settings.mcpSetupGuide")}</p>
                    <pre className="overflow-x-auto text-xs text-muted-foreground">
                      {`{
  "mcpServers": {
    "colign": {
      "url": "https://app.colign.dev/mcp",
      "headers": {
        "Authorization": "Bearer col_your_token_here"
      }
    }
  }
}`}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </>
          )}

          {/* Appearance */}
          {activeTab === "appearance" && (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{t("settings.appearance")}</CardTitle>
                <CardDescription>{t("settings.customizeAppearance")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-2">
                  <Label>{t("settings.theme")}</Label>
                  <Select value={theme} onValueChange={setTheme}>
                    <SelectTrigger className="cursor-pointer">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="dark" className="cursor-pointer">
                        {t("settings.dark")}
                      </SelectItem>
                      <SelectItem value="light" className="cursor-pointer">
                        {t("settings.light")}
                      </SelectItem>
                      <SelectItem value="system" className="cursor-pointer">
                        {t("settings.system")}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>{t("settings.language")}</Label>
                  <Select value={locale} onValueChange={(v) => setLocale(v as "en" | "ko")}>
                    <SelectTrigger className="cursor-pointer">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="en" className="cursor-pointer">
                        English
                      </SelectItem>
                      <SelectItem value="ko" className="cursor-pointer">
                        한국어
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-center gap-3 pt-2">
                  <Button
                    onClick={() => handleSave("appearance")}
                    disabled={saving}
                    className="cursor-pointer"
                  >
                    {saving ? t("common.saving") : t("common.save")}
                  </Button>
                  {saved === "appearance" && (
                    <span className="text-sm text-emerald-400">{t("common.saved")}</span>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Notifications */}
          {activeTab === "notifications" && (
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{t("settings.notifications")}</CardTitle>
                <CardDescription>{t("settings.chooseNotifications")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-5">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">{t("settings.emailNotifications")}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("settings.emailNotificationsDesc")}
                    </p>
                  </div>
                  <Switch
                    checked={emailNotifications}
                    onCheckedChange={setEmailNotifications}
                    className="cursor-pointer"
                  />
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">{t("settings.comments")}</p>
                    <p className="text-xs text-muted-foreground">{t("settings.commentsDesc")}</p>
                  </div>
                  <Switch
                    checked={commentNotifications}
                    onCheckedChange={setCommentNotifications}
                    className="cursor-pointer"
                  />
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">{t("settings.reviewRequests")}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("settings.reviewRequestsDesc")}
                    </p>
                  </div>
                  <Switch
                    checked={reviewNotifications}
                    onCheckedChange={setReviewNotifications}
                    className="cursor-pointer"
                  />
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">{t("settings.stageChanges")}</p>
                    <p className="text-xs text-muted-foreground">
                      {t("settings.stageChangesDesc")}
                    </p>
                  </div>
                  <Switch
                    checked={stageNotifications}
                    onCheckedChange={setStageNotifications}
                    className="cursor-pointer"
                  />
                </div>
                <div className="flex items-center gap-3 pt-2">
                  <Button
                    onClick={() => handleSave("notifications")}
                    disabled={saving}
                    className="cursor-pointer"
                  >
                    {saving ? t("common.saving") : t("common.save")}
                  </Button>
                  {saved === "notifications" && (
                    <span className="text-sm text-emerald-400">{t("common.saved")}</span>
                  )}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

async function fileToAvatarDataUrl(file: File): Promise<string> {
  const imageUrl = URL.createObjectURL(file);

  try {
    const img = await loadImage(imageUrl);
    const maxSize = 256;
    const scale = Math.min(maxSize / img.width, maxSize / img.height, 1);
    const width = Math.max(1, Math.round(img.width * scale));
    const height = Math.max(1, Math.round(img.height * scale));

    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;

    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new Error("Canvas unavailable");
    }

    ctx.drawImage(img, 0, 0, width, height);
    return canvas.toDataURL("image/webp", 0.9);
  } finally {
    URL.revokeObjectURL(imageUrl);
  }
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error("Image load failed"));
    img.src = src;
  });
}
