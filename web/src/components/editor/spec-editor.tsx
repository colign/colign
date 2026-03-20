"use client";

import { useEditor, EditorContent } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { CommentHighlight } from "./extensions/comment-highlight";
import { useCallback, useEffect, useRef } from "react";
import {
  Bold,
  Italic,
  Heading2,
  Heading3,
  List,
  Code,
  MessageSquarePlus,
} from "lucide-react";
import { useState } from "react";
import { useI18n } from "@/lib/i18n";

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
  } | null>;
}

export function SpecEditor({
  initialContent = "",
  placeholder = "Start writing...",
  onSave,
  readOnly = false,
  onAddComment,
  onHighlightClick,
  editorRef,
}: SpecEditorProps) {
  const { t } = useI18n();
  const [saveStatus, setSaveStatus] = useState<"saved" | "saving" | "error" | "idle">("idle");
  const savedSelectionRef = useRef<{ from: number; to: number } | null>(null);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Placeholder.configure({ placeholder }),
      CommentHighlight,
    ],
    content: initialContent,
    editable: !readOnly,
    immediatelyRender: false,
    onUpdate: ({ editor }) => {
      debouncedSave(editor.getHTML());
    },
  });

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
        // Trigger save so the mark is persisted in HTML
        if (onSave) {
          debouncedSave(editor.getHTML());
        }
      },
      removeHighlight: (commentId: string) => {
        editor.chain().focus().unsetCommentHighlight(commentId).run();
        if (onSave) {
          debouncedSave(editor.getHTML());
        }
      },
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

  useEffect(() => {
    if (editor && initialContent) {
      editor.commands.setContent(initialContent);
    }
  }, [initialContent, editor]);

  const handleCommentClick = () => {
    if (!editor || !onAddComment) return;
    const { from, to } = editor.state.selection;
    if (from === to) return;
    const text = editor.state.doc.textBetween(from, to, " ");
    if (!text.trim()) return;
    savedSelectionRef.current = { from, to };
    // Collapse selection to hide BubbleMenu
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
        <span className="text-[11px] text-muted-foreground">
          {saveStatus === "saved" && t("common.saved")}
          {saveStatus === "saving" && t("common.saving")}
          {saveStatus === "error" && "Save failed"}
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

            {/* Separator + Comment */}
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
