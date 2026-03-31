"use client";

import { useCallback, useEffect, useRef } from "react";
import { Bot, User } from "lucide-react";
import { ChatProposalResultCard, dispatchApplyProposal } from "./chat-proposal-result";
import { ChatACResultCard, dispatchApplyAC } from "./chat-ac-result";
import { ChatToolConfirm } from "./chat-tool-confirm";
import type { ChatMessage, ChatProposalResult, ChatACResult } from "./types";

interface ChatMessageListProps {
  messages: ChatMessage[];
  isStreaming: boolean;
  onMarkApplied: (messageId: string) => void;
  onToolConfirm?: (messageId: string) => void;
  onToolReject?: (messageId: string) => void;
  toolExecuting?: boolean;
}

function isProposalResult(result: unknown): result is ChatProposalResult {
  return !!result && typeof result === "object" && "problem" in result;
}

function isACResult(result: unknown): result is ChatACResult[] {
  return Array.isArray(result) && result.length > 0 && "scenario" in result[0];
}

export function ChatMessageList({ messages, isStreaming, onMarkApplied, onToolConfirm, onToolReject, toolExecuting }: ChatMessageListProps) {
  const endRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const isNearBottom =
      container.scrollHeight - container.scrollTop - container.clientHeight < 100;
    if (isNearBottom) {
      endRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

  const handleApplyProposal = useCallback(
    (msgId: string, result: ChatProposalResult) => {
      dispatchApplyProposal(result);
      onMarkApplied(msgId);
    },
    [onMarkApplied],
  );

  const handleApplyAC = useCallback(
    (msgId: string, selected: ChatACResult[]) => {
      dispatchApplyAC(selected);
      onMarkApplied(msgId);
    },
    [onMarkApplied],
  );

  return (
    <div ref={containerRef} className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
      {messages.map((msg) => (
        <div key={msg.id} className={`flex gap-2.5 ${msg.role === "user" ? "flex-row-reverse" : ""}`}>
          {/* Avatar */}
          <div
            className={`flex size-6 shrink-0 items-center justify-center rounded-full ${
              msg.role === "user"
                ? "bg-primary/10 text-primary"
                : "bg-muted text-muted-foreground"
            }`}
          >
            {msg.role === "user" ? (
              <User className="size-3.5" />
            ) : (
              <Bot className="size-3.5" />
            )}
          </div>

          {/* Message bubble */}
          <div
            className={`max-w-[85%] rounded-xl px-3.5 py-2.5 text-sm ${
              msg.role === "user"
                ? "bg-primary text-primary-foreground"
                : "bg-muted/50 text-foreground"
            }`}
          >
            <div className="whitespace-pre-wrap break-words">{msg.content}</div>

            {/* Streaming cursor */}
            {msg.role === "assistant" && isStreaming && msg === messages[messages.length - 1] && !msg.result && (
              <span className="ml-0.5 inline-block h-4 w-0.5 animate-pulse bg-current" />
            )}

            {/* Proposal result card */}
            {msg.result && isProposalResult(msg.result) && (
              <ChatProposalResultCard
                result={msg.result}
                appliedAt={msg.appliedAt}
                onApply={() => handleApplyProposal(msg.id, msg.result as ChatProposalResult)}
              />
            )}

            {/* AC result card */}
            {msg.result && isACResult(msg.result) && (
              <ChatACResultCard
                results={msg.result}
                appliedAt={msg.appliedAt}
                onApply={(selected) => handleApplyAC(msg.id, selected)}
              />
            )}

            {/* Tool confirmation card */}
            {msg.pendingToolCall && (
              <ChatToolConfirm
                toolCall={msg.pendingToolCall}
                onConfirm={() => onToolConfirm?.(msg.id)}
                onReject={() => onToolReject?.(msg.id)}
                executed={msg.toolExecuted}
                disabled={toolExecuting}
              />
            )}
          </div>
        </div>
      ))}

      {/* Typing indicator */}
      {isStreaming && messages.length > 0 && messages[messages.length - 1].content === "" && !messages[messages.length - 1].result && (
        <div className="flex gap-2.5">
          <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
            <Bot className="size-3.5" />
          </div>
          <div className="flex items-center gap-1 rounded-xl bg-muted/50 px-3.5 py-2.5">
            <div className="size-1.5 animate-pulse rounded-full bg-muted-foreground/50" />
            <div className="size-1.5 animate-pulse rounded-full bg-muted-foreground/50 [animation-delay:150ms]" />
            <div className="size-1.5 animate-pulse rounded-full bg-muted-foreground/50 [animation-delay:300ms]" />
          </div>
        </div>
      )}

      <div ref={endRef} />
    </div>
  );
}
