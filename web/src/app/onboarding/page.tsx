"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { orgClient } from "@/lib/organization";
import { showError } from "@/lib/toast";
import { useOrg } from "@/lib/org-context";
import { useI18n } from "@/lib/i18n";

export default function OnboardingPage() {
  const router = useRouter();
  const { t } = useI18n();
  const { currentOrg, loading: orgLoading, refresh } = useOrg();
  const [step, setStep] = useState(1);
  const [workspaceName, setWorkspaceName] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [retryCount, setRetryCount] = useState(0);

  // Wait for org context to load after signup
  useEffect(() => {
    if (!orgLoading && !currentOrg && retryCount < 10) {
      const timer = setTimeout(() => {
        refresh();
        setRetryCount((c) => c + 1);
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [orgLoading, currentOrg, retryCount, refresh]);

  async function handleSetupWorkspace(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!workspaceName.trim()) return;
    if (!currentOrg) {
      setError("Organization not loaded. Please refresh and try again.");
      return;
    }
    setSaving(true);
    try {
      await orgClient.updateOrganization({
        id: currentOrg.id,
        name: workspaceName.trim(),
      });
      await refresh();
      setStep(2);
    } catch (err: unknown) {
      showError("Failed to create workspace", err);
      setError(err instanceof Error ? err.message : "Failed to create workspace");
    } finally {
      setSaving(false);
    }
  }

  function handleFinish() {
    router.push("/projects");
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-lg">
        {/* Progress */}
        <div className="mb-8 flex items-center justify-center gap-2">
          {[1, 2].map((s) => (
            <div
              key={s}
              className={`h-1.5 w-12 rounded-full transition-colors ${
                s <= step ? "bg-primary" : "bg-border"
              }`}
            />
          ))}
        </div>

        {step === 1 && (
          <div className="space-y-6">
            <div className="text-center">
              <h1 className="text-2xl font-bold tracking-tight">{t("onboarding.createWorkspace")}</h1>
              <p className="mt-2 text-sm text-muted-foreground">
                {t("onboarding.createWorkspaceDesc")}
              </p>
            </div>

            <form onSubmit={handleSetupWorkspace} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="workspace-name">{t("onboarding.workspaceName")}</Label>
                <Input
                  id="workspace-name"
                  value={workspaceName}
                  onChange={(e) => setWorkspaceName(e.target.value)}
                  placeholder="Acme Inc."
                  required
                  autoFocus
                  className="h-11 text-base"
                />
                <p className="text-xs text-muted-foreground">
                  {t("onboarding.changeInSettings")}
                </p>
              </div>

              {error && <p className="text-sm text-destructive">{error}</p>}

              <Button
                type="submit"
                className="w-full cursor-pointer h-11"
                disabled={saving || !workspaceName.trim() || orgLoading}
              >
                {orgLoading ? t("common.loading") : saving ? t("common.settingUp") : t("common.continue")}
              </Button>
            </form>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-6 text-center">
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10">
              <svg
                className="h-8 w-8 text-primary"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>

            <div>
              <h1 className="text-2xl font-bold tracking-tight">{t("onboarding.allSet")}</h1>
              <p className="mt-2 text-sm text-muted-foreground">
                {t("onboarding.workspaceReady")}
              </p>
            </div>

            <Button onClick={handleFinish} className="cursor-pointer h-11 px-8">
              {t("onboarding.goToProjects")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
