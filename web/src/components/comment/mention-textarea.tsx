"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@/lib/utils";

export interface MentionMember {
  userId: bigint;
  userName: string;
  userEmail?: string;
}

interface MentionTextareaProps {
  value: string;
  onChange: (value: string) => void;
  members: MentionMember[];
  onMentionedIdsChange?: (ids: bigint[]) => void;
  placeholder?: string;
  className?: string;
  rows?: number;
  autoFocus?: boolean;
  disabled?: boolean;
  onSubmit?: () => void;
  submitShortcut?: "enter" | "mod-enter";
  onEscape?: () => void;
}

interface MentionMatch {
  query: string;
  start: number;
  end: number;
}

function normalizeHandlePart(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "");
}

function getHandleBase(member: MentionMember): string {
  const emailLocalPart = normalizeHandlePart(member.userEmail?.split("@")[0] ?? "");
  if (emailLocalPart) return emailLocalPart;
  return normalizeHandlePart(member.userName.replace(/\s+/g, ""));
}

function getHandleDomainSuffix(member: MentionMember): string {
  const domain = member.userEmail?.split("@")[1] ?? "";
  const firstLabel = domain.split(".")[0] ?? "";
  return normalizeHandlePart(firstLabel);
}

function buildMentionHandleMap(members: MentionMember[]): Map<bigint, string> {
  const bases = new Map<bigint, string>();
  const baseCounts = new Map<string, number>();

  for (const member of members) {
    const base = getHandleBase(member);
    if (!base) continue;
    bases.set(member.userId, base);
    baseCounts.set(base, (baseCounts.get(base) ?? 0) + 1);
  }

  const handles = new Map<bigint, string>();
  const used = new Set<string>();

  for (const member of members) {
    const base = bases.get(member.userId);
    if (!base) continue;

    let handle = base;
    if ((baseCounts.get(base) ?? 0) > 1) {
      const suffix = getHandleDomainSuffix(member);
      if (suffix) {
        handle = `${base}-${suffix}`;
      }
      if (used.has(handle)) {
        handle = `${base}-${String(member.userId)}`;
      }
    }

    used.add(handle);
    handles.set(member.userId, handle);
  }

  return handles;
}

export function getMentionHandle(member: MentionMember, members: MentionMember[] = []): string {
  if (members.length === 0) return getHandleBase(member);
  return buildMentionHandleMap(members).get(member.userId) ?? getHandleBase(member);
}

/**
 * Render comment body with styled mention chips.
 * Parses `@handle` patterns and matches against members list.
 */
export function renderMentionBody(
  body: string,
  members: MentionMember[],
): React.ReactNode {
  if (!body) return null;

  const handleToMember = new Map<string, MentionMember>();
  const handleMap = buildMentionHandleMap(members);
  for (const member of members) {
    const handle = (handleMap.get(member.userId) ?? getHandleBase(member)).toLowerCase();
    if (handle && !handleToMember.has(handle)) {
      handleToMember.set(handle, member);
    }
  }

  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  const regex = /(^|\s)@([^\s@]+)/g;
  let match;

  while ((match = regex.exec(body)) !== null) {
    const prefix = match[1] ?? "";
    const handle = match[2] ?? "";
    const member = handleToMember.get(handle.toLowerCase());

    if (member) {
      const beforeStart = match.index + prefix.length;
      if (lastIndex < beforeStart) {
        parts.push(body.slice(lastIndex, beforeStart));
      }
      parts.push(
        <span
          key={`mention-${beforeStart}`}
          className="inline-flex items-center gap-0.5 rounded bg-primary/15 px-1 py-0.5 text-xs font-medium text-primary cursor-default"
          title={`@${handle}`}
        >
          @{member.userName}
        </span>,
      );
      lastIndex = match.index + match[0].length;
    }
  }

  if (lastIndex < body.length) {
    parts.push(body.slice(lastIndex));
  }

  return parts.length > 0 ? parts : body;
}

/**
 * @deprecated Use onMentionedIdsChange callback from MentionTextarea instead.
 * This helper parses the current text only and cannot distinguish manually
 * typed ambiguous handles from explicitly selected mentions.
 */
