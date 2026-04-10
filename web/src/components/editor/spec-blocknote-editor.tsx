"use client";

import { useCallback, useEffect, useRef, useState, forwardRef, useImperativeHandle } from "react";
import {
  useCreateBlockNote,
  FormattingToolbar,
  FormattingToolbarController,
} from "@blocknote/react";
import { BlockNoteView } from "@blocknote/shadcn";
import { BlockNoteSchema, defaultStyleSpecs } from "@blocknote/core";
import type { PartialBlock } from "@blocknote/core";
import { HocuspocusProvider } from "@hocuspocus/provider";
import * as Y from "yjs";
import { CommentHighlight } from "./extensions/comment-highlight";
import { getAccessToken, getTokenPayload } from "@/lib/auth";
import { MessageSquarePlus } from "lucide-react";
import "@blocknote/shadcn/style.css";

const CURSOR_COLORS = [
  "#3b82f6", "#ef4444", "#10b981", "#f59e0b",
  "#8b5cf6", "#ec4899", "#06b6d4", "#f97316",
];

function getUserColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return CURSOR_COLORS[Math.abs(hash) % CURSOR_COLORS.length];
}

function normalizeCollaboratorName(value: unknown): string {
  return typeof value === "string" && value.trim() ? value : "Anonymous";
}

const schema = BlockNoteSchema.create({
  styleSpecs: {
    ...defaultStyleSpecs,
    commentHighlight: CommentHighlight,
  },
});

export interface SpecBlockNoteEditorHandle {
  addHighlightAtSavedSelection: (commentId: string) => void;
  removeHighlight: (commentId: string) => void;
  scrollToHighlight: (commentId: string) => void;
  getEditorDom: () => HTMLElement | null;
}

interface SpecBlockNoteEditorProps {
  initialContent?: string;
  readOnly?: boolean;
  onAddComment?: (quotedText: string, rect: { top: number; left: number; width: number }) => void;
  onHighlightClick?: (commentId: string) => void;
  documentId: string;
}

