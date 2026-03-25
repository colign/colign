"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
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
import { projectClient } from "@/lib/project";
import { showError } from "@/lib/toast";
import { useI18n } from "@/lib/i18n";

interface CreateProjectDialogProps {
  onCreated?: () => void;
  children?: React.ReactNode;
}

export function CreateProjectDialog({ onCreated, children }: CreateProjectDialogProps) {
  const router = useRouter();
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) return;

    setSaving(true);
    setError("");
    try {
      const res = await projectClient.createProject({ name: trimmedName, description: description.trim() });
      setOpen(false);
      setName("");
      setDescription("");
      onCreated?.();
      router.push(`/projects/${res.project?.slug}`);
    } catch (err: unknown) {
      showError(t("toast.createFailed"), err);
      setError(err instanceof Error ? err.message : t("toast.createFailed"));
      setSaving(false);
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) {
      setName("");
      setDescription("");
      setError("");
      setSaving(false);
    }
    setOpen(nextOpen);
  }

  return (
    <>
      <span onClick={() => setOpen(true)}>
        {children ?? (
          <Button size="sm" className="cursor-pointer">
            <Plus className="mr-1.5 h-4 w-4" />
            {t("projects.newProject")}
          </Button>
        )}
      </span>

      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("projects.createProject")}</DialogTitle>
            <DialogDescription>{t("projects.setupSDD")}</DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="project-name">{t("projects.projectName")}</Label>
              <Input
                id="project-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My App"
                autoFocus
                disabled={saving}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="project-description">{t("projects.description")}</Label>
              <Input
                id="project-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="A brief description of your project"
                disabled={saving}
              />
            </div>

            {error && <p className="text-sm text-destructive">{error}</p>}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)} disabled={saving} className="cursor-pointer">
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={saving || !name.trim()} className="cursor-pointer">
                {saving ? t("common.creating") : t("projects.createProject")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
