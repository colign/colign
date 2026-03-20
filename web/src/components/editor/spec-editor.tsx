"use client";

import { useEditor, EditorContent } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Collaboration from "@tiptap/extension-collaboration";
import { HocuspocusProvider } from "@hocuspocus/provider";
import { CommentHighlight } from "./extensions/comment-highlight";
import { useEffect, useRef, useState } from "react";
import {
  Bold,
  Italic,
  Heading2,
  Heading3,
  List,
  Code,
  MessageSquarePlus,
} from "lucide-react";
import { getAccessToken } from "@/lib/auth";
import * as Y from "yjs";

interface SpecEditorProps {
  initialContent?: string;
  placeholder?: string;
  readOnly?: boolean;
  onAddComment?: (quotedText: string) => void;
  onHighlightClick?: (commentId: string) => void;
  editorRef?: React.MutableRefObject<{
    addHighlightAtSavedSelection: (commentId: string) => void;
    removeHighlight: (commentId: string) => void;
    scrollToHighlight: (commentId: string) => void;
    getEditorDom: () => HTMLElement | null;
  } | null>;
  documentId: string;
  userName?: string;
}

// Outer component: manages Y.js lifecycle
export function SpecEditor(props: SpecEditorProps) {
  const { documentId, userName = "Anonymous" } = props;
  const hocuspocusUrl = process.env.NEXT_PUBLIC_HOCUSPOCUS_URL ?? "ws://localhost:1234";

  const [collabReady, setCollabReady] = useState<{
    ydoc: Y.Doc;
    provider: HocuspocusProvider;
  } | null>(null);

  useEffect(() => {
    const ydoc = new Y.Doc();
    const provider = new HocuspocusProvider({
      url: hocuspocusUrl,
      name: documentId,
      document: ydoc,
      token: getAccessToken() ?? undefined,
      onAuthenticationFailed: () => {
        console.warn("Hocuspocus auth failed");
      },
    });

    const onSynced = () => {
      setCollabReady({ ydoc, provider });
    };
    provider.on("synced", onSynced);

    const timeout = setTimeout(() => {
      setCollabReady({ ydoc, provider });
    }, 3000);

    return () => {
      clearTimeout(timeout);
      provider.off("synced", onSynced);
      setCollabReady(null);
      setTimeout(() => {
        provider.destroy();
        ydoc.destroy();
      }, 100);
    };
  }, [documentId, hocuspocusUrl]);

  if (!collabReady) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <SpecEditorInner
      key={`collab-${documentId}`}
      {...props}
      collab={collabReady}
      userName={userName}
    />
  );
}

// Inner component: actual Tiptap editor
function SpecEditorInner({
  initialContent = "",
  placeholder = "Start writing...",
  readOnly = false,
  onAddComment,
  onHighlightClick,
  editorRef,
  collab,
}: SpecEditorProps & {
  collab: { ydoc: Y.Doc; provider: HocuspocusProvider };
}) {
  const savedSelectionRef = useRef<{ from: number; to: number } | null>(null);
  const initializedRef = useRef(false);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const extensions: any[] = [
    StarterKit.configure({ undoRedo: false } as any),
    Placeholder.configure({ placeholder }),
    CommentHighlight,
    Collaboration.configure({
      fragment: collab.ydoc.getXmlFragment("default"),
    }),
  ];

  const editor = useEditor({
    extensions,
    content: undefined,
    editable: !readOnly,
    immediatelyRender: false,
  });

  // Initialize content only for brand-new documents
  useEffect(() => {
    if (!editor || initializedRef.current) return;
    initializedRef.current = true;

    if (editor.isEmpty && initialContent) {
      const yMeta = collab.ydoc.getMap("meta");
      const legacyHtml = yMeta.get("initialHtml") as string | undefined;
      const contentToUse = legacyHtml || initialContent;
      editor.commands.setContent(contentToUse);
      if (legacyHtml) {
        yMeta.delete("initialHtml");
      }
    }
  }, [editor, initialContent, collab.ydoc]);

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
      },
      removeHighlight: (commentId: string) => {
        editor.chain().focus().unsetCommentHighlight(commentId).run();
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
