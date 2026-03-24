"use client";

import { useCallback, useRef, useState, useEffect } from "react";
import { SpecEditor } from "@/components/editor/spec-editor";
import { MarginComments } from "@/components/comment/margin-comments";
import { sddTemplates } from "@/components/editor/templates";
import { commentClient } from "@/lib/comment";
import { documentClient } from "@/lib/document";
import { showError } from "@/lib/toast";
import { getTokenPayload } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { AcceptanceCriteria } from "@/components/change/acceptance-criteria";

interface DocumentTabProps {
  changeId: bigint;
  docType: "proposal" | "design" | "spec";
  currentStage?: string;
}

export function DocumentTab({ changeId, docType, currentStage }: DocumentTabProps) {
  const { t } = useI18n();
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(true);
  const [pendingQuotedText, setPendingQuotedText] = useState<string | null>(null);
  const [commentInput, setCommentInput] = useState("");
  const [commentPosition, setCommentPosition] = useState<{ top: number } | null>(null);
  const [editorDom, setEditorDom] = useState<HTMLElement | null>(null);
  const editorWrapperRef = useRef<HTMLDivElement>(null);
  const payload = typeof window !== "undefined" ? getTokenPayload() : null;

  const editorRef = useRef<{
    addHighlightAtSavedSelection: (commentId: string) => void;
    removeHighlight: (commentId: string) => void;
    scrollToHighlight: (commentId: string) => void;
    getEditorDom: () => HTMLElement | null;
  } | null>(null);

  const commentRefreshRef = useRef<(() => void) | null>(null);

  // Load document from server
  useEffect(() => {
    async function loadDocument() {
      try {
        const res = await documentClient.getDocument({ changeId, type: docType });
        if (res.document) {
          setContent(res.document.content);
        } else {
          setContent(sddTemplates[docType] || "");
        }
      } catch (err) {
        showError("Failed to load document", err);
        setContent(sddTemplates[docType] || "");
      } finally {
        setLoading(false);
      }
    }
    loadDocument();
  }, [changeId, docType]);

  // Get editor DOM once editor is ready
  useEffect(() => {
    const interval = setInterval(() => {
      if (editorRef.current) {
        const dom = editorRef.current.getEditorDom();
        if (dom) {
          setEditorDom(dom);
          clearInterval(interval);
        }
      }
    }, 100);
    return () => clearInterval(interval);
  }, [loading]);

  const handleAddComment = (
    quotedText: string,
    rect: { top: number; left: number; width: number },
  ) => {
    setPendingQuotedText(quotedText);
    setCommentInput("");
    setCommentPosition({ top: rect.top });
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
      if (res.comment && editorRef.current) {
        editorRef.current.addHighlightAtSavedSelection(String(res.comment.id));
      }
      setPendingQuotedText(null);
      setCommentInput("");
      setCommentPosition(null);
      commentRefreshRef.current?.();
    } catch (err) {
      showError("Failed to save comment", err);
    }
  };

  const handleHighlightClick = (commentId: string) => {
    // handled by margin comments hover
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="py-4">
      {/* Editor + Margin Comments */}
      <div className="flex gap-4">
        <div className="relative min-w-0 flex-1" ref={editorWrapperRef}>
          {/* Inline comment form — positioned below selected text */}
          {pendingQuotedText && commentPosition && (
            <div
              className="absolute left-0 right-0 z-30 mx-6"
              style={{ top: commentPosition.top + 8 }}
            >
              <div className="rounded-lg border border-primary/30 bg-card p-3 shadow-xl">
                <div className="rounded border-l-2 border-yellow-500/50 bg-yellow-500/5 px-2 py-1 text-xs text-muted-foreground">
                  &ldquo;{pendingQuotedText}&rdquo;
                </div>
                <textarea
                  autoFocus
                  value={commentInput}
                  onChange={(e) => setCommentInput(e.target.value)}
                  placeholder={t("comments.placeholder")}
                  className="mt-2 w-full resize-none rounded-md border border-border/50 bg-transparent px-2 py-1.5 text-sm outline-none focus:border-primary"
                  rows={2}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                      e.preventDefault();
                      handleSubmitComment();
                    }
                    if (e.key === "Escape") {
                      setPendingQuotedText(null);
                      setCommentPosition(null);
                    }
                  }}
                />
                <div className="mt-1.5 flex justify-end gap-1.5">
                  <button
                    onClick={() => {
                      setPendingQuotedText(null);
                      setCommentPosition(null);
                    }}
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
            </div>
          )}

          <SpecEditor
            initialContent={content}
            placeholder={`Start writing your ${docType}...`}
            onAddComment={handleAddComment}
            onHighlightClick={handleHighlightClick}
            editorRef={editorRef}
            documentId={`change-${changeId}-${docType}`}
            userName={payload?.name || payload?.email?.split("@")[0] || "Anonymous"}
          />
        </div>

        {/* Margin comments */}
        <MarginComments
          changeId={changeId}
          documentType={docType}
          currentUserId={payload?.user_id}
          editorDom={editorDom}
          refreshRef={commentRefreshRef}
          onRemoveHighlight={(commentId) => {
            editorRef.current?.removeHighlight(commentId);
          }}
        />
      </div>

      {/* Acceptance Criteria — show on proposal tab */}
      {docType === "proposal" && (
        <AcceptanceCriteria
          changeId={changeId}
          reviewMode={currentStage === "review" || currentStage === "ready"}
        />
      )}
    </div>
  );
}
