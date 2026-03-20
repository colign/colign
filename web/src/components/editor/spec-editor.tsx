"use client";

import { useEditor, EditorContent } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Collaboration from "@tiptap/extension-collaboration";
import CollaborationCursor from "@tiptap/extension-collaboration-cursor";
import { HocuspocusProvider } from "@hocuspocus/provider";
import { CommentHighlight } from "./extensions/comment-highlight";
import { useCallback, useEffect, useRef, useMemo } from "react";
import {
  Bold,
  Italic,
  Heading2,
  Heading3,
  List,
  Code,
  MessageSquarePlus,
  Wifi,
  WifiOff,
} from "lucide-react";
import { useState } from "react";
import { useI18n } from "@/lib/i18n";
import { getAccessToken } from "@/lib/auth";
import * as Y from "yjs";

// Generate a consistent color from user name
function userColor(name: string): string {
  const colors = ["#3b82f6", "#ef4444", "#22c55e", "#f59e0b", "#8b5cf6", "#ec4899", "#06b6d4", "#f97316"];
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash);
  return colors[Math.abs(hash) % colors.length];
}

interface SpecEditorProps {
  initialContent?: string;
  placeholder?: string;
  onSave?: (content: string) => void;
  readOnly?: boolean;
  onAddComment?: (quotedText: string) => void;
  onHighlightClick?: (commentId: string) => void;
  editorRef?: React.MutableRefObject<{
    addHighlightAtSavedSelection: (commentId: string) => void;
    removeHighlight: (commentId: string) => void;
    scrollToHighlight: (commentId: string) => void;
    getEditorDom: () => HTMLElement | null;
  } | null>;
  documentId?: string;
  userName?: string;
}

