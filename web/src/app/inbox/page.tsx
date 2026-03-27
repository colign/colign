"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/layout/header";
import { useI18n } from "@/lib/i18n";
import { notificationClient } from "@/lib/notification";
import { NOTIFICATIONS_UPDATED_EVENT } from "@/lib/notification-events";
import { orgClient } from "@/lib/organization";
import { getTokenPayload, saveTokens } from "@/lib/auth";
import { showError } from "@/lib/toast";
import {
  Eye,
  MessageSquare,
  AtSign,
  ArrowRightLeft,
  UserPlus,
  Check,
  CheckCheck,
  Inbox,
} from "lucide-react";

type NotificationType = "review_request" | "comment" | "mention" | "stage_change" | "invite";
type FilterType = "all" | "unread" | NotificationType;

interface Notification {
  id: bigint;
  type: string;
  read: boolean;
  actorName: string;
  changeName: string;
  changeId: bigint;
  projectName: string;
  projectSlug: string;
  organizationId: bigint;
  stage: string;
  commentPreview: string;
  createdAt?: { seconds: bigint };
}

const stageConfig: Record<string, { label: string; color: string }> = {
  draft: { label: "Draft", color: "text-amber-400" },
  design: { label: "Design", color: "text-blue-400" },
  review: { label: "Review", color: "text-violet-400" },
  ready: { label: "Ready", color: "text-emerald-400" },
};

function timeAgo(seconds: bigint | undefined): string {
  if (!seconds) return "";
  const now = Math.floor(Date.now() / 1000);
  const diff = now - Number(seconds);
  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  return `${Math.floor(diff / 604800)}w ago`;
}

const typeIcon: Record<string, typeof Eye> = {
  review_request: Eye,
  comment: MessageSquare,
  mention: AtSign,
  stage_change: ArrowRightLeft,
  invite: UserPlus,
};

const typeColor: Record<string, string> = {
  review_request: "text-violet-400 bg-violet-400/10",
  comment: "text-blue-400 bg-blue-400/10",
  mention: "text-amber-400 bg-amber-400/10",
  stage_change: "text-emerald-400 bg-emerald-400/10",
  invite: "text-primary bg-primary/10",
};

