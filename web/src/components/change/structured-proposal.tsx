"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n";
import { documentClient } from "@/lib/document";
import { useEvents } from "@/lib/events";
import { showError } from "@/lib/toast";
import { AcceptanceCriteria } from "@/components/change/acceptance-criteria";
import { ChevronDown, ChevronRight, ExternalLink, Figma, Link2, Plus, Trash2 } from "lucide-react";
import { AIProposalGenerator } from "@/components/ai/ai-proposal-generator";
import { CommentPanel } from "@/components/comment/comment-panel";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { MessageSquare, PanelRightClose, PanelRightOpen } from "lucide-react";
import { getTokenPayload } from "@/lib/auth";
import type { MentionMember } from "@/components/comment/mention-textarea";

interface ProposalSections {
  problem: string;
  scope: string;
  outOfScope: string;
  approach: string;
  designLinks?: string[];
}

const EMPTY_SECTIONS: ProposalSections = {
  problem: "",
  scope: "",
  outOfScope: "",
  approach: "",
  designLinks: [],
};

type TextSectionKey = "problem" | "scope" | "outOfScope" | "approach";

interface SectionConfig {
  key: TextSectionKey;
  i18nKey: string;
  placeholderKey: string;
  required: boolean;
  minRows: number;
}

const SECTIONS: SectionConfig[] = [
  {
    key: "problem",
    i18nKey: "proposal.problem",
    placeholderKey: "proposal.problemPlaceholder",
    required: true,
    minRows: 3,
  },
  {
    key: "scope",
    i18nKey: "proposal.scope",
    placeholderKey: "proposal.scopePlaceholder",
    required: true,
    minRows: 4,
  },
  {
    key: "outOfScope",
    i18nKey: "proposal.outOfScope",
    placeholderKey: "proposal.outOfScopePlaceholder",
    required: false,
    minRows: 2,
  },
  {
    key: "approach",
    i18nKey: "proposal.approach",
    placeholderKey: "proposal.approachPlaceholder",
    required: false,
    minRows: 3,
  },
];

function parseContent(content: string): ProposalSections {
  try {
    const parsed = JSON.parse(content);
    if (parsed && typeof parsed.problem === "string") {
      return { ...EMPTY_SECTIONS, ...parsed };
    }
  } catch {
    // Legacy HTML content — put it all in problem
    if (content.trim()) {
      // Strip HTML tags for plain text
      const text = content
        .replace(/<[^>]*>/g, "")
        .replace(/\s+/g, " ")
        .trim();
      if (
        text &&
        text !== "Explain the motivation for this change. What problem does this solve?"
      ) {
        return { ...EMPTY_SECTIONS, problem: text };
      }
    }
  }
  return { ...EMPTY_SECTIONS };
}

interface StructuredProposalProps {
  changeId: bigint;
  projectId: bigint;
  currentStage?: string;
  members?: MentionMember[];
}