export function SpecEditor({
  initialContent = "",
  placeholder = "Start writing...",
  onSave,
  readOnly = false,
  onAddComment,
  onHighlightClick,
  editorRef,
  documentId,
  userName = "Anonymous",
}: SpecEditorProps) {
  const { t } = useI18n();
  const [saveStatus, setSaveStatus] = useState<"saved" | "saving" | "error" | "idle">("idle");
  const [connectionStatus, setConnectionStatus] = useState<"connected" | "disconnected" | "connecting">("connecting");
  const savedSelectionRef = useRef<{ from: number; to: number } | null>(null);

  const isCollaborative = !!documentId;
  const hocuspocusUrl = process.env.NEXT_PUBLIC_HOCUSPOCUS_URL ?? "ws://localhost:1234";

  // Y.js doc and provider (only for collaborative mode)
  const [ydoc, setYdoc] = useState<Y.Doc | null>(null);
  const [provider, setProvider] = useState<HocuspocusProvider | null>(null);

  useEffect(() => {
    if (!isCollaborative || !documentId) return;

    const doc = new Y.Doc();
    const prov = new HocuspocusProvider({
      url: hocuspocusUrl,
      name: documentId,
      document: doc,
      token: getAccessToken() ?? undefined,
    });

    setYdoc(doc);
    setProvider(prov);

    return () => {
      prov.destroy();
      doc.destroy();
      setYdoc(null);
      setProvider(null);
    };
  }, [isCollaborative, documentId, hocuspocusUrl]);

  // Track connection status
  useEffect(() => {
    if (!provider) {
      setConnectionStatus("disconnected");
      return;
    }
    const onStatus = ({ status }: { status: string }) => {
      if (status === "connected") setConnectionStatus("connected");
      else if (status === "connecting") setConnectionStatus("connecting");
      else setConnectionStatus("disconnected");
    };
    provider.on("status", onStatus);
    return () => {
      provider.off("status", onStatus);
    };
  }, [provider]);


  // Build extensions
  const canCollaborate = isCollaborative && ydoc != null && provider != null;

  const extensions = useMemo(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const exts: any[] = [
      canCollaborate
        ? StarterKit.configure({ history: false } as any)
        : StarterKit,
      Placeholder.configure({ placeholder }),
      CommentHighlight,
    ];

    if (canCollaborate) {
      exts.push(Collaboration.configure({ document: ydoc! }));
      exts.push(
        CollaborationCursor.configure({
          provider: provider!,
          user: { name: userName, color: userColor(userName) },
        }),
      );
    }

    return exts;
  }, [canCollaborate, ydoc, provider, placeholder, userName]);

  const editor = useEditor({
    extensions,
    content: canCollaborate ? undefined : initialContent,
    editable: !readOnly,
    immediatelyRender: false,
    onUpdate: ({ editor }) => {
      debouncedSave(editor.getHTML());
    },
  }, [canCollaborate]);

  // For non-collaborative mode: set initial content
  useEffect(() => {
    if (!canCollaborate && editor && initialContent) {
      editor.commands.setContent(initialContent);
    }
  }, [canCollaborate, initialContent, editor]);

  // Expose editor methods via ref
  useEffect(() => {
    if (!editor || !editorRef) return;
    editorRef.current = {
      addHighlightAtSavedSelection: (commentId: string) => {
        const sel = savedSelectionRef.current;
        if (!sel) return;
        editor
          .chain()
          .focus()
          .setTextSelection(sel)
          .setCommentHighlight({ commentId })
          .run();
        savedSelectionRef.current = null;
        if (onSave) debouncedSave(editor.getHTML());
      },
      removeHighlight: (commentId: string) => {
        editor.chain().focus().unsetCommentHighlight(commentId).run();
        if (onSave) debouncedSave(editor.getHTML());
      },
      getEditorDom: () => editor.view.dom,
      scrollToHighlight: (commentId: string) => {
        const dom = editor.view.dom;
        const el = dom.querySelector(`[data-comment-id="${commentId}"]`);
        if (el) {
          el.scrollIntoView({ behavior: "smooth", block: "center" });
          el.classList.add("active");
          setTimeout(() => el.classList.remove("active"), 2000);
        }
      },
    };
  }, [editor, editorRef]);

  // Handle click on comment highlights
  useEffect(() => {
    if (!editor || !onHighlightClick) return;
    const handleClick = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      const highlight = target.closest("[data-comment-id]");
      if (highlight) {
        const commentId = highlight.getAttribute("data-comment-id");
        if (commentId) onHighlightClick(commentId);
      }
    };
    const dom = editor.view.dom;
    dom.addEventListener("click", handleClick);
    return () => dom.removeEventListener("click", handleClick);
  }, [editor, onHighlightClick]);

  const debouncedSave = useCallback(
    (() => {
      let timeout: NodeJS.Timeout;
      return (content: string) => {
        setSaveStatus("idle");
        clearTimeout(timeout);
        timeout = setTimeout(async () => {
          if (onSave) {
            setSaveStatus("saving");
            try {
              onSave(content);
              setSaveStatus("saved");
            } catch {
              setSaveStatus("error");
            }
          }
        }, 500);
      };
    })(),
    [onSave],
  );

  const handleCommentClick = () => {
    if (!editor || !onAddComment) return;
    const { from, to } = editor.state.selection;
    if (from === to) return;
    const text = editor.state.doc.textBetween(from, to, " ");
    if (!text.trim()) return;
    savedSelectionRef.current = { from, to };
    editor.commands.setTextSelection(to);
    onAddComment(text);
  };

  const bubbleBtn = (
    active: boolean,
    onClick: () => void,
    children: React.ReactNode,
  ) => (
    <button
      onMouseDown={(e) => {
        e.preventDefault();
        onClick();
      }}
      className={`flex cursor-pointer items-center justify-center rounded px-1.5 py-1 transition-colors hover:bg-accent ${
        active ? "bg-accent text-foreground" : "text-muted-foreground"
      }`}
    >
      {children}
    </button>
  );

  return (
    <div>
      {/* Status bar */}
      <div className="flex items-center justify-between px-1 py-1">
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-muted-foreground">
            {saveStatus === "saved" && t("common.saved")}
            {saveStatus === "saving" && t("common.saving")}
            {saveStatus === "error" && "Save failed"}
          </span>
          {isCollaborative && (
            <span className={`flex items-center gap-1 text-[11px] ${
              connectionStatus === "connected" ? "text-emerald-400" : "text-muted-foreground"
            }`}>
              {connectionStatus === "connected" ? <Wifi className="size-3" /> : <WifiOff className="size-3" />}
              {connectionStatus === "connecting" && "Connecting..."}
            </span>
          )}
        </div>
        {readOnly && (
          <span className="text-[11px] text-muted-foreground">View only</span>
        )}
      </div>

      {/* Editor */}
      <div className="min-h-[400px] p-6">
        {editor && !readOnly && (
          <BubbleMenu
            editor={editor}
            className="flex items-center gap-0.5 rounded-lg border border-border bg-popover p-1 shadow-xl"
          >
            {bubbleBtn(
              editor.isActive("bold"),
              () => editor.chain().focus().toggleBold().run(),
              <Bold className="size-4" />,
            )}
            {bubbleBtn(
              editor.isActive("italic"),
              () => editor.chain().focus().toggleItalic().run(),
              <Italic className="size-4" />,
            )}
            {bubbleBtn(
              editor.isActive("heading", { level: 2 }),
              () => editor.chain().focus().toggleHeading({ level: 2 }).run(),
              <Heading2 className="size-4" />,
            )}
            {bubbleBtn(
              editor.isActive("heading", { level: 3 }),
              () => editor.chain().focus().toggleHeading({ level: 3 }).run(),
              <Heading3 className="size-4" />,
            )}
            {bubbleBtn(
              editor.isActive("bulletList"),
              () => editor.chain().focus().toggleBulletList().run(),
              <List className="size-4" />,
            )}
            {bubbleBtn(
              editor.isActive("codeBlock"),
              () => editor.chain().focus().toggleCodeBlock().run(),
              <Code className="size-4" />,
            )}

            {onAddComment && (
              <>
                <div className="mx-0.5 h-5 w-px bg-border" />
                {bubbleBtn(false, handleCommentClick, <MessageSquarePlus className="size-4" />)}
              </>
            )}
          </BubbleMenu>
        )}

        <EditorContent editor={editor} className="prose prose-invert max-w-none" />
      </div>
    </div>
  );
}