export function extractMentionedUserIds(value: string, members: MentionMember[]): bigint[] {
  if (!value.trim()) return [];

  const handleMap = buildMentionHandleMap(members);
  const uniqueByHandle = new Map<string, MentionMember>();
  for (const member of members) {
    const key = (handleMap.get(member.userId) ?? getHandleBase(member)).toLowerCase();
    if (!key) continue;
    uniqueByHandle.set(key, member);
  }

  const mentioned = new Set<bigint>();
  const matches = value.matchAll(/(^|\s)@([^\s@]+)/g);
  for (const match of matches) {
    const key = (match[2] ?? "").trim().toLowerCase();
    const member = uniqueByHandle.get(key);
    if (member) {
      mentioned.add(member.userId);
    }
  }

  return Array.from(mentioned);
}

function getMentionMatch(value: string, caret: number): MentionMatch | null {
  const beforeCaret = value.slice(0, caret);
  const match = /(^|\s)@([^\s@]*)$/.exec(beforeCaret);
  if (!match) return null;
  const query = match[2] ?? "";
  const start = beforeCaret.length - query.length - 1;
  return { query, start, end: beforeCaret.length };
}

/** Extract all @handles present in text */
function extractHandlesFromText(text: string): Set<string> {
  const handles = new Set<string>();
  const matches = text.matchAll(/(^|\s)@([^\s@]+)/g);
  for (const match of matches) {
    handles.add((match[2] ?? "").toLowerCase());
  }
  return handles;
}