export function StructuredProposal({ changeId, projectId, currentStage, members = [] }: StructuredProposalProps) {
  const { t } = useI18n();
  const { on } = useEvents();
  const payload = typeof window !== "undefined" ? getTokenPayload() : null;
  const [mobileCommentsOpen, setMobileCommentsOpen] = useState(false);
  const [commentsOpen, setCommentsOpen] = useState(true);
  const [sections, setSections] = useState<ProposalSections>(EMPTY_SECTIONS);
  const [loading, setLoading] = useState(true);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({
    outOfScope: true,
    approach: true,
  });
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const localRevisionRef = useRef(0);
  const lastSavedRevisionRef = useRef(0);
  const inFlightSaveRevisionRef = useRef<number | null>(null);
  const sectionsRef = useRef(sections);
  sectionsRef.current = sections;

  const load = useCallback(async () => {
    try {
      const res = await documentClient.getDocument({ changeId, type: "proposal", projectId });
      if (res.document?.content) {
        const parsed = parseContent(res.document.content);
        setSections(parsed);
        setCollapsed({
          outOfScope: !parsed.outOfScope,
          approach: !parsed.approach,
        });
      } else {
        setSections(EMPTY_SECTIONS);
      }
    } catch (err) {
      showError("Failed to save proposal", err);
    } finally {
      setLoading(false);
    }
  }, [changeId, projectId]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    return on((event) => {
      if (event.type !== "document_updated" || event.changeId !== changeId) return;
      // Skip remote reload while local edits are pending or currently being saved.
      if (
        saveTimerRef.current ||
        inFlightSaveRevisionRef.current !== null ||
        localRevisionRef.current !== lastSavedRevisionRef.current
      ) {
        return;
      }
      try {
        const eventPayload = event.payload ? JSON.parse(event.payload) : {};
        if (eventPayload.docType === "proposal") {
          load();
        }
      } catch {
        // Ignore malformed payloads.
      }
    });
  }, [on, changeId, load]);

  // Save to server
  const saveNow = useCallback(async () => {
    const revisionToSave = localRevisionRef.current;
    inFlightSaveRevisionRef.current = revisionToSave;
    try {
      await documentClient.saveDocument({
        changeId,
        type: "proposal",
        title: "Proposal",
        content: JSON.stringify(sectionsRef.current),
        projectId,
      });
      if (localRevisionRef.current === revisionToSave) {
        lastSavedRevisionRef.current = revisionToSave;
      }
    } catch (err) {
      showError("Failed to save proposal", err);
    } finally {
      if (inFlightSaveRevisionRef.current === revisionToSave) {
        inFlightSaveRevisionRef.current = null;
      }
    }
  }, [changeId, projectId]);

  // Debounced save for text input
  const save = useCallback(() => {
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveTimerRef.current = setTimeout(async () => {
      saveTimerRef.current = null;
      await saveNow();
    }, 1000);
  }, [saveNow]);

  function updateSection(key: TextSectionKey, value: string) {
    setSections((prev) => ({ ...prev, [key]: value }));
    localRevisionRef.current += 1;
    save();
  }

  function addDesignLink(url: string) {
    const trimmed = url.trim();
    if (!trimmed) return;
    localRevisionRef.current += 1;
    setSections((prev) => {
      const updated = { ...prev, designLinks: [...(prev.designLinks || []), trimmed] };
      sectionsRef.current = updated;
      return updated;
    });
    saveNow();
  }

  function removeDesignLink(index: number) {
    localRevisionRef.current += 1;
    setSections((prev) => {
      const updated = { ...prev, designLinks: (prev.designLinks || []).filter((_, i) => i !== index) };
      sectionsRef.current = updated;
      return updated;
    });
    saveNow();
  }

  function toggleCollapse(key: string) {
    setCollapsed((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const isReviewMode = currentStage === "review" || currentStage === "ready";

  const isProposalEmpty =
    !sections.problem.trim() &&
    !sections.scope.trim() &&
    !sections.outOfScope.trim() &&
    !sections.approach.trim();

  function handleAIApply(applied: { problem: string; scope: string; outOfScope: string; approach: string }) {
    setSections((prev) => ({ ...prev, ...applied }));
    localRevisionRef.current += 1;
    // Expand optional sections that now have content
    setCollapsed({
      outOfScope: !applied.outOfScope,
      approach: !applied.approach,
    });
    save();
  }

  return (
    <div className="flex gap-4 py-4">
      <div className="min-w-0 flex-1 space-y-4">
      {!isReviewMode && (
        <AIProposalGenerator
          changeId={changeId}
          onApply={handleAIApply}
          hasExistingContent={!isProposalEmpty}
        />
      )}
      {SECTIONS.map((section) => {
        const isCollapsed = collapsed[section.key] && !sections[section.key];
        const hasContent = !!sections[section.key].trim();

        return (
          <div
            key={section.key}
            className="rounded-xl border border-border/40 bg-card/50 transition-colors"
          >
            {/* Section header */}
            <button
              onClick={() => !section.required && toggleCollapse(section.key)}
              className={`flex w-full items-center justify-between px-5 py-3 ${
                section.required ? "" : "cursor-pointer"
              }`}
            >
              <div className="flex items-center gap-2.5">
                {!section.required &&
                  (isCollapsed ? (
                    <ChevronRight className="size-3.5 text-muted-foreground/50" />
                  ) : (
                    <ChevronDown className="size-3.5 text-muted-foreground/50" />
                  ))}
                <span className="text-sm font-medium">{t(section.i18nKey)}</span>
                {section.required ? (
                  <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                    {t("proposal.required")}
                  </span>
                ) : (
                  <span className="rounded bg-muted/80 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                    {t("proposal.optional")}
                  </span>
                )}
                {hasContent && <div className="h-1.5 w-1.5 rounded-full bg-emerald-400" />}
              </div>
            </button>

            {/* Section body */}
            {(!isCollapsed || section.required) && (
              <div className="border-t border-border/30 px-5 py-4">
                <textarea
                  value={sections[section.key]}
                  onChange={(e) => updateSection(section.key, e.target.value)}
                  placeholder={t(section.placeholderKey)}
                  readOnly={isReviewMode}
                  rows={Math.max(section.minRows, sections[section.key].split("\n").length + 1)}
                  className="w-full resize-none bg-transparent text-sm leading-relaxed text-foreground outline-none placeholder:text-muted-foreground/40"
                />
              </div>
            )}
          </div>
        );
      })}

      {/* Design Links */}
      <DesignLinksSection
        links={sections.designLinks || []}
        onAdd={addDesignLink}
        onRemove={removeDesignLink}
        readOnly={isReviewMode}
        t={t}
      />

      {/* Acceptance Criteria */}
      <AcceptanceCriteria
        changeId={changeId}
        projectId={projectId}
        reviewMode={isReviewMode}
        hasProposal={!!(sections.problem || sections.scope)}
      />
      </div>

      {/* Comment Panel — desktop */}
      <div className="hidden shrink-0 md:block">
        <div className="sticky top-56">
          {commentsOpen ? (
            <div className="w-80 rounded-xl border border-border/40 bg-card/50" style={{ maxHeight: "calc(100vh - 15rem)" }}>
              <div className="flex items-center justify-end border-b border-border/50 px-2 py-1">
                <button
                  onClick={() => setCommentsOpen(false)}
                  className="cursor-pointer rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
                  title={t("comments.hidePanel")}
                >
                  <PanelRightClose className="size-4" />
                </button>
              </div>
              <CommentPanel
                changeId={changeId}
                projectId={projectId}
                documentType="proposal"
                currentUserId={payload?.user_id}
                members={members}
                showCompose
              />
            </div>
          ) : (
            <button
              onClick={() => setCommentsOpen(true)}
              className="cursor-pointer rounded-xl border border-border/40 bg-card/50 p-2 text-muted-foreground hover:bg-accent hover:text-foreground"
              title={t("comments.showPanel")}
            >
              <PanelRightOpen className="size-4" />
            </button>
          )}
        </div>
      </div>

      {/* Comment Panel — mobile */}
      <div className="fixed bottom-6 right-6 z-40 md:hidden">
        <Button
          size="icon"
          onClick={() => setMobileCommentsOpen(true)}
          className="h-12 w-12 cursor-pointer rounded-full shadow-lg"
        >
          <MessageSquare className="size-5" />
        </Button>
      </div>
      {mobileCommentsOpen && (
        <Sheet open={mobileCommentsOpen} onOpenChange={setMobileCommentsOpen}>
          <SheetContent side="right">
            <SheetTitle>{t("comments.comments")}</SheetTitle>
            <div className="flex-1 overflow-y-auto">
              <CommentPanel
                changeId={changeId}
                projectId={projectId}
                documentType="proposal"
                currentUserId={payload?.user_id}
                members={members}
                showCompose
              />
            </div>
          </SheetContent>
        </Sheet>
      )}
    </div>
  );
}

// --- Design Links Section ---

function getFigmaEmbedUrl(url: string): string | null {
  if (/figma\.com\/(file|design|proto|board)\//.test(url)) {
    return `https://www.figma.com/embed?embed_host=colign&url=${encodeURIComponent(url)}`;
  }
  return null;
}

function getLinkType(url: string): "figma" | "generic" {
  return /figma\.com/.test(url) ? "figma" : "generic";
}

function extractLinkTitle(url: string): string {
  try {
    const parsed = new URL(url);
    // Figma URLs: /design/{fileKey}/{file-name-slug} or /file/{fileKey}/{file-name-slug}
    if (/figma\.com/.test(parsed.hostname)) {
      const segments = parsed.pathname.split("/").filter(Boolean);
      // segments: ["design", "fileKey", "file-name-slug"] or ["file", "fileKey", "file-name-slug"]
      const nameSlug = segments[2];
      if (nameSlug) {
        return decodeURIComponent(nameSlug)
          .replace(/-+/g, " ")
          .replace(/^\s+|\s+$/g, "")
          .trim();
      }
      return "Figma";
    }
    // Generic URLs: use hostname + pathname
    return parsed.hostname.replace("www.", "") + parsed.pathname;
  } catch {
    return url;
  }
}

interface DesignLinksSectionProps {
  links: string[];
  onAdd: (url: string) => void;
  onRemove: (index: number) => void;
  readOnly: boolean;
  t: (key: string) => string;
}

function DesignLinksSection({ links, onAdd, onRemove, readOnly, t }: DesignLinksSectionProps) {
  const [inputValue, setInputValue] = useState("");
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});

  function handleAdd() {
    if (!inputValue.trim()) return;
    onAdd(inputValue);
    setInputValue("");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAdd();
    }
  }

  function togglePreview(index: number) {
    setExpanded((prev) => ({ ...prev, [index]: !prev[index] }));
  }

  return (
    <div className="rounded-xl border border-border/40 bg-card/50">
      <div className="flex items-center gap-2.5 px-5 py-3">
        <Link2 className="size-3.5 text-muted-foreground/50" />
        <span className="text-sm font-medium">{t("proposal.designLinks")}</span>
        <span className="rounded bg-muted/80 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
          {t("proposal.optional")}
        </span>
        {links.length > 0 && <div className="h-1.5 w-1.5 rounded-full bg-emerald-400" />}
      </div>

      <div className="border-t border-border/30 px-5 py-4 space-y-3">
        {/* Link list */}
        {links.map((link, index) => {
          const type = getLinkType(link);
          const embedUrl = getFigmaEmbedUrl(link);
          const isExpanded = expanded[index] ?? (type === "figma");

          return (
            <div key={index} className="space-y-2">
              <div className="group flex items-center gap-2 rounded-lg border border-border/30 bg-background/50 px-3 py-2">
                {type === "figma" ? (
                  <Figma className="size-4 shrink-0 text-muted-foreground" />
                ) : (
                  <Link2 className="size-4 shrink-0 text-muted-foreground" />
                )}
                <a
                  href={link}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex-1 truncate text-sm text-primary hover:underline"
                  title={link}
                >
                  {extractLinkTitle(link)}
                </a>
                {embedUrl && (
                  <button
                    onClick={() => togglePreview(index)}
                    className="cursor-pointer text-xs text-muted-foreground hover:text-foreground"
                  >
                    {isExpanded ? t("proposal.hidePreview") : t("proposal.showPreview")}
                  </button>
                )}
                <a
                  href={link}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground"
                >
                  <ExternalLink className="size-3.5" />
                </a>
                {!readOnly && (
                  <button
                    onClick={() => onRemove(index)}
                    className="cursor-pointer text-muted-foreground/50 opacity-0 transition-opacity hover:text-destructive group-hover:opacity-100"
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                )}
              </div>

              {/* Figma embed preview */}
              {embedUrl && isExpanded && (
                <div className="overflow-hidden rounded-lg border border-border/30">
                  <iframe
                    src={embedUrl}
                    className="h-[450px] w-full"
                    allowFullScreen
                  />
                </div>
              )}
            </div>
          );
        })}

        {/* Add link input */}
        {!readOnly && (
          <div className="flex items-center gap-2">
            <input
              type="url"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t("proposal.designLinkPlaceholder")}
              className="flex-1 rounded-lg border border-border/30 bg-background/50 px-3 py-2 text-sm outline-none placeholder:text-muted-foreground/40 focus:border-primary/50"
            />
            <button
              onClick={handleAdd}
              disabled={!inputValue.trim()}
              className="cursor-pointer rounded-lg border border-border/30 px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40"
            >
              <Plus className="size-4" />
            </button>
          </div>
        )}

        {links.length === 0 && readOnly && (
          <p className="text-sm text-muted-foreground/40">{t("proposal.noDesignLinks")}</p>
        )}
      </div>
    </div>
  );
}