// Outer component: manages Y.js lifecycle
export const SpecBlockNoteEditor = forwardRef<SpecBlockNoteEditorHandle, SpecBlockNoteEditorProps>(
  function SpecBlockNoteEditor(props, ref) {
    const { documentId } = props;
    const hocuspocusUrl = process.env.NEXT_PUBLIC_HOCUSPOCUS_URL ?? "ws://localhost:1234";

    const [collab, setCollab] = useState<{
      doc: Y.Doc;
      provider: HocuspocusProvider;
    } | null>(null);

    useEffect(() => {
      const doc = new Y.Doc();
      const provider = new HocuspocusProvider({
        url: hocuspocusUrl,
        name: documentId,
        document: doc,
        token: getAccessToken() ?? undefined,
        onAuthenticationFailed: () => {
          console.warn("Hocuspocus auth failed for spec editor");
        },
      });

      setCollab({ doc, provider });

      return () => {
        provider.destroy();
        doc.destroy();
      };
    }, [documentId, hocuspocusUrl]);

    if (!collab) {
      return (
        <div className="flex items-center justify-center py-20">
          <div className="size-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      );
    }

    return (
      <CollaborativeSpecEditor
        key={`collab-spec-${documentId}`}
        ref={ref}
        doc={collab.doc}
        provider={collab.provider}
        {...props}
      />
    );
  },
);

// Inner component: BlockNote editor with collaboration and comment highlights
const CollaborativeSpecEditor = forwardRef<
  SpecBlockNoteEditorHandle,
  SpecBlockNoteEditorProps & { doc: Y.Doc; provider: HocuspocusProvider }
>(function CollaborativeSpecEditor(
  { initialContent, readOnly = false, onAddComment, onHighlightClick, doc, provider },
  ref,
) {
  const seededRef = useRef(false);
  const savedSelectionRef = useRef<{ from: number; to: number } | null>(null);
  const payload = getTokenPayload();
  const userName = normalizeCollaboratorName(payload?.name);

  const editor = useCreateBlockNote({
    schema,
    collaboration: {
      provider: provider as HocuspocusProvider & {
        awareness?: NonNullable<HocuspocusProvider["awareness"]>;
      },
      fragment: doc.getXmlFragment("default"),
      user: {
        name: userName,
        color: getUserColor(userName),
      },
    },
  });

  // Seed editor from initialContent when Yjs doc is empty
  useEffect(() => {
    if (seededRef.current || !initialContent) return;

    const fragment = doc.getXmlFragment("default");

    const trySeed = () => {
      if (seededRef.current) return;
      seededRef.current = true;
      if (fragment.length === 0 && initialContent.trim()) {
        try {
          const trimmed = initialContent.trim();
          if (trimmed.startsWith("{") && trimmed.includes('"type"')) {
            // ProseMirror JSON — feed directly to Tiptap's setContent
            const parsed = JSON.parse(trimmed);
            editor._tiptapEditor.commands.setContent(parsed);
          } else if (trimmed.startsWith("<")) {
            const blocks = editor.tryParseHTMLToBlocks(initialContent);
            if (blocks.length > 0) {
              editor.replaceBlocks(editor.document, blocks);
            }
          } else {
            const blocks = editor.tryParseMarkdownToBlocks(initialContent);
            if (blocks.length > 0) {
              editor.replaceBlocks(editor.document, blocks);
            }
          }
        } catch {
          // Editor may not be ready yet
        }
      }
    };

    if (provider.isSynced) {
      trySeed();
      return;
    }

    const onSynced = () => trySeed();
    provider.on("synced", onSynced);
    const fallback = setTimeout(trySeed, 2000);

    return () => {
      provider.off("synced", onSynced);
      clearTimeout(fallback);
    };
  }, [doc, editor, initialContent, provider]);

  // Expose editor methods via ref
  useImperativeHandle(
    ref,
    () => ({
      addHighlightAtSavedSelection: (commentId: string) => {
        const sel = savedSelectionRef.current;
        if (!sel) return;
        const tiptap = editor._tiptapEditor;
        tiptap.chain().focus().setTextSelection(sel).setMark("commentHighlight", { stringValue: commentId }).run();
        savedSelectionRef.current = null;
      },
      removeHighlight: (commentId: string) => {
        const tiptap = editor._tiptapEditor;
        const { tr, doc } = tiptap.state;
        doc.descendants((node, pos) => {
          node.marks.forEach((mark) => {
            if (mark.type.name === "commentHighlight" && mark.attrs.stringValue === commentId) {
              tr.removeMark(pos, pos + node.nodeSize, mark);
            }
          });
        });
        tiptap.view.dispatch(tr);
      },
      scrollToHighlight: (commentId: string) => {
        const dom = editor._tiptapEditor.view.dom;
        const el = dom.querySelector(`[data-comment-id="${commentId}"]`);
        if (el) {
          el.scrollIntoView({ behavior: "smooth", block: "center" });
          el.classList.add("active");
          setTimeout(() => el.classList.remove("active"), 2000);
        }
      },
      getEditorDom: () => editor._tiptapEditor.view.dom ?? null,
    }),
    [editor],
  );

  // Handle click on comment highlights
  useEffect(() => {
    if (!onHighlightClick) return;
    const dom = editor._tiptapEditor.view.dom;
    const handleClick = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      const highlight = target.closest("[data-comment-id]");
      if (highlight) {
        const commentId = highlight.getAttribute("data-comment-id");
        if (commentId) onHighlightClick(commentId);
      }
    };
    dom.addEventListener("click", handleClick);
    return () => dom.removeEventListener("click", handleClick);
  }, [editor, onHighlightClick]);

  const handleCommentClick = useCallback(() => {
    if (!onAddComment) return;
    const tiptap = editor._tiptapEditor;
    const { from, to } = tiptap.state.selection;
    if (from === to) return;
    const text = tiptap.state.doc.textBetween(from, to, " ");
    if (!text.trim()) return;
    savedSelectionRef.current = { from, to };

    const coords = tiptap.view.coordsAtPos(to);
    const editorDom = tiptap.view.dom.closest("[data-editor-wrapper]") || tiptap.view.dom.parentElement;
    const editorRect = editorDom?.getBoundingClientRect() || { top: 0, left: 0, width: 600 };
    const rect = {
      top: coords.bottom - editorRect.top,
      left: 0,
      width: editorRect.width,
    };

    tiptap.commands.setTextSelection(to);
    onAddComment(text, rect);
  }, [editor, onAddComment]);

  return (
    <div data-editor-wrapper className="relative">
      <div className="min-h-[400px] p-6">
        <BlockNoteView
          editor={editor}
          editable={!readOnly}
          theme="dark"
          className="spec-blocknote-editor"
          formattingToolbar={false}
        >
          <FormattingToolbarWithComment
            onAddComment={onAddComment ? handleCommentClick : undefined}
          />
        </BlockNoteView>
      </div>
    </div>
  );
});

function FormattingToolbarWithComment({
  onAddComment,
}: {
  onAddComment?: () => void;
}) {
  return (
    <FormattingToolbarController
      formattingToolbar={() => (
        <FormattingToolbar>
          {onAddComment && (
            <button
              onMouseDown={(e: React.MouseEvent) => {
                e.preventDefault();
                onAddComment();
              }}
              className="bn-button flex cursor-pointer items-center justify-center rounded px-1.5 py-1 text-muted-foreground transition-colors hover:bg-accent"
              title="Add comment"
            >
              <MessageSquarePlus className="size-4" />
            </button>
          )}
        </FormattingToolbar>
      )}
    />
  );
}