export function MentionTextarea({
  value,
  onChange,
  members,
  onMentionedIdsChange,
  placeholder,
  className,
  rows = 2,
  autoFocus,
  disabled,
  onSubmit,
  submitShortcut,
  onEscape,
}: MentionTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [caret, setCaret] = useState(0);
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const [dismissedMentionKey, setDismissedMentionKey] = useState<string | null>(null);
  // Track selected mentions: userId → handle (from the moment of selection)
  const [selectedMentions, setSelectedMentions] = useState<Map<bigint, string>>(new Map());
  const mentionHandles = useMemo(() => buildMentionHandleMap(members), [members]);

  const mentionMatch = useMemo(() => getMentionMatch(value, caret), [value, caret]);
  const mentionKey = mentionMatch ? `${mentionMatch.start}:${mentionMatch.end}:${mentionMatch.query}` : null;
  const suggestions = useMemo(() => {
    if (!mentionMatch) return [];
    const query = mentionMatch.query.trim().toLowerCase();
    const uniqueMembers = members.filter(
      (member, index, list) =>
        !!member.userName &&
        list.findIndex((candidate) => candidate.userId === member.userId) === index,
    );

    return uniqueMembers
      .filter((member) => {
        if (!query) return true;
        const handle = (mentionHandles.get(member.userId) ?? getHandleBase(member)).toLowerCase();
        return member.userName.toLowerCase().includes(query) || handle.includes(query);
      })
      .sort((a, b) => {
        const aHandle = (mentionHandles.get(a.userId) ?? getHandleBase(a)).toLowerCase();
        const bHandle = (mentionHandles.get(b.userId) ?? getHandleBase(b)).toLowerCase();
        const aStarts = query ? aHandle.startsWith(query) || a.userName.toLowerCase().startsWith(query) : false;
        const bStarts = query ? bHandle.startsWith(query) || b.userName.toLowerCase().startsWith(query) : false;
        if (aStarts !== bStarts) return aStarts ? -1 : 1;
        return aHandle.localeCompare(bHandle);
      })
      .slice(0, 8);
  }, [members, mentionHandles, mentionMatch]);
  const isMentionDismissed = mentionKey !== null && mentionKey === dismissedMentionKey;
  const activeSuggestions = isMentionDismissed ? [] : suggestions;
  const safeHighlightedIndex =
    activeSuggestions.length === 0 ? 0 : Math.min(highlightedIndex, activeSuggestions.length - 1);

  // Prune mentions whose @handle no longer appears in the text, and notify parent
  const syncMentions = useCallback(
    (text: string) => {
      if (!onMentionedIdsChange) return;
      const handlesInText = extractHandlesFromText(text);
      setSelectedMentions((prev) => {
        const next = new Map<bigint, string>();
        for (const [userId, handle] of prev) {
          if (handlesInText.has(handle.toLowerCase())) {
            next.set(userId, handle);
          }
        }
        if (next.size !== prev.size) {
          // State update is async; fire callback with the pruned set
          onMentionedIdsChange(Array.from(next.keys()));
          return next;
        }
        return prev;
      });
    },
    [onMentionedIdsChange],
  );

  // Reset mentions when value is cleared externally (e.g. after submit)
  useEffect(() => {
    if (!value.trim() && selectedMentions.size > 0) {
      setSelectedMentions(new Map());
      onMentionedIdsChange?.([]);
    }
  }, [value, selectedMentions.size, onMentionedIdsChange]);

  function syncDismissedMention(nextValue: string, nextCaret: number) {
    if (!dismissedMentionKey) return;
    const nextMatch = getMentionMatch(nextValue, nextCaret);
    const nextKey = nextMatch ? `${nextMatch.start}:${nextMatch.end}:${nextMatch.query}` : null;
    if (nextKey !== dismissedMentionKey) {
      setDismissedMentionKey(null);
      setHighlightedIndex(0);
    }
  }

  function syncCaret() {
    const el = textareaRef.current;
    if (!el) return;
    const nextCaret = el.selectionStart ?? 0;
    syncDismissedMention(value, nextCaret);
    setCaret(nextCaret);
  }

  function selectMember(index: number) {
    const match = mentionMatch;
    const member = activeSuggestions[index];
    const el = textareaRef.current;
    if (!match || !member || !el) return;

    const handle = mentionHandles.get(member.userId) ?? getHandleBase(member);
    const nextValue = `${value.slice(0, match.start)}@${handle} ${value.slice(match.end)}`;
    const nextCaret = match.start + handle.length + 2;

    // Track this mention by userId (handle collision safe)
    setSelectedMentions((prev) => {
      const next = new Map(prev);
      next.set(member.userId, handle);
      onMentionedIdsChange?.(Array.from(next.keys()));
      return next;
    });

    setDismissedMentionKey(null);
    onChange(nextValue);
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(nextCaret, nextCaret);
      setCaret(nextCaret);
    });
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (activeSuggestions.length > 0 && mentionMatch) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setHighlightedIndex((prev) => (prev + 1) % activeSuggestions.length);
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setHighlightedIndex((prev) => (prev - 1 + activeSuggestions.length) % activeSuggestions.length);
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        selectMember(safeHighlightedIndex);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setDismissedMentionKey(mentionKey);
        return;
      }
    }

    if (submitShortcut === "enter" && e.key === "Enter" && !e.shiftKey && !e.metaKey && !e.ctrlKey) {
      e.preventDefault();
      onSubmit?.();
      return;
    }

    if (submitShortcut === "mod-enter" && e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      onSubmit?.();
      return;
    }

    if (e.key === "Escape") {
      onEscape?.();
    }
  }

  return (
    <div className="relative">
      <textarea
        ref={textareaRef}
        autoFocus={autoFocus}
        value={value}
        disabled={disabled}
        rows={rows}
        placeholder={placeholder}
        onChange={(e) => {
          onChange(e.target.value);
          const nextCaret = e.target.selectionStart ?? 0;
          syncDismissedMention(e.target.value, nextCaret);
          setCaret(nextCaret);
          syncMentions(e.target.value);
        }}
        onClick={syncCaret}
        onKeyUp={syncCaret}
        onSelect={syncCaret}
        onKeyDown={handleKeyDown}
        className={className}
      />
      {mentionMatch && activeSuggestions.length > 0 && (
        <div className="absolute left-0 right-0 top-full z-40 mt-1 overflow-hidden rounded-md border border-border/60 bg-popover shadow-lg">
          <div className="max-h-56 overflow-y-auto py-1">
            {activeSuggestions.map((member, index) => (
              <button
                key={String(member.userId)}
                type="button"
                onMouseDown={(e) => {
                  e.preventDefault();
                  selectMember(index);
                }}
                className={cn(
                  "flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent",
                  index === safeHighlightedIndex && "bg-accent",
                )}
              >
                <div className="flex size-6 items-center justify-center rounded-full bg-primary/15 text-[10px] font-bold uppercase text-primary">
                  {member.userName.charAt(0)}
                </div>
                <div className="min-w-0">
                  <div className="truncate">{member.userName}</div>
                  <div className="truncate text-xs text-muted-foreground">
                    @{mentionHandles.get(member.userId) ?? getHandleBase(member)}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
