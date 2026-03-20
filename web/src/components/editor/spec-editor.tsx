"use client";

import { useEditor, EditorContent } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Collaboration from "@tiptap/extension-collaboration";
import CollaborationCursor from "@tiptap/extension-collaboration-cursor";
import { HocuspocusProvider } from "@hocuspocus/provider";
import { CommentHighlight } from "./extensions/comment-highlight";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Bold,
  Italic,
  Heading2,
  Heading3,
  List,
  Code,
  MessageSquarePlus,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { getAccessToken } from "@/lib/auth";
import * as Y from "yjs";

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

// Outer component: manages Y.js lifecycle, renders inner editor only when ready
export function SpecEditor(props: SpecEditorProps) {
  const { documentId, userName = "Anonymous" } = props;
  const isCollaborative = !!documentId;
  const hocuspocusUrl = process.env.NEXT_PUBLIC_HOCUSPOCUS_URL ?? "ws://localhost:1234";

  const [collabReady, setCollabReady] = useState<{
    ydoc: Y.Doc;
    provider: HocuspocusProvider;
  } | null>(null);
  const [collabFailed, setCollabFailed] = useState(false);

  useEffect(() => {
    if (!isCollaborative || !documentId) return;

    try {
      const ydoc = new Y.Doc();
      const provider = new HocuspocusProvider({
        url: hocuspocusUrl,
        name: documentId,
        document: ydoc,
        token: getAccessToken() ?? undefined,
        onAuthenticationFailed: () => {
          console.warn("Hocuspocus auth failed, falling back to standalone mode");
          setCollabFailed(true);
        },
      });

      // Wait for provider to sync before mounting editor
      const onSynced = () => {
        setCollabReady({ ydoc, provider });
      };
      provider.on("synced", onSynced);

      // Timeout fallback — mount after 3s even if not synced
      const timeout = setTimeout(() => {
        setCollabReady({ ydoc, provider });
      }, 3000);

      return () => {
        clearTimeout(timeout);
        provider.off("synced", onSynced);
        setCollabReady(null);
        // Delay destruction to avoid React Strict Mode stale reference crash
        setTimeout(() => {
          provider.destroy();
          ydoc.destroy();
        }, 100);
      };
    } catch (err) {
      console.warn("Failed to create Hocuspocus provider:", err);
      setCollabFailed(true);
    }
  }, [isCollaborative, documentId, hocuspocusUrl]);

  if (collabFailed) {
    return (
      <SpecEditorInner
        key="standalone-fallback"
        {...props}
        collab={null}
        userName={userName}
      />
    );
  }

  if (isCollaborative && !collabReady) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <SpecEditorInner
      key={isCollaborative ? `collab-${documentId}` : "standalone"}
      {...props}
      collab={collabReady}
      userName={userName}
    />
  );
}

// Inner component: actual Tiptap editor, receives ready ydoc/provider
function SpecEditorInner({
  initialContent = "",
  placeholder = "Start writing...",
  onSave,
  readOnly = false,
  onAddComment,
  onHighlightClick,
  editorRef,
  collab,
  userName = "Anonymous",
}: SpecEditorProps & {
  collab: { ydoc: Y.Doc; provider: HocuspocusProvider } | null;
}) {
  const { t } = useI18n();
  const [saveStatus, setSaveStatus] = useState<"saved" | "saving" | "error" | "idle">("idle");
  const savedSelectionRef = useRef<{ from: number; to: number } | null>(null);

  const isCollaborative = collab != null;

  // Build extensions — collab.ydoc/provider are guaranteed ready here
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const extensions: any[] = [
    isCollaborative
      ? StarterKit.configure({ undoRedo: false } as any)
      : StarterKit,
    Placeholder.configure({ placeholder }),
    CommentHighlight,
  ];

  if (collab) {
    extensions.push(Collaboration.configure({
      fragment: collab.ydoc.getXmlFragment("default"),
    }));
    // TODO: CollaborationCursor disabled due to y-prosemirror cursor-plugin
    // crash (ystate undefined in createDecorations). This is a known compat
    // issue between @tiptap/extension-collaboration-cursor v3 and y-prosemirror.
    // Cursor display will be added via custom awareness subscription.
  }

  const editor = useEditor({
    extensions,
    content: isCollaborative ? undefined : initialContent,
    editable: !readOnly,
    immediatelyRender: false,
    onUpdate: ({ editor }) => {
      debouncedSave(editor.getHTML());
    },
  });

  // Set initial content
  useEffect(() => {
    if (!editor || !initialContent) return;

    if (isCollaborative && collab) {
      // Wait for sync, then load content if Y.js doc is empty
      const onSynced = () => {
        if (editor.isEmpty) {
          editor.commands.setContent(initialContent);
        }
      };
      collab.provider.on("synced", onSynced);
      // Also check immediately in case already synced
      if (editor.isEmpty) {
        editor.commands.setContent(initialContent);
      }
      return () => {
        collab.provider.off("synced", onSynced);
      };
    } else if (!isCollaborative) {
      editor.commands.setContent(initialContent);
    }
  }, [isCollaborative, initialContent, editor, collab]);

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
      <div className="flex h-6 items-center justify-between px-1">
        <span className="text-[11px] text-muted-foreground">
          {saveStatus === "saved" && t("common.saved")}
          {saveStatus === "saving" && t("common.saving")}
          {saveStatus === "error" && "Save failed"}
          {saveStatus === "idle" && "\u00A0"}
        </span>
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
