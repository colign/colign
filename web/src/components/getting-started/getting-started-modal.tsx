"use client";

import { useState } from "react";
import { Briefcase, Code2, ExternalLink, Sparkles, TerminalSquare } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";

const REPO_URL = "https://github.com/colign/plugin";

const INSTALL_COMMANDS = [
  "/plugin marketplace add https://github.com/colign/plugin",
  "/plugin install colign@colign",
  "/reload-plugins",
];

const PM_DESIGNER_EXAMPLES = [
  "/colign:onboard",
  "/colign:explore review the current project and summarize the active changes",
  "/colign:propose turn this requirement into a structured proposal with scope and out-of-scope",
];

const DEVELOPER_EXAMPLES = [
  "/colign:plan break this approved proposal into architecture and implementation tasks",
  "/colign:implement pick the next task from the current change and start coding",
  "/colign:implement continue the current change and update task progress as you go",
];

interface GettingStartedModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function GettingStartedModal({ open, onOpenChange }: GettingStartedModalProps) {
  const { t } = useI18n();
  const [, setClipboardTick] = useState(0);

  async function copyText(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setClipboardTick((tick) => tick + 1);
    } catch {
      // Ignore clipboard failures; users can still copy manually.
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[calc(100vh-2rem)] flex-col overflow-hidden p-0 sm:max-w-2xl">
        <div className="border-b border-border/60 bg-gradient-to-br from-primary/10 via-background to-emerald-500/8 px-6 py-6">
          <DialogHeader className="gap-3">
            <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-[0.18em] text-primary/80">
              <Sparkles className="h-3.5 w-3.5" />
              <span>{t("gettingStarted.eyebrow")}</span>
            </div>
            <DialogTitle className="text-2xl leading-tight">
              {t("gettingStarted.title")}
            </DialogTitle>
            <DialogDescription className="max-w-xl text-sm">
              {t("gettingStarted.description")}
            </DialogDescription>
          </DialogHeader>
        </div>

        <div className="grid flex-1 gap-4 overflow-y-auto px-6 py-6">
          <section className="rounded-xl border border-border/60 bg-card/70 p-4">
            <div className="mb-3 flex items-center gap-2">
              <TerminalSquare className="h-4 w-4 text-primary" />
              <h3 className="text-sm font-semibold">{t("gettingStarted.installTitle")}</h3>
            </div>
            <p className="mb-3 text-sm text-muted-foreground">{t("gettingStarted.installDescription")}</p>
            <div className="space-y-2">
              {INSTALL_COMMANDS.map((command) => (
                <div
                  key={command}
                  className="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-background px-3 py-2"
                >
                  <code className="min-w-0 flex-1 overflow-x-auto text-xs sm:text-sm">{command}</code>
                  <Button variant="outline" size="sm" onClick={() => copyText(command)}>
                    {t("common.copy")}
                  </Button>
                </div>
              ))}
            </div>
          </section>

          <section className="rounded-xl border border-border/60 bg-card/70 p-4">
            <div className="mb-3 flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-primary" />
              <h3 className="text-sm font-semibold">{t("gettingStarted.skillsTitle")}</h3>
            </div>
            <p className="mb-3 text-sm text-muted-foreground">{t("gettingStarted.skillsDescription")}</p>
            <div className="grid gap-4 md:grid-cols-2">
              <section className="rounded-lg border border-border/50 bg-background p-4">
                <div className="mb-2 flex items-center gap-2">
                  <Briefcase className="h-4 w-4 text-primary" />
                  <h4 className="text-sm font-semibold">{t("gettingStarted.pmTitle")}</h4>
                </div>
                <p className="mb-3 text-sm text-muted-foreground">{t("gettingStarted.pmDescription")}</p>
                <div className="space-y-2">
                  {PM_DESIGNER_EXAMPLES.map((example) => (
                    <div
                      key={example}
                      className="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-card px-3 py-2"
                    >
                      <code className="min-w-0 flex-1 overflow-x-auto text-xs sm:text-sm">{example}</code>
                      <Button variant="outline" size="sm" onClick={() => copyText(example)}>
                        {t("common.copy")}
                      </Button>
                    </div>
                  ))}
                </div>
              </section>

              <section className="rounded-lg border border-border/50 bg-background p-4">
                <div className="mb-2 flex items-center gap-2">
                  <Code2 className="h-4 w-4 text-primary" />
                  <h4 className="text-sm font-semibold">{t("gettingStarted.devTitle")}</h4>
                </div>
                <p className="mb-3 text-sm text-muted-foreground">{t("gettingStarted.devDescription")}</p>
                <div className="space-y-2">
                  {DEVELOPER_EXAMPLES.map((example) => (
                    <div
                      key={example}
                      className="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-card px-3 py-2"
                    >
                      <code className="min-w-0 flex-1 overflow-x-auto text-xs sm:text-sm">{example}</code>
                      <Button variant="outline" size="sm" onClick={() => copyText(example)}>
                        {t("common.copy")}
                      </Button>
                    </div>
                  ))}
                </div>
              </section>
            </div>
          </section>
        </div>

        <DialogFooter className="justify-between sm:justify-between">
          <a href={REPO_URL} target="_blank" rel="noreferrer">
            <Button variant="outline">
              {t("gettingStarted.openGuide")}
              <ExternalLink className="h-4 w-4" />
            </Button>
          </a>
          <Button onClick={() => onOpenChange(false)}>{t("gettingStarted.startInProjects")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
