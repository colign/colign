"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { MessageSquare, PanelRightClose, PanelRightOpen, RotateCcw, Sparkles } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { useAIPanel } from "./ai-panel-context";
import { ChatMessageList } from "./chat-message-list";
import { ChatInput } from "./chat-input";
import { streamChat, loadChatHistory, deleteChatSession } from "@/lib/ai";
import { showError } from "@/lib/toast";
import type { AIChatMode, ChatMessage } from "./types";

interface AISidePanelProps {
  changeId: bigint;
  projectId: bigint;
}

const modeI18nKeys: Record<AIChatMode, string> = {
  proposal: "ai.modeProposal",
  ac: "ai.modeAC",
  general: "ai.modeGeneral",
};

export function AISidePanel({ changeId, projectId }: AISidePanelProps) {
  const { isOpen, mode, close, setMode } = useAIPanel();
  const { t } = useI18n();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [historyLoaded, setHistoryLoaded] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  void projectId;

  // Load chat history on mount
  useEffect(() => {
    if (historyLoaded) return;
    let cancelled = false;

    async function load() {
      try {
        const history = await loadChatHistory(changeId);
        if (cancelled || !history) return;
        setMessages(
          history.messages.map((m) => ({
            id: m.id,
            role: m.role,
            content: m.content,
            result: m.result as ChatMessage["result"],
            appliedAt: m.appliedAt,
          })),
        );
      } catch {
        // History endpoint may not exist yet — silently ignore
      } finally {
        if (!cancelled) setHistoryLoaded(true);
      }
    }

    load();
    return () => { cancelled = true; };
  }, [changeId, historyLoaded]);

  const handleSend = useCallback(
    async (content: string) => {
      if (isStreaming) return;

      const userMessage: ChatMessage = {
        id: crypto.randomUUID(),
        role: "user",
        content,
      };

      const assistantMessage: ChatMessage = {
        id: crypto.randomUUID(),
        role: "assistant",
        content: "",
      };

      setMessages((prev) => [...prev, userMessage, assistantMessage]);
      setIsStreaming(true);

      abortRef.current?.abort();
      const abort = new AbortController();
      abortRef.current = abort;

      try {
        const MAX_HISTORY = 20;
        const history = [...messages, userMessage]
          .slice(-MAX_HISTORY)
          .map((m) => ({
            role: m.role,
            content: m.content,
          }));

        for await (const chunk of streamChat(changeId, history, mode, abort.signal)) {
          if (abort.signal.aborted) break;

          if (chunk.content) {
            setMessages((prev) => {
              const updated = [...prev];
              const last = updated[updated.length - 1];
              updated[updated.length - 1] = {
                ...last,
                content: last.content + chunk.content,
              };
              return updated;
            });
          }

          if (chunk.result) {
            setMessages((prev) => {
              const updated = [...prev];
              const last = updated[updated.length - 1];
              updated[updated.length - 1] = {
                ...last,
                result: chunk.result as ChatMessage["result"],
              };
              return updated;
            });
          }
        }
      } catch (err) {
        if (abort.signal.aborted) return;
        showError(t("ai.connectionError"), err);
        // Remove the empty assistant message on error
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last?.role === "assistant" && last.content === "") {
            return prev.slice(0, -1);
          }
          return prev;
        });
      } finally {
        setIsStreaming(false);
      }
    },
    [changeId, mode, messages, isStreaming, t],
  );

  const handleMarkApplied = useCallback((messageId: string) => {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === messageId ? { ...m, appliedAt: new Date().toISOString() } : m,
      ),
    );
  }, []);

  function handleNewConversation() {
    abortRef.current?.abort();
    setMessages([]);
    setIsStreaming(false);
    // Delete server-side session in background
    deleteChatSession(changeId).catch(() => {});
  }

  return (
    <div
      className={`shrink-0 border-l border-border/40 transition-[width] duration-200 ease-out ${
        isOpen ? "w-[400px]" : "w-0"
      }`}
    >
      <div
        className={`flex h-full w-[400px] flex-col overflow-hidden transition-opacity duration-200 ${
          isOpen ? "opacity-100" : "pointer-events-none opacity-0"
        }`}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border/40 px-4 py-3">
          <div className="flex items-center gap-2">
            <Sparkles className="size-4 text-primary" />
            <span className="text-sm font-medium">{t("ai.chatTitle")}</span>
          </div>
          <div className="flex items-center gap-1">
            {messages.length > 0 && (
              <button
                onClick={handleNewConversation}
                className="cursor-pointer rounded-md p-1 text-muted-foreground transition-colors hover:text-foreground"
                title={t("ai.newConversation")}
              >
                <RotateCcw className="size-3.5" />
              </button>
            )}
            <button
              onClick={close}
              className="cursor-pointer rounded-md p-1 text-muted-foreground transition-colors hover:text-foreground"
              title={t("ai.closePanel")}
            >
              <PanelRightClose className="size-4" />
            </button>
          </div>
        </div>

        {/* Mode Selector */}
        <div className="flex gap-1 border-b border-border/40 px-4 py-2">
          {(["proposal", "ac", "general"] as const).map((m) => (
            <button
              key={m}
              onClick={() => setMode(m)}
              className={`cursor-pointer rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                mode === m
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {t(modeI18nKeys[m])}
            </button>
          ))}
        </div>

        {/* Chat Area */}
        {messages.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center px-6">
            <div className="flex size-12 items-center justify-center rounded-full bg-primary/10">
              <Sparkles className="size-6 text-primary" />
            </div>
            <p className="mt-4 text-sm font-medium text-foreground">
              {t("ai.chatEmptyTitle")}
            </p>
            <p className="mt-1.5 text-center text-xs leading-relaxed text-muted-foreground">
              {t("ai.chatEmptyDescription")}
            </p>
          </div>
        ) : (
          <ChatMessageList messages={messages} isStreaming={isStreaming} onMarkApplied={handleMarkApplied} />
        )}

        {/* Input */}
        <ChatInput onSend={handleSend} disabled={isStreaming} />
      </div>
    </div>
  );
}

export function AIPanelToggle() {
  const { isOpen, toggle } = useAIPanel();
  const { t } = useI18n();

  if (isOpen) return null;

  return (
    <button
      onClick={toggle}
      className="fixed bottom-6 right-6 z-30 flex cursor-pointer items-center gap-2 rounded-full border border-primary/20 bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground shadow-lg transition-all hover:shadow-xl hover:brightness-110"
    >
      <Sparkles className="size-4" />
      {t("ai.openPanel")}
    </button>
  );
}