export default function InboxPage() {
  const { t } = useI18n();
  const router = useRouter();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [filter, setFilter] = useState<FilterType>("all");
  const [loading, setLoading] = useState(true);

  function notifyUnreadChanged() {
    window.dispatchEvent(new CustomEvent(NOTIFICATIONS_UPDATED_EVENT));
  }

  async function load() {
    try {
      const res = await notificationClient.listNotifications({ filter });
      setNotifications(
        res.notifications.map((n) => ({
          id: n.id,
          type: n.type,
          read: n.read,
          actorName: n.actorName,
          changeName: n.changeName,
          changeId: n.changeId,
          projectName: n.projectName,
          projectSlug: n.projectSlug,
          organizationId: n.organizationId,
          stage: n.stage,
          commentPreview: n.commentPreview,
          createdAt: n.createdAt ? { seconds: n.createdAt.seconds } : undefined,
        })),
      );
      setUnreadCount(res.unreadCount);
    } catch (err) {
      showError("Failed to load notifications", err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, [filter]);

  function updateNotificationReadState(id: bigint, read: boolean) {
    setNotifications((prev) => {
      if (filter === "unread" && read) {
        return prev.filter((n) => n.id !== id);
      }
      return prev.map((n) => (n.id === id ? { ...n, read } : n));
    });
    setUnreadCount((prev) => Math.max(0, prev + (read ? -1 : 1)));
    notifyUnreadChanged();
  }

  async function toggleRead(id: bigint, currentRead: boolean) {
    try {
      const nextRead = !currentRead;
      await notificationClient.markRead({ id, read: nextRead });
      updateNotificationReadState(id, nextRead);
    } catch (err) {
      showError("Failed to load notifications", err);
    }
  }

  async function markAllRead() {
    try {
      await notificationClient.markAllRead({});
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
      setUnreadCount(0);
      notifyUnreadChanged();
    } catch (err) {
      showError("Failed to load notifications", err);
    }
  }

  function getNotificationHref(n: Notification): string {
    if (n.type === "invite") return `/projects/${n.projectSlug}`;
    if (n.changeId) return `/projects/${n.projectSlug}/changes/${n.changeId}`;
    return `/projects/${n.projectSlug}`;
  }

  async function navigateToNotification(n: Notification) {
    const targetHref = getNotificationHref(n);
    const currentOrgId = getTokenPayload()?.org_id;

    try {
      if (!n.read) {
        await notificationClient.markRead({ id: n.id, read: true });
        updateNotificationReadState(n.id, true);
      }
      if (n.organizationId > 0n && currentOrgId !== Number(n.organizationId)) {
        const res = await orgClient.switchOrganization({ organizationId: n.organizationId });
        saveTokens(res.accessToken, res.refreshToken);
      }
      router.push(targetHref);
    } catch (err) {
      showError("Failed to open notification target", err);
    }
  }

  function renderNotificationText(n: Notification) {
    switch (n.type) {
      case "review_request":
        return (
          <p className="text-sm">
            <span className="font-medium text-foreground">{n.actorName}</span>
            <span className="text-muted-foreground"> {t("inbox.requestedReview")} </span>
            <span className="font-medium text-foreground">{n.changeName}</span>
          </p>
        );
      case "comment":
        return (
          <div>
            <p className="text-sm">
              <span className="font-medium text-foreground">{n.actorName}</span>
              <span className="text-muted-foreground"> {t("inbox.commentedOn")} </span>
              <span className="font-medium text-foreground">{n.changeName}</span>
            </p>
            {n.commentPreview && (
              <p className="mt-0.5 truncate text-xs text-muted-foreground/70">{n.commentPreview}</p>
            )}
          </div>
        );
      case "mention":
        return (
          <div>
            <p className="text-sm">
              <span className="font-medium text-foreground">{n.actorName}</span>
              <span className="text-muted-foreground"> {t("inbox.mentionedYou")} </span>
              <span className="font-medium text-foreground">{n.changeName}</span>
            </p>
            {n.commentPreview && (
              <p className="mt-0.5 truncate text-xs text-muted-foreground/70">{n.commentPreview}</p>
            )}
          </div>
        );
      case "stage_change": {
        const stage = stageConfig[n.stage] ?? stageConfig.draft;
        return (
          <p className="text-sm">
            <span className="font-medium text-foreground">{n.changeName}</span>
            <span className="text-muted-foreground"> {t("inbox.movedTo")} </span>
            <span className={`font-medium ${stage.color}`}>{stage.label}</span>
          </p>
        );
      }
      case "invite":
        return (
          <p className="text-sm">
            <span className="font-medium text-foreground">{n.actorName}</span>
            <span className="text-muted-foreground"> {t("inbox.invitedYou")} </span>
            <span className="font-medium text-foreground">{n.projectName}</span>
          </p>
        );
      default:
        return null;
    }
  }

  const filters: { id: FilterType; label: string; count?: number }[] = [
    { id: "all", label: t("inbox.all"), count: notifications.length },
    { id: "unread", label: t("inbox.unread"), count: unreadCount },
    { id: "review_request", label: t("inbox.reviewRequests") },
    { id: "comment", label: t("inbox.comments") },
    { id: "mention", label: t("inbox.mentions") },
    { id: "stage_change", label: t("inbox.stageChanges") },
    { id: "invite", label: t("inbox.invites") },
  ];

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <Header breadcrumbs={[{ label: t("inbox.title") }]} />

      <main className="mx-auto max-w-4xl px-6 pt-8 pb-16">
        {/* Header row */}
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold tracking-tight">{t("inbox.title")}</h1>
            {unreadCount > 0 && (
              <span className="rounded-full bg-primary px-2 py-0.5 text-xs font-medium text-primary-foreground">
                {unreadCount}
              </span>
            )}
          </div>
          {unreadCount > 0 && (
            <button
              onClick={markAllRead}
              className="flex cursor-pointer items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <CheckCheck className="size-3.5" />
              {t("inbox.markAllRead")}
            </button>
          )}
        </div>

        {/* Filter tabs */}
        <div className="mb-5 flex gap-1 overflow-x-auto border-b border-border/40 pb-px">
          {filters.map((f) => (
            <button
              key={f.id}
              onClick={() => setFilter(f.id)}
              className={`cursor-pointer whitespace-nowrap rounded-t-md px-3 py-2 text-xs font-medium transition-colors ${
                filter === f.id
                  ? "border-b-2 border-primary text-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {f.label}
              {f.count !== undefined && f.count > 0 && (
                <span className="ml-1.5 text-muted-foreground/60">{f.count}</span>
              )}
            </button>
          ))}
        </div>

        {/* Notification list */}
        <div className="rounded-xl border border-border/40 bg-card/50">
          {notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20">
              <div className="mb-4 rounded-2xl bg-muted/50 p-4">
                <Inbox className="size-8 text-muted-foreground/30" />
              </div>
              <p className="text-sm font-medium text-foreground/70">{t("inbox.noNotifications")}</p>
              <p className="mt-1 text-xs text-muted-foreground">{t("inbox.noNotificationsDesc")}</p>
            </div>
          ) : (
            <div className="divide-y divide-border/20">
              {notifications.map((notification) => {
                const Icon = typeIcon[notification.type] || Eye;
                const colorClass = typeColor[notification.type] || typeColor.comment;
                return (
                  <div
                    key={String(notification.id)}
                    className={`group flex items-start gap-4 px-5 py-4 transition-colors hover:bg-accent/30 ${
                      !notification.read ? "bg-primary/[0.02]" : ""
                    }`}
                  >
                    {/* Unread dot */}
                    <div className="flex h-6 w-2 shrink-0 items-center">
                      {!notification.read && <div className="h-2 w-2 rounded-full bg-primary" />}
                    </div>

                    {/* Icon */}
                    <div
                      className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${colorClass}`}
                    >
                      <Icon className="size-4" />
                    </div>

                    {/* Content */}
                    <button
                      type="button"
                      onClick={() => void navigateToNotification(notification)}
                      className="min-w-0 flex-1 cursor-pointer"
                    >
                      {renderNotificationText(notification)}
                      <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground/50">
                        <span>{notification.projectName}</span>
                        <span>·</span>
                        <span>{timeAgo(notification.createdAt?.seconds)}</span>
                      </div>
                    </button>

                    {/* Actions */}
                    <button
                      onClick={() => toggleRead(notification.id, notification.read)}
                      className="shrink-0 cursor-pointer rounded-md p-1.5 text-muted-foreground/40 opacity-0 transition-all hover:bg-accent hover:text-foreground group-hover:opacity-100"
                      title={notification.read ? t("inbox.markUnread") : t("inbox.markRead")}
                    >
                      <Check className="size-3.5" />
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
