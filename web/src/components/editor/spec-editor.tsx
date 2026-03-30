"use client";

import { useEditor, EditorContent, type Editor as TiptapEditor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Collaboration from "@tiptap/extension-collaboration";
import { Table, TableCell, TableHeader, TableRow } from "@tiptap/extension-table";
import type { AnyExtension, JSONContent } from "@tiptap/core";
import { HocuspocusProvider } from "@hocuspocus/provider";
import { CommentHighlight } from "./extensions/comment-highlight";
import { useEffect, useRef, useState } from "react";
import { Bold, Italic, Heading2, Heading3, List, Code, MessageSquarePlus } from "lucide-react";
import { getAccessToken } from "@/lib/auth";
import { marked } from "marked";
import DOMPurify from "isomorphic-dompurify";
import * as Y from "yjs";

interface SpecEditorProps {
  initialContent?: string;
  placeholder?: string;
  readOnly?: boolean;
  onAddComment?: (quotedText: string, rect: { top: number; left: number; width: number }) => void;
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

function normalizeInitialContent(content: string | undefined): string | JSONContent {
  if (!content) return "";
  const trimmed = content.trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed) as JSONContent;
      if (parsed?.type === "doc") {
        return parsed;
      }
    } catch {
      // fall through
    }
  }
  if (trimmed.startsWith("<")) return DOMPurify.sanitize(content);
  return DOMPurify.sanitize(marked.parse(content, { async: false }) as string);
}

function toggleSmartCodeBlock(editor: TiptapEditor) {
  if (editor.isActive("codeBlock")) {
    editor.chain().focus().toggleCodeBlock().run();
    return;
  }

  const { from, to, empty } = editor.state.selection;
  if (empty) {
    editor.chain().focus().toggleCodeBlock().run();
    return;
  }

  let blockCount = 0;
  editor.state.doc.nodesBetween(from, to, (node) => {
    if (node.isBlock) {
      blockCount += 1;
    }
  });

  if (blockCount <= 1) {
    editor.chain().focus().toggleCodeBlock().run();
    return;
  }

  const selectedText = editor.state.doc.textBetween(from, to, "\n");
  editor
    .chain()
    .focus()
    .insertContentAt(
      { from, to },
      {
        type: "codeBlock",
        content: selectedText ? [{ type: "text", text: selectedText }] : [],
      },
    )
    .run();
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

  const extensions: AnyExtension[] = [
    StarterKit.configure({ undoRedo: false }),
    Placeholder.configure({ placeholder }),
    Table.configure({
      resizable: false,
    }),
    TableRow,
    TableHeader,
    TableCell,
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

    const yMeta = collab.ydoc.getMap("meta");
    const collabInitialContent = normalizeInitialContent(yMeta.get("initialContent") as string | undefined);
    const propInitialContent = normalizeInitialContent(initialContent);

    // Check if Y.js fragment has real structured content (headings, lists, etc.)
    const fragment = collab.ydoc.getXmlFragment("default");
    const hasStructure = fragment.toArray().some((node) => {
      const name = node instanceof Y.XmlElement ? node.nodeName : null;
      return name && name !== "paragraph";
    });

    // Initialize if editor is empty OR Y.js has no structured content
    if ((editor.isEmpty || !hasStructure) && (collabInitialContent || propInitialContent)) {
      const contentToUse = collabInitialContent || propInitialContent;
      editor.commands.setContent(contentToUse);
      if (collabInitialContent) {
        yMeta.delete("initialContent");
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
        editor.chain().focus().setTextSelection(sel).setCommentHighlight({ commentId }).run();
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

    // Get selection coordinates relative to editor container
    const coords = editor.view.coordsAtPos(to);
    const editorDom =
      editor.view.dom.closest("[data-editor-wrapper]") || editor.view.dom.parentElement;
    const editorRect = editorDom?.getBoundingClientRect() || { top: 0, left: 0, width: 600 };
    const rect = {
      top: coords.bottom - editorRect.top,
      left: 0,
      width: editorRect.width,
    };

    editor.commands.setTextSelection(to);
    onAddComment(text, rect);
  };

  const bubbleBtn = (active: boolean, onClick: () => void, children: React.ReactNode) => (
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
    <div data-editor-wrapper className="relative">
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
              () => toggleSmartCodeBlock(editor),
              <Code className="size-4" />,
            )}

            {onAddComment && (
              <>
                <div className="mx-0.5 h-5 w-px bg-border" />
                <button
                  onMouseDown={(e) => {
                    e.preventDefault();
                    handleCommentClick();
                  }}
                  className="flex cursor-pointer items-center justify-center rounded px-1.5 py-1 text-muted-foreground transition-colors hover:bg-accent"
                >
                  <MessageSquarePlus className="size-4" />
                </button>
              </>
            )}
          </BubbleMenu>
        )}

        <EditorContent editor={editor} className="prose prose-invert max-w-none" />
      </div>
    </div>
  );
}
