"use client";

import { useCallback, useRef, useState } from "react";
import { SpecEditor } from "@/components/editor/spec-editor";
import { CommentPanel } from "@/components/comment/comment-panel";
import { sddTemplates } from "@/components/editor/templates";
import { commentClient } from "@/lib/comment";
import { getTokenPayload } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { MessageSquare, X } from "lucide-react";

interface DocumentTabProps {
  changeId: bigint;
  docType: "proposal" | "design" | "spec" | "tasks";
  initialContent?: string;
}

export function DocumentTab({ changeId, docType, initialContent }: DocumentTabProps) {
  const { t } = useI18n();
  const [content] = useState(initialContent || sddTemplates[docType] || "");
  const [commentPanelOpen, setCommentPanelOpen] = useState(false);
  const [pendingQuotedText, setPendingQuotedText] = useState<string | null>(null);
  const [commentInput, setCommentInput] = useState("");
  const payload = typeof window !== "undefined" ? getTokenPayload() : null;

  const editorRef = useRef<{
    addHighlightAtSavedSelection: (commentId: string) => void;
    removeHighlight: (commentId: string) => void;
    scrollToHighlight: (commentId: string) => void;
  } | null>(null);

  const commentPanelRef = useRef<HTMLDivElement>(null);
  const commentRefreshRef = useRef<(() => void) | null>(null);

  const handleSave = useCallback(
    async (newContent: string) => {
      try {
        await fetch(`${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/documents`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            change_id: Number(changeId),
            type: docType,
            content: newContent,
          }),
        });
      } catch {
        // retry handled by editor
      }
    },
    [changeId, docType],
  );

  const handleAddComment = (quotedText: string) => {
    setPendingQuotedText(quotedText);
    setCommentPanelOpen(true);
    setCommentInput("");
  };

  const handleSubmitComment = async () => {
    if (!commentInput.trim() || !pendingQuotedText) return;
    try {
      const res = await commentClient.createComment({
        changeId,
        documentType: docType,
        quotedText: pendingQuotedText,
        body: commentInput,
      });
      // Add highlight mark in editor at saved selection
      if (res.comment && editorRef.current) {
        editorRef.current.addHighlightAtSavedSelection(String(res.comment.id));
      }
      setPendingQuotedText(null);
      setCommentInput("");
      // Refresh comment list immediately
      commentRefreshRef.current?.();
    } catch {
      // handle error
    }
  };

  const handleHighlightClick = (commentId: string) => {
    setCommentPanelOpen(true);
    // Scroll to comment in panel
    setTimeout(() => {
      const el = commentPanelRef.current?.querySelector(`[data-comment-id="${commentId}"]`);
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 100);
  };

  const handleCommentClick = (commentId: string) => {
    editorRef.current?.scrollToHighlight(commentId);
  };

  return (
    <div className="flex gap-4 py-4">
      {/* Editor */}
      <div className={`min-w-0 ${commentPanelOpen ? "flex-1" : "w-full"}`}>
        <SpecEditor
          initialContent={content}
          placeholder={`Start writing your ${docType}...`}
          onSave={handleSave}
          onAddComment={handleAddComment}
          onHighlightClick={handleHighlightClick}
          editorRef={editorRef}
        />
      </div>

      {/* Comment Panel Toggle (when closed) */}
      {!commentPanelOpen && (
        <button
          onClick={() => setCommentPanelOpen(true)}
          className="fixed right-6 top-1/2 z-20 flex cursor-pointer items-center gap-1 rounded-l-lg border border-border bg-popover px-2 py-2 text-xs shadow-md transition-colors hover:bg-accent"
        >
          <MessageSquare className="size-4" />
        </button>
      )}

      {/* Comment Panel (when open) */}
      {commentPanelOpen && (
        <div
          ref={commentPanelRef}
          className="w-80 shrink-0 rounded-lg border border-border/50"
        >
          <div className="flex items-center justify-between border-b border-border/50 px-3 py-2">
            <span className="text-sm font-medium">{t("comments.comments")}</span>
            <button
              onClick={() => setCommentPanelOpen(false)}
              className="cursor-pointer rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <X className="size-3.5" />
            </button>
          </div>

          {/* New comment input (when adding) */}
          {pendingQuotedText && (
            <div className="border-b border-border/50 p-3">
              <div className="rounded border-l-2 border-yellow-500/50 bg-yellow-500/5 px-2 py-1 text-xs text-muted-foreground">
                &ldquo;{pendingQuotedText}&rdquo;
              </div>
              <textarea
                autoFocus
                value={commentInput}
                onChange={(e) => setCommentInput(e.target.value)}
                placeholder={t("comments.placeholder")}
                className="mt-2 w-full resize-none rounded-md border border-border/50 bg-transparent px-2 py-1.5 text-sm outline-none focus:border-primary"
                rows={3}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                    e.preventDefault();
                    handleSubmitComment();
                  }
                }}
              />
              <div className="mt-1.5 flex justify-end gap-1.5">
                <button
                  onClick={() => setPendingQuotedText(null)}
                  className="cursor-pointer rounded px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  {t("common.cancel")}
                </button>
                <button
                  onClick={handleSubmitComment}
                  disabled={!commentInput.trim()}
                  className="cursor-pointer rounded bg-primary px-2 py-1 text-xs text-primary-foreground disabled:opacity-50"
                >
                  {t("comments.addComment")}
                </button>
              </div>
            </div>
          )}

          {/* Comment list */}
          <CommentPanel
            changeId={changeId}
            documentType={docType}
            currentUserId={payload?.user_id}
            onCommentClick={handleCommentClick}
            refreshRef={commentRefreshRef}
          />
        </div>
      )}
    </div>
  );
}
