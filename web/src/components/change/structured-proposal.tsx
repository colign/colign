"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { marked } from "marked";
import DOMPurify from "isomorphic-dompurify";
import { useI18n } from "@/lib/i18n";
import { documentClient } from "@/lib/document";
import { useEvents } from "@/lib/events";
import { showError } from "@/lib/toast";
import { AcceptanceCriteria } from "@/components/change/acceptance-criteria";
import {
  Bold,
  ChevronDown,
  ChevronRight,
  Code,
  ExternalLink,
  FileText,
  Figma,
  Heading2,
  Heading3,
  Italic,
  Link2,
  List,
  NotebookText,
  Plus,
  Trash2,
} from "lucide-react";
import { AIProposalGenerator } from "@/components/ai/ai-proposal-generator";
import { AI_APPLY_PROPOSAL_EVENT } from "@/components/ai/chat-proposal-result";
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
  designLinks?: string[];
  documentLinks?: string[];
}

const EMPTY_SECTIONS: ProposalSections = {
  problem: "",
  scope: "",
  outOfScope: "",
  designLinks: [],
  documentLinks: [],
};

type TextSectionKey = "problem" | "scope" | "outOfScope";

interface SectionConfig {
  key: TextSectionKey;
  i18nKey: string;
  placeholderKey: string;
  required: boolean;
}

const SECTIONS: SectionConfig[] = [
  {
    key: "problem",
    i18nKey: "proposal.problem",
    placeholderKey: "proposal.problemPlaceholder",
    required: true,
  },
  {
    key: "scope",
    i18nKey: "proposal.scope",
    placeholderKey: "proposal.scopePlaceholder",
    required: true,
  },
  {
    key: "outOfScope",
    i18nKey: "proposal.outOfScope",
    placeholderKey: "proposal.outOfScopePlaceholder",
    required: false,
  },
];

function normalizeSectionContent(content: string): string {
  const trimmed = content.trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("<")) return DOMPurify.sanitize(content);
  return DOMPurify.sanitize(marked.parse(content, { async: false }) as string);
}

function getPlainText(content: string): string {
  const normalized = content
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/(p|div|li|h1|h2|h3|h4|h5|h6|blockquote)>/gi, "\n")
    .replace(/<[^>]*>/g, " ")
    .replace(/\u00a0/g, " ");
  return normalized.replace(/\s+\n/g, "\n").replace(/\n\s+/g, "\n").replace(/\s+/g, " ").trim();
}

function isSafeUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return ["http:", "https:"].includes(parsed.protocol);
  } catch {
    return false;
  }
}

