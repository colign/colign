"use client";

import { useState, useRef, useCallback, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { showError } from "@/lib/toast";
import { useI18n } from "@/lib/i18n";

interface FloatingChatPanelProps {
  children: React.ReactNode;
}

const STORAGE_KEY = "colign-chat-panel";
const MIN_WIDTH = 360;
const MIN_HEIGHT = 400;
const DEFAULT_WIDTH = 400;
const DEFAULT_HEIGHT = 500;

interface PanelState {
  x: number;
  y: number;
  width: number;
  height: number;
}

function loadState(): PanelState | null {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    return saved ? JSON.parse(saved) : null;
  } catch (err) {
    showError("Failed to load chat panel state", err);
    return null;
  }
}

function saveState(state: PanelState) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch (err) {
    showError("Failed to save chat panel state", err);
  }
}

export function FloatingChatPanel({ children }: FloatingChatPanelProps) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{ startX: number; startY: number; startPosX: number; startPosY: number } | null>(null);
  const resizeRef = useRef<{ startX: number; startY: number; startW: number; startH: number } | null>(null);

  const [pos, setPos] = useState<PanelState>(() => {
    const saved = loadState();
    if (saved) return saved;
    return {
      x: typeof window !== "undefined" ? window.innerWidth - DEFAULT_WIDTH - 24 : 100,
      y: typeof window !== "undefined" ? window.innerHeight - DEFAULT_HEIGHT - 24 : 100,
      width: DEFAULT_WIDTH,
      height: DEFAULT_HEIGHT,
    };
  });

  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 768);
    check();
    window.addEventListener("resize", check);
    return () => window.removeEventListener("resize", check);
  }, []);

  useEffect(() => {
    saveState(pos);
  }, [pos]);

  // Clamp to viewport
  const clamp = useCallback((state: PanelState): PanelState => {
    const maxX = window.innerWidth - state.width;
    const maxY = window.innerHeight - state.height;
    return {
      ...state,
      x: Math.max(0, Math.min(state.x, maxX)),
      y: Math.max(0, Math.min(state.y, maxY)),
    };
  }, []);

  // Drag handlers
  const onDragStart = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    dragRef.current = { startX: e.clientX, startY: e.clientY, startPosX: pos.x, startPosY: pos.y };

    const onMove = (ev: PointerEvent) => {
      if (!dragRef.current) return;
      const dx = ev.clientX - dragRef.current.startX;
      const dy = ev.clientY - dragRef.current.startY;
      setPos((prev) =>
        clamp({ ...prev, x: dragRef.current!.startPosX + dx, y: dragRef.current!.startPosY + dy }),
      );
    };
    const onUp = () => {
      dragRef.current = null;
      document.removeEventListener("pointermove", onMove);
      document.removeEventListener("pointerup", onUp);
    };
    document.addEventListener("pointermove", onMove);
    document.addEventListener("pointerup", onUp);
  }, [pos.x, pos.y, clamp]);

  // Resize handlers
  const onResizeStart = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    e.stopPropagation();
    resizeRef.current = { startX: e.clientX, startY: e.clientY, startW: pos.width, startH: pos.height };

    const onMove = (ev: PointerEvent) => {
      if (!resizeRef.current) return;
      const dx = ev.clientX - resizeRef.current.startX;
      const dy = ev.clientY - resizeRef.current.startY;
      setPos((prev) =>
        clamp({
          ...prev,
          width: Math.max(MIN_WIDTH, resizeRef.current!.startW - dx),
          height: Math.max(MIN_HEIGHT, resizeRef.current!.startH - dy),
          x: prev.x + (resizeRef.current!.startW - Math.max(MIN_WIDTH, resizeRef.current!.startW - dx)),
          y: prev.y + (resizeRef.current!.startH - Math.max(MIN_HEIGHT, resizeRef.current!.startH - dy)),
        }),
      );
    };
    const onUp = () => {
      resizeRef.current = null;
      document.removeEventListener("pointermove", onMove);
      document.removeEventListener("pointerup", onUp);
    };
    document.addEventListener("pointermove", onMove);
    document.addEventListener("pointerup", onUp);
  }, [pos.width, pos.height, clamp]);

  // FAB button
  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-6 right-6 z-50 flex h-12 w-12 cursor-pointer items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105 active:scale-95"
      >
        <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"
          />
        </svg>
      </button>
    );
  }

  // Mobile: bottom sheet style
  if (isMobile) {
    return (
      <>
        <div className="fixed inset-0 z-40 bg-black/50" onClick={() => setOpen(false)} />
        <div className="fixed inset-x-0 bottom-0 z-50 flex h-[70vh] flex-col rounded-t-2xl border-t border-border bg-background shadow-2xl">
          <div className="flex items-center justify-between border-b border-border/50 px-4 py-3">
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-medium">{t("change.aiChat")}</h3>
              <span className="inline-flex h-4 items-center rounded-full bg-primary/10 px-1.5 text-[10px] font-medium text-primary">AI</span>
            </div>
            <button onClick={() => setOpen(false)} className="cursor-pointer rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground">
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div className="flex-1 overflow-hidden px-4">{children}</div>
        </div>
      </>
    );
  }

  // Desktop: floating panel
  return (
    <div
      ref={panelRef}
      className="fixed z-50 flex flex-col rounded-xl border border-border bg-background shadow-2xl"
      style={{ left: pos.x, top: pos.y, width: pos.width, height: pos.height }}
    >
      {/* Resize handle (top-left corner) */}
      <div
        onPointerDown={onResizeStart}
        className="absolute -left-1 -top-1 z-10 h-4 w-4 cursor-nw-resize"
      />

      {/* Header (draggable) */}
      <div
        onPointerDown={onDragStart}
        className="flex shrink-0 cursor-grab items-center justify-between border-b border-border/50 px-4 py-2.5 active:cursor-grabbing"
      >
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium">{t("change.aiChat")}</h3>
          <span className="inline-flex h-4 items-center rounded-full bg-primary/10 px-1.5 text-[10px] font-medium text-primary">AI</span>
        </div>
        <button
          onClick={() => setOpen(false)}
          className="cursor-pointer rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden px-4">{children}</div>
    </div>
  );
}
