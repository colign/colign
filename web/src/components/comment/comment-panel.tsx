"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";
import { commentClient } from "@/lib/comment";
import { MessageSquare, Check, Trash2, ChevronDown, ChevronUp, Send } from "lucide-react";
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

interface CommentPanelProps {
  changeId: bigint;
  projectId: bigint;
  documentType: string;
  currentUserId?: number;
  showCompose?: boolean;
  members?: MentionMember[];
  onCommentClick?: (commentId: string) => void;
  refreshRef?: React.MutableRefObject<(() => void) | null>;
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

export function CommentPanel({
  changeId,
  projectId,
  documentType,
  currentUserId,
  showCompose,
  members = [],
  onCommentClick,
  refreshRef,
}: CommentPanelProps) {
  const { t } = useI18n();
  const [comments, setComments] = useState<CommentData[]>([]);
  const [showResolved, setShowResolved] = useState(false);
  const [expandedThreads, setExpandedThreads] = useState<Set<string>>(new Set());
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [replyText, setReplyText] = useState("");
  const [composeText, setComposeText] = useState("");
  const [replyMentionIds, setReplyMentionIds] = useState<bigint[]>([]);
  const [composeMentionIds, setComposeMentionIds] = useState<bigint[]>([]);
  const submittingRef = useRef(false);

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

  // Expose refresh for parent
  useEffect(() => {
    if (refreshRef) refreshRef.current = loadComments;
  }, [refreshRef, loadComments]);

  const handleResolve = async (commentId: bigint) => {
    await commentClient.resolveComment({ commentId, projectId });
    if (replyingTo === String(commentId)) {
      setReplyingTo(null);
      setReplyText("");
    }
    loadComments();
  };

  const handleDelete = async (commentId: bigint) => {
    if (!confirm(t("comments.deleteConfirm"))) return;
    await commentClient.deleteComment({ commentId, projectId });
    loadComments();
  };

  const handleReply = async (commentId: bigint) => {
    if (!replyText.trim() || submittingRef.current) return;
    const targetComment = comments.find((comment) => comment.id === commentId);
    if (!targetComment || targetComment.resolved) {
      setReplyingTo(null);
      setReplyText("");
      return;
    }
    submittingRef.current = true;
    try {
      await commentClient.createReply({
        commentId,
        body: replyText,
        projectId,
        mentionedUserIds: replyMentionIds,
      });
      const threadId = String(commentId);
      setReplyText("");
      setReplyMentionIds([]);
      setReplyingTo(null);
      setExpandedThreads((prev) => new Set(prev).add(threadId));
      loadComments();
    } finally {
      submittingRef.current = false;
    }
  };

  const handleCompose = async () => {
    if (!composeText.trim() || submittingRef.current) return;
    submittingRef.current = true;
    try {
      await commentClient.createComment({
        changeId,
        documentType,
        quotedText: "",
        body: composeText,
        projectId,
        mentionedUserIds: composeMentionIds,
      });
      setComposeText("");
      setComposeMentionIds([]);
      loadComments();
    } catch (err) {
      showError(t("toast.saveFailed"), err);
    } finally {
      submittingRef.current = false;
    }
  };

  const toggleThread = (id: string) => {
    setExpandedThreads((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleReplyComposer = (id: string) => {
    setReplyingTo((prev) => (prev === id ? null : id));
    setReplyText("");
    setReplyMentionIds([]);
  };

  const visibleComments = showResolved
    ? comments
    : comments.filter((c) => !c.resolved);

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border/50 px-4 py-3">
        <div className="flex items-center gap-2">
          <MessageSquare className="size-4 text-muted-foreground" />
          <span className="text-sm font-medium">{t("comments.comments")}</span>
          <span className="text-xs text-muted-foreground">({comments.filter((c) => !c.resolved).length})</span>
        </div>
        <button
          onClick={() => setShowResolved(!showResolved)}
          className="cursor-pointer text-xs text-muted-foreground hover:text-foreground"
        >
          {showResolved ? t("comments.hideResolved") : t("comments.showResolved")}
        </button>
      </div>

      {/* Comments list */}
      <div className="flex-1 overflow-y-auto">
        {visibleComments.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12">
            <MessageSquare className="size-8 text-muted-foreground/30" />
            <p className="mt-2 text-sm text-muted-foreground">{t("comments.noComments")}</p>
          </div>
        ) : (
          <div className="divide-y divide-border/30">
            {visibleComments.map((comment) => {
              const id = String(comment.id);
              const isExpanded = expandedThreads.has(id);
              const isOwner = currentUserId !== undefined && comment.userId === BigInt(currentUserId);

              return (
                <div
                  key={id}
                  className={`px-4 py-3 transition-colors hover:bg-accent/30 ${comment.resolved ? "opacity-50" : ""}`}
                >
                  {/* Comment header */}
                  <div className="flex items-center gap-2">
                    <div className="flex size-6 items-center justify-center rounded-full bg-primary/20 text-[10px] font-bold text-primary uppercase">
                      {comment.userName.charAt(0)}
                    </div>
                    <span className="text-xs font-medium">{comment.userName}</span>
                    <span className="text-[10px] text-muted-foreground">{timeAgo(comment.createdAt)}</span>
                    {comment.resolved && (
                      <span className="ml-auto text-[10px] text-emerald-400">{t("comments.resolved")}</span>
                    )}
                  </div>

                  {/* Quoted text */}
                  {comment.quotedText && (
                    <div
                      className="mt-2 cursor-pointer rounded border-l-2 border-yellow-500/50 bg-yellow-500/5 px-2 py-1 text-xs text-muted-foreground"
                      onClick={() => onCommentClick?.(id)}
                    >
                      &ldquo;{comment.quotedText}&rdquo;
                    </div>
                  )}

                  {/* Comment body */}
                  <p className="mt-1.5 text-sm">{renderMentionBody(comment.body, members)}</p>

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
                    {!comment.resolved && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleReplyComposer(id)}
                        className="h-6 cursor-pointer gap-1 px-1.5 text-[10px] text-muted-foreground"
                      >
                        {t("comments.reply")}
                      </Button>
                    )}
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

                    {/* Thread toggle */}
                    {comment.replies.length > 0 && (
                      <button
                        onClick={() => toggleThread(id)}
                        className="ml-auto flex cursor-pointer items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground"
                      >
                        {comment.replies.length} {t("comments.reply").toLowerCase()}
                        {isExpanded ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />}
                      </button>
                    )}
                  </div>

                  {/* Replies */}
                  {isExpanded && comment.replies.length > 0 && (
                    <div className="ml-4 mt-2 space-y-2 border-l border-border/30 pl-3">
                      {comment.replies.map((reply) => (
                        <div key={String(reply.id)}>
                          <div className="flex items-center gap-2">
                            <div className="flex size-5 items-center justify-center rounded-full bg-muted text-[9px] font-bold uppercase">
                              {reply.userName.charAt(0)}
                            </div>
                            <span className="text-[10px] font-medium">{reply.userName}</span>
                            <span className="text-[10px] text-muted-foreground">{timeAgo(reply.createdAt)}</span>
                          </div>
                          <p className="ml-7 text-xs">{renderMentionBody(reply.body, members)}</p>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Reply input */}
                  {replyingTo === id && !comment.resolved && (
                    <div className="mt-2 flex gap-1.5">
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
                        className="flex-1 rounded-md border border-border/50 bg-transparent px-2 py-1 text-xs outline-none focus:border-primary"
                      />
                      <Button
                        size="sm"
                        onClick={() => handleReply(comment.id)}
                        disabled={!replyText.trim()}
                        className="h-7 cursor-pointer px-2"
                      >
                        <Send className="size-3" />
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Compose new comment */}
      {showCompose && (
        <div className="border-t border-border/50 px-4 py-3">
          <MentionTextarea
            value={composeText}
            onChange={setComposeText}
            members={members}
            onMentionedIdsChange={setComposeMentionIds}
            submitShortcut="mod-enter"
            onSubmit={handleCompose}
            placeholder={t("comments.composePlaceholder")}
            rows={2}
            className="w-full resize-none rounded-md border border-border/50 bg-transparent px-2 py-1.5 text-sm outline-none placeholder:text-muted-foreground/50 focus:border-primary"
          />
          <div className="mt-1.5 flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground">⌘+Enter</span>
            <Button
              size="sm"
              onClick={handleCompose}
              disabled={!composeText.trim()}
              className="h-7 cursor-pointer gap-1 px-2"
            >
              <Send className="size-3" />
              {t("comments.send")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
