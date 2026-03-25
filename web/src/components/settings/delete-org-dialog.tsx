"use client";

import { useState } from "react";
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
import { useOrg } from "@/lib/org-context";
import { useI18n } from "@/lib/i18n";
import { showError } from "@/lib/toast";

interface DeleteOrgDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgName: string;
  orgId: bigint;
}

export function DeleteOrgDialog({ open, onOpenChange, orgName, orgId }: DeleteOrgDialogProps) {
  const { switchOrg } = useOrg();
  const { t } = useI18n();
  const [confirmName, setConfirmName] = useState("");
  const [deleting, setDeleting] = useState(false);

  const nameMatches = confirmName === orgName;

  async function handleDelete() {
    if (!nameMatches) return;
    setDeleting(true);
    try {
      const res = await orgClient.deleteOrganization({ organizationId: orgId });
      await switchOrg(res.nextOrganizationId);
    } catch (err) {
      showError(t("toast.deleteFailed"), err);
      setDeleting(false);
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) {
      setConfirmName("");
      setDeleting(false);
    }
    onOpenChange(nextOpen);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-destructive">{t("org.deleteOrg")}</DialogTitle>
          <DialogDescription>
            {t("org.deleteOrgDesc")}
          </DialogDescription>
        </DialogHeader>

        <ul className="space-y-1 text-sm text-muted-foreground">
          <li className="flex items-start gap-2">
            <span className="mt-0.5 text-destructive">&#x2022;</span>
            {t("org.deleteOrgConsequences1")}
          </li>
          <li className="flex items-start gap-2">
            <span className="mt-0.5 text-destructive">&#x2022;</span>
            {t("org.deleteOrgConsequences2")}
          </li>
          <li className="flex items-start gap-2">
            <span className="mt-0.5 text-destructive">&#x2022;</span>
            {t("org.deleteOrgConsequences3")}
          </li>
        </ul>

        <div className="space-y-2">
          <Label htmlFor="confirm-org-name">
            {t("org.deleteOrgConfirm", { name: orgName })}
          </Label>
          <Input
            id="confirm-org-name"
            value={confirmName}
            onChange={(e) => setConfirmName(e.target.value)}
            placeholder={t("org.deleteOrgTypeName")}
            autoFocus
            disabled={deleting}
          />
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={deleting}
            className="cursor-pointer"
          >
            {t("common.cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={!nameMatches || deleting}
            onClick={handleDelete}
            className="cursor-pointer"
          >
            {deleting ? t("org.deleting") : t("org.deleteOrg")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
