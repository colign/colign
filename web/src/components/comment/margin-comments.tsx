"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useI18n } from "@/lib/i18n";
import { commentClient } from "@/lib/comment";
import { Check, Trash2, Send, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { showError } from "@/lib/toast";
import { MentionTextarea, renderMentionBody, type MentionMember } from "./mention-textarea";

interface CommentData {
  id: bigint;
  quotedText: string;
  body: string;
  userId: bigint;
  userName: string;
  resolved: boolean;
  replies: ReplyData[];
  createdAt: Date;
}

interface ReplyData {
  id: bigint;
  body: string;
  userId: bigint;
  userName: string;
  createdAt: Date;
}

interface MarginCommentsProps {
  changeId: bigint;
  projectId: bigint;
  documentType: string;
  currentUserId?: number;
  members?: MentionMember[];
  editorDom: HTMLElement | null;
  refreshRef?: React.MutableRefObject<(() => void) | null>;
  onRemoveHighlight?: (commentId: string) => void;
}

function timeAgo(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  return `${days}d`;
}

export function MarginComments({
  changeId,
  projectId,
  documentType,
  currentUserId,
  members = [],
  editorDom,
  refreshRef,
  onRemoveHighlight,
}: MarginCommentsProps) {
  const { t } = useI18n();
  const [comments, setComments] = useState<CommentData[]>([]);
  const [positions, setPositions] = useState<Map<string, number>>(new Map());
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [hoveredCommentId, setHoveredCommentId] = useState<string | null>(null);
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [replyText, setReplyText] = useState("");
  const [replyMentionIds, setReplyMentionIds] = useState<bigint[]>([]);
  const [showResolved, setShowResolved] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const loadComments = useCallback(async () => {
    try {
      const res = await commentClient.listComments({ changeId, documentType, projectId });
      setComments(
        res.comments.map((c) => ({
          id: c.id,
          quotedText: c.quotedText,
          body: c.body,
          userId: c.userId,
          userName: c.userName,
          resolved: c.resolved,
          replies: c.replies.map((r) => ({
            id: r.id,
            body: r.body,
            userId: r.userId,
            userName: r.userName,
            createdAt: r.createdAt ? new Date(Number(r.createdAt.seconds) * 1000) : new Date(),
          })),
          createdAt: c.createdAt ? new Date(Number(c.createdAt.seconds) * 1000) : new Date(),
        })),
      );
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    }
  }, [changeId, documentType, projectId, t]);

  useEffect(() => {
    loadComments();
    const interval = setInterval(loadComments, 30000);
    return () => clearInterval(interval);
  }, [loadComments]);

  useEffect(() => {
    if (refreshRef) refreshRef.current = loadComments;
  }, [refreshRef, loadComments]);

  // Calculate comment positions based on editor DOM
  const calculatePositions = useCallback(() => {
    if (!editorDom) return;
    const newPositions = new Map<string, number>();
    let lastBottom = 0;

    const visibleComments = showResolved ? comments : comments.filter((c) => !c.resolved);

    for (const comment of visibleComments) {
      const el = editorDom.querySelector(`[data-comment-id="${comment.id}"]`);
      if (el) {
        const rect = el.getBoundingClientRect();
        const editorRect = editorDom.getBoundingClientRect();
        let top = rect.top - editorRect.top;

        // Avoid overlap with previous comment card
        if (top < lastBottom + 8) {
          top = lastBottom + 8;
        }

        newPositions.set(String(comment.id), top);
        // Estimate card height (collapsed ~48px, expanded ~200px)
        const isExpanded = expandedId === String(comment.id);
        lastBottom = top + (isExpanded ? 200 : 48);
      }
    }

    setPositions(newPositions);
  }, [editorDom, comments, expandedId, showResolved]);

  useEffect(() => {
    calculatePositions();
  }, [calculatePositions]);

  // Recalculate on resize
  useEffect(() => {
    if (!editorDom) return;
    const observer = new ResizeObserver(() => {
      requestAnimationFrame(calculatePositions);
    });
    observer.observe(editorDom);
    return () => observer.disconnect();
  }, [editorDom, calculatePositions]);

  // Highlight hover sync: editor → margin
  useEffect(() => {
    if (!editorDom) return;
    const handleMouseOver = (e: MouseEvent) => {
      const target = (e.target as HTMLElement).closest("[data-comment-id]");
      if (target) {
        setHoveredCommentId(target.getAttribute("data-comment-id"));
      }
    };
    const handleMouseOut = (e: MouseEvent) => {
      const target = (e.target as HTMLElement).closest("[data-comment-id]");
      if (target) {
        setHoveredCommentId(null);
      }
    };
    editorDom.addEventListener("mouseover", handleMouseOver);
    editorDom.addEventListener("mouseout", handleMouseOut);
    return () => {
      editorDom.removeEventListener("mouseover", handleMouseOver);
      editorDom.removeEventListener("mouseout", handleMouseOut);
    };
  }, [editorDom]);

  // Highlight hover sync: margin → editor
  const highlightInEditor = (commentId: string, active: boolean) => {
    if (!editorDom) return;
    const els = editorDom.querySelectorAll(`[data-comment-id="${commentId}"]`);
    els.forEach((el) => {
      if (active) el.classList.add("active");
      else el.classList.remove("active");
    });
  };

  const handleResolve = async (commentId: bigint) => {
    await commentClient.resolveComment({ commentId, projectId });
    onRemoveHighlight?.(String(commentId));
    loadComments();
  };

  const handleDelete = async (commentId: bigint) => {
    if (!confirm(t("comments.deleteConfirm"))) return;
    await commentClient.deleteComment({ commentId, projectId });
    onRemoveHighlight?.(String(commentId));
    loadComments();
  };

  const handleReply = async (commentId: bigint) => {
    if (!replyText.trim() || submitting) return;
    setSubmitting(true);
    try {
      await commentClient.createReply({
        commentId,
        body: replyText,
        projectId,
        mentionedUserIds: replyMentionIds,
      });
      setReplyText("");
      setReplyMentionIds([]);
      setReplyingTo(null);
      loadComments();
    } finally {
      setSubmitting(false);
    }
  };

  const toggleReplyComposer = (id: string) => {
    setReplyingTo((prev) => (prev === id ? null : id));
    setReplyText("");
    setReplyMentionIds([]);
  };

  const visibleComments = showResolved ? comments : comments.filter((c) => !c.resolved);

  if (visibleComments.length === 0) return null;

  return (
    <div className="hidden w-72 shrink-0 md:block">
      {/* Show resolved toggle */}
      {comments.some((c) => c.resolved) && (
        <button
          onClick={() => setShowResolved(!showResolved)}
          className="mb-2 flex cursor-pointer items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground"
        >
          <ChevronDown className="size-3" />
          {showResolved ? t("comments.hideResolved") : t("comments.showResolved")}
        </button>
      )}

      <div ref={containerRef} className="relative">
        {visibleComments.map((comment) => {
          const id = String(comment.id);
          const top = positions.get(id);
          if (top === undefined) return null;

          const isExpanded = expandedId === id;
          const isHovered = hoveredCommentId === id;
          const isOwner = currentUserId !== undefined && comment.userId === BigInt(currentUserId);

          return (
            <div
              key={id}
              className={`absolute left-0 right-0 cursor-pointer rounded-lg border px-3 py-2 text-xs transition-all duration-150 ${
                isHovered
                  ? "border-primary/50 bg-accent/50 shadow-md"
                  : "border-border/30 bg-card/50 hover:border-border/60"
              } ${comment.resolved ? "opacity-40" : ""}`}
              style={{ top }}
              onClick={() => setExpandedId(isExpanded ? null : id)}
              onMouseEnter={() => {
                setHoveredCommentId(id);
                highlightInEditor(id, true);
              }}
              onMouseLeave={() => {
                setHoveredCommentId(null);
                highlightInEditor(id, false);
              }}
            >
              {/* Collapsed: avatar + first line */}
              <div className="flex items-center gap-2">
                <div className="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/20 text-[9px] font-bold text-primary uppercase">
                  {comment.userName.charAt(0)}
                </div>
                <span className="font-medium text-foreground/80">{comment.userName}</span>
                <span className="text-muted-foreground">{timeAgo(comment.createdAt)}</span>
              </div>

              {!isExpanded && (
                <div className="mt-1 flex items-center gap-2">
                  <p className="flex-1 truncate text-foreground/70">{comment.body}</p>
                  {comment.replies.length > 0 && (
                    <span className="shrink-0 text-[10px] text-muted-foreground">
                      {comment.replies.length} {t("comments.reply").toLowerCase()}
                    </span>
                  )}
                </div>
              )}

              {/* Expanded */}
              {isExpanded && (
                <div className="mt-2" onClick={(e) => e.stopPropagation()}>
                  {comment.quotedText && (
                    <div className="mb-2 rounded border-l-2 border-yellow-500/50 bg-yellow-500/5 px-2 py-1 text-muted-foreground">
                      &ldquo;{comment.quotedText}&rdquo;
                    </div>
                  )}

                  <p className="text-foreground">{renderMentionBody(comment.body, members)}</p>

                  {/* Replies */}
                  {comment.replies.length > 0 && (
                    <div className="mt-2 space-y-2 border-l border-border/30 pl-2">
                      {comment.replies.map((reply) => (
                        <div key={String(reply.id)}>
                          <div className="flex items-center gap-1">
                            <span className="font-medium">{reply.userName}</span>
                            <span className="text-muted-foreground">
                              {timeAgo(reply.createdAt)}
                            </span>
                          </div>
                          <p className="text-foreground/80">{renderMentionBody(reply.body, members)}</p>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Actions */}
                  <div className="mt-2 flex items-center gap-1">
                    {!comment.resolved && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleResolve(comment.id)}
                        className="h-6 cursor-pointer gap-1 px-1.5 text-[10px] text-muted-foreground"
                      >
                        <Check className="size-3" />
                        {t("comments.resolve")}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => toggleReplyComposer(id)}
                      className="h-6 cursor-pointer px-1.5 text-[10px] text-muted-foreground"
                    >
                      {t("comments.reply")}
                    </Button>
                    {isOwner && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDelete(comment.id)}
                        className="h-6 cursor-pointer px-1.5 text-[10px] text-destructive"
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    )}
                  </div>

                  {/* Reply input */}
                  {replyingTo === id && (
                    <div className="mt-2 flex gap-1">
                      <MentionTextarea
                        value={replyText}
                        onChange={setReplyText}
                        members={members}
                        onMentionedIdsChange={setReplyMentionIds}
                        autoFocus
                        rows={2}
                        submitShortcut="enter"
                        onSubmit={() => handleReply(comment.id)}
                        placeholder={t("comments.replyPlaceholder")}
                        className="flex-1 rounded border border-border/50 bg-transparent px-2 py-1 text-xs outline-none focus:border-primary"
                      />
                      <Button
                        size="sm"
                        onClick={() => handleReply(comment.id)}
                        disabled={!replyText.trim() || submitting}
                        className="h-6 cursor-pointer px-1.5"
                      >
                        <Send className="size-3" />
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
