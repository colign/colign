"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { orgClient } from "@/lib/organization";
import { showError } from "@/lib/toast";
import { useOrg } from "@/lib/org-context";
import { useI18n } from "@/lib/i18n";

interface CreateOrganizationDialogProps {
  triggerClassName?: string;
  compact?: boolean;
  onCreated?: () => void;
}

export function CreateOrganizationDialog({
  triggerClassName,
  compact = false,
  onCreated,
}: CreateOrganizationDialogProps) {
  const { refresh, switchOrg } = useOrg();
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) return;

    setSaving(true);
    setError("");
    try {
      const res = await orgClient.createOrganization({ name: trimmedName });
      await refresh();
      setOpen(false);
      setName("");
      onCreated?.();
      await switchOrg(res.organization!.id);
    } catch (err: unknown) {
      showError("Failed to create organization", err);
      setError(err instanceof Error ? err.message : "Failed to create organization");
      setSaving(false);
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={triggerClassName}
      >
        <Plus className="h-3.5 w-3.5" />
        {!compact && <span>{t("org.newOrganization")}</span>}
      </button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("org.createOrganization")}</DialogTitle>
            <DialogDescription>
              {t("org.createOrganizationDesc")}
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="organization-name">{t("settings.organizationName")}</Label>
              <Input
                id="organization-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Acme Inc."
                autoFocus
                disabled={saving}
              />
            </div>

            {error && <p className="text-sm text-destructive">{error}</p>}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={saving}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={saving || !name.trim()}>
                {saving ? t("common.creating") : t("org.createOrganization")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
