"use client";

import { useEditor, EditorContent, type Editor as TiptapEditor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { useEffect, useRef, useCallback } from "react";
import { Bold, Italic, Heading2, Heading3, List, Code } from "lucide-react";

interface ReadmeEditorProps {
  initialContent: string;
  onSave: (html: string) => void;
  placeholder?: string;
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

export function ReadmeEditor({
  initialContent,
  onSave,
  placeholder = "Write your README...",
}: ReadmeEditorProps) {
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastSavedRef = useRef(initialContent);

  const editor = useEditor({
    extensions: [StarterKit, Placeholder.configure({ placeholder })],
    content: initialContent || undefined,
    immediatelyRender: false,
    onUpdate: ({ editor: e }) => {
      const html = e.getHTML();
      if (html === lastSavedRef.current) return;

      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        lastSavedRef.current = html;
        onSave(html);
      }, 1000);
    },
  });

  // Sync if initialContent changes externally
  useEffect(() => {
    if (editor && initialContent && !editor.isFocused && initialContent !== lastSavedRef.current) {
      editor.commands.setContent(initialContent);
      lastSavedRef.current = initialContent;
    }
  }, [editor, initialContent]);

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    };
  }, []);

  const bubbleBtn = useCallback(
    (active: boolean, onClick: () => void, children: React.ReactNode) => (
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
    ),
    [],
  );

  return (
    <div className="min-h-[120px]">
      {editor && (
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
        </BubbleMenu>
      )}

      <EditorContent editor={editor} className="prose prose-invert prose-sm max-w-none px-5 py-4" />
    </div>
  );
}