function parseContent(content: string): ProposalSections {
  try {
    const parsed = JSON.parse(content);
    if (parsed && typeof parsed.problem === "string") {
      return {
        ...EMPTY_SECTIONS,
        ...parsed,
        problem: normalizeSectionContent(parsed.problem ?? ""),
        scope: normalizeSectionContent(parsed.scope ?? ""),
        outOfScope: normalizeSectionContent(parsed.outOfScope ?? ""),
      };
    }
  } catch {
    // Legacy HTML content — put it all in problem
    if (content.trim()) {
      const text = getPlainText(content);
      if (
        text &&
        text !== "Explain the motivation for this change. What problem does this solve?"
      ) {
        return { ...EMPTY_SECTIONS, problem: normalizeSectionContent(text) };
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

export function StructuredProposal({
  changeId,
  projectId,
  currentStage,
  members = [],
}: StructuredProposalProps) {
  const { t } = useI18n();
  const { on } = useEvents();
  const payload = typeof window !== "undefined" ? getTokenPayload() : null;
  const [mobileCommentsOpen, setMobileCommentsOpen] = useState(false);
  const [commentsOpen, setCommentsOpen] = useState(true);
  const [sections, setSections] = useState<ProposalSections>(EMPTY_SECTIONS);
  const [loading, setLoading] = useState(true);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({
    outOfScope: true,
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
          outOfScope: !getPlainText(parsed.outOfScope),
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
    if (!trimmed || !isSafeUrl(trimmed)) return;
    if ((sectionsRef.current.designLinks || []).includes(trimmed)) return;
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
      const updated = {
        ...prev,
        designLinks: (prev.designLinks || []).filter((_, i) => i !== index),
      };
      sectionsRef.current = updated;
      return updated;
    });
    saveNow();
  }

  function addDocumentLink(url: string) {
    const trimmed = url.trim();
    if (!trimmed || !isSafeUrl(trimmed)) return;
    if ((sectionsRef.current.documentLinks || []).includes(trimmed)) return;
    localRevisionRef.current += 1;
    setSections((prev) => {
      const updated = { ...prev, documentLinks: [...(prev.documentLinks || []), trimmed] };
      sectionsRef.current = updated;
      return updated;
    });
    saveNow();
  }

  function removeDocumentLink(index: number) {
    localRevisionRef.current += 1;
    setSections((prev) => {
      const updated = {
        ...prev,
        documentLinks: (prev.documentLinks || []).filter((_, i) => i !== index),
      };
      sectionsRef.current = updated;
      return updated;
    });
    saveNow();
  }

  function toggleCollapse(key: string) {
    setCollapsed((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  const handleAIApply = useCallback((applied: { problem: string; scope: string; outOfScope: string }) => {
    setSections((prev) => ({
      ...prev,
      problem: normalizeSectionContent(applied.problem),
      scope: normalizeSectionContent(applied.scope),
      outOfScope: normalizeSectionContent(applied.outOfScope),
    }));
    localRevisionRef.current += 1;
    setCollapsed({
      outOfScope: !applied.outOfScope.trim(),
    });
    save();
  }, [save]);

  // Listen for AI panel proposal apply events
  useEffect(() => {
    function handlePanelApply(e: Event) {
      const detail = (e as CustomEvent).detail;
      if (detail && typeof detail === "object" && "problem" in detail) {
        handleAIApply(detail);
      }
    }
    window.addEventListener(AI_APPLY_PROPOSAL_EVENT, handlePanelApply);
    return () => window.removeEventListener(AI_APPLY_PROPOSAL_EVENT, handlePanelApply);
  }, [handleAIApply]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const isReviewMode = currentStage === "approved";

  const isProposalEmpty =
    !getPlainText(sections.problem) &&
    !getPlainText(sections.scope) &&
    !getPlainText(sections.outOfScope);

  return (
    <div className="flex items-stretch gap-4 py-4">
      <div className="min-w-0 flex-1 space-y-4">
        {!isReviewMode && (
          <AIProposalGenerator
            changeId={changeId}
            onApply={handleAIApply}
            hasExistingContent={!isProposalEmpty}
          />
        )}
        {SECTIONS.map((section) => {
          const plainText = getPlainText(sections[section.key]);
          const isCollapsed = !section.required && !!collapsed[section.key];
          const hasContent = !!plainText;

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
                  <ProposalSectionEditor
                    content={sections[section.key]}
                    onChange={(value) => updateSection(section.key, value)}
                    placeholder={t(section.placeholderKey)}
                    readOnly={isReviewMode}
                    plainTextLength={plainText.length}
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

        {/* Document Links */}
        <DocumentLinksSection
          links={sections.documentLinks || []}
          onAdd={addDocumentLink}
          onRemove={removeDocumentLink}
          readOnly={isReviewMode}
          t={t}
        />

        {/* Acceptance Criteria */}
        <AcceptanceCriteria
          changeId={changeId}
          projectId={projectId}
          reviewMode={isReviewMode}
          hasProposal={!!(getPlainText(sections.problem) || getPlainText(sections.scope))}
        />
      </div>

      {/* Comment Panel — desktop */}
      <div className="hidden shrink-0 self-stretch md:block">
        <div className="sticky top-4">
          {commentsOpen ? (
            <div
              className="flex w-80 flex-col overflow-hidden rounded-xl border border-border/40 bg-card/50"
              style={{ maxHeight: "calc(100svh - 2rem)" }}
            >
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

interface ProposalSectionEditorProps {
  content: string;
  onChange: (value: string) => void;
  placeholder: string;
  readOnly: boolean;
  plainTextLength: number;
}

function ProposalSectionEditor({
  content,
  onChange,
  placeholder,
  readOnly,
  plainTextLength,
}: ProposalSectionEditorProps) {
  const editor = useEditor({
    extensions: [StarterKit, Placeholder.configure({ placeholder })],
    content: content || undefined,
    editable: !readOnly,
    immediatelyRender: false,
    onUpdate: ({ editor: currentEditor }) => {
      onChange(currentEditor.getHTML());
    },
  });

  useEffect(() => {
    if (!editor) return;
    const nextContent = content || "";
    if (editor.getHTML() === nextContent) return;
    if (editor.isFocused) return;
    editor.commands.setContent(nextContent || "<p></p>");
  }, [editor, content]);

  useEffect(() => {
    if (!editor) return;
    editor.setEditable(!readOnly);
  }, [editor, readOnly]);

  const toolbarBtn = (
    active: boolean,
    onClick: () => void,
    children: React.ReactNode,
    label: string,
  ) => (
    <button
      onMouseDown={(e) => {
        e.preventDefault();
        onClick();
      }}
      className={`flex h-8 min-w-8 cursor-pointer items-center justify-center rounded-md border px-2 transition-colors ${
        active
          ? "border-primary/40 bg-primary/10 text-foreground"
          : "border-transparent text-muted-foreground hover:border-border/50 hover:bg-accent"
      }`}
      title={label}
      aria-label={label}
    >
      {children}
    </button>
  );

  const shouldScroll = plainTextLength > 3000;

  return (
    <div className="space-y-2">
      {editor && !readOnly && (
        <BubbleMenu
          editor={editor}
          className="flex items-center gap-1 rounded-lg border border-border/40 bg-popover/95 p-1.5 shadow-xl backdrop-blur"
        >
          {toolbarBtn(
            editor.isActive("bold"),
            () => editor.chain().focus().toggleBold().run(),
            <Bold className="size-4" />,
            "Bold",
          )}
          {toolbarBtn(
            editor.isActive("italic"),
            () => editor.chain().focus().toggleItalic().run(),
            <Italic className="size-4" />,
            "Italic",
          )}
          {toolbarBtn(
            editor.isActive("heading", { level: 2 }),
            () => editor.chain().focus().toggleHeading({ level: 2 }).run(),
            <Heading2 className="size-4" />,
            "Heading 2",
          )}
          {toolbarBtn(
            editor.isActive("heading", { level: 3 }),
            () => editor.chain().focus().toggleHeading({ level: 3 }).run(),
            <Heading3 className="size-4" />,
            "Heading 3",
          )}
          {toolbarBtn(
            editor.isActive("bulletList"),
            () => editor.chain().focus().toggleBulletList().run(),
            <List className="size-4" />,
            "Bullet list",
          )}
          {toolbarBtn(
            editor.isActive("codeBlock"),
            () => editor.chain().focus().toggleCodeBlock().run(),
            <Code className="size-4" />,
            "Code block",
          )}
        </BubbleMenu>
      )}

      <div
        className={`rounded-xl border border-border/20 bg-background/20 transition-colors ${
          readOnly ? "" : "focus-within:border-primary/40 focus-within:bg-background/30"
        }`}
      >
        <EditorContent
          editor={editor}
          className={`prose max-w-none px-4 py-3 text-sm dark:prose-invert [&_.ProseMirror]:min-h-[7rem] [&_.ProseMirror]:leading-relaxed [&_.ProseMirror]:outline-none [&_.ProseMirror_h2]:mt-5 [&_.ProseMirror_h2]:mb-2 [&_.ProseMirror_h2]:text-lg [&_.ProseMirror_h2]:font-semibold [&_.ProseMirror_h3]:mt-4 [&_.ProseMirror_h3]:mb-2 [&_.ProseMirror_h3]:text-base [&_.ProseMirror_h3]:font-semibold [&_.ProseMirror_ul]:my-2 [&_.ProseMirror_ul]:pl-6 [&_.ProseMirror_li]:my-1 [&_.ProseMirror_code]:rounded [&_.ProseMirror_code]:bg-muted [&_.ProseMirror_code]:px-1 [&_.ProseMirror_code]:py-0.5 [&_.ProseMirror_pre]:overflow-x-auto [&_.ProseMirror_pre]:rounded-lg [&_.ProseMirror_pre]:bg-muted/70 [&_.ProseMirror_pre]:p-3 [&_.ProseMirror_p.is-editor-empty:first-child::before]:pointer-events-none [&_.ProseMirror_p.is-editor-empty:first-child::before]:float-left [&_.ProseMirror_p.is-editor-empty:first-child::before]:h-0 [&_.ProseMirror_p.is-editor-empty:first-child::before]:text-muted-foreground/40 [&_.ProseMirror_p.is-editor-empty:first-child::before]:content-[attr(data-placeholder)] ${
            shouldScroll
              ? "[&_.ProseMirror]:max-h-[32rem] [&_.ProseMirror]:overflow-y-auto [&_.ProseMirror]:pr-2"
              : ""
          }`}
        />
      </div>
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
          const isExpanded = expanded[index] ?? type === "figma";

          const safeHref = isSafeUrl(link) ? link : "#";

          return (
            <div key={index} className="space-y-2">
              <div className="group flex items-center gap-2 rounded-lg border border-border/30 bg-background/50 px-3 py-2">
                {type === "figma" ? (
                  <Figma className="size-4 shrink-0 text-muted-foreground" />
                ) : (
                  <Link2 className="size-4 shrink-0 text-muted-foreground" />
                )}
                <a
                  href={safeHref}
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
                  href={safeHref}
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
                  <iframe src={embedUrl} className="h-[450px] w-full" allowFullScreen />
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

// --- Document Links Section ---

type DocumentLinkType = "google" | "notion" | "confluence" | "generic";

function getDocumentLinkType(url: string): DocumentLinkType {
  if (/docs\.google\.com|drive\.google\.com|sheets\.google\.com|slides\.google\.com/.test(url))
    return "google";
  if (/notion\.so|notion\.site/.test(url)) return "notion";
  if (/atlassian\.net\/wiki|\.confluence\./.test(url)) return "confluence";
  return "generic";
}

function getDocumentLinkIcon(type: DocumentLinkType) {
  switch (type) {
    case "notion":
      return <NotebookText className="size-4 shrink-0 text-muted-foreground" />;
    case "google":
    case "confluence":
    default:
      return <FileText className="size-4 shrink-0 text-muted-foreground" />;
  }
}

function getDocumentLinkLabel(type: DocumentLinkType): string {
  switch (type) {
    case "google":
      return "Google Docs";
    case "notion":
      return "Notion";
    case "confluence":
      return "Confluence";
    default:
      return "";
  }
}

interface DocumentLinksSectionProps {
  links: string[];
  onAdd: (url: string) => void;
  onRemove: (index: number) => void;
  readOnly: boolean;
  t: (key: string) => string;
}

function DocumentLinksSection({ links, onAdd, onRemove, readOnly, t }: DocumentLinksSectionProps) {
  const [inputValue, setInputValue] = useState("");

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

  return (
    <div className="rounded-xl border border-border/40 bg-card/50">
      <div className="flex items-center gap-2.5 px-5 py-3">
        <FileText className="size-3.5 text-muted-foreground/50" />
        <span className="text-sm font-medium">{t("proposal.documentLinks")}</span>
        <span className="rounded bg-muted/80 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
          {t("proposal.optional")}
        </span>
        {links.length > 0 && <div className="h-1.5 w-1.5 rounded-full bg-emerald-400" />}
      </div>

      <div className="border-t border-border/30 px-5 py-4 space-y-3">
        {links.map((link, index) => {
          const type = getDocumentLinkType(link);
          const label = getDocumentLinkLabel(type);
          const safeHref = isSafeUrl(link) ? link : "#";

          return (
            <div
              key={link}
              className="group flex items-center gap-2 rounded-lg border border-border/30 bg-background/50 px-3 py-2"
            >
              {getDocumentLinkIcon(type)}
              <a
                href={safeHref}
                target="_blank"
                rel="noopener noreferrer"
                className="flex-1 truncate text-sm text-primary hover:underline"
                title={link}
              >
                {extractLinkTitle(link)}
              </a>
              {label && (
                <span className="shrink-0 rounded bg-muted/80 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                  {label}
                </span>
              )}
              <a
                href={safeHref}
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
          );
        })}

        {!readOnly && (
          <div className="flex items-center gap-2">
            <input
              type="url"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t("proposal.documentLinkPlaceholder")}
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
          <p className="text-sm text-muted-foreground/40">{t("proposal.noDocumentLinks")}</p>
        )}
      </div>
    </div>
  );
}
