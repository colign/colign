"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useCreateBlockNote } from "@blocknote/react";
import { BlockNoteView } from "@blocknote/shadcn";
import type { PartialBlock } from "@blocknote/core";
import { HocuspocusProvider } from "@hocuspocus/provider";
import * as Y from "yjs";
import { wikiClient } from "@/lib/wiki";
import { getAccessToken, getTokenPayload } from "@/lib/auth";
import "@blocknote/shadcn/style.css";

interface WikiEditorProps {
  projectId: bigint;
  pageId: string;
  initialContent?: PartialBlock[];
  onContentChange?: (json: string) => void;
}

const CURSOR_COLORS = [
  "#3b82f6", "#ef4444", "#10b981", "#f59e0b",
  "#8b5cf6", "#ec4899", "#06b6d4", "#f97316",
];

function normalizeCollaboratorName(value: unknown): string {
  return typeof value === "string" && value.trim() ? value : "Anonymous";
}

function getCollaborationProvider(provider: HocuspocusProvider) {
  return provider as HocuspocusProvider & {
    awareness?: NonNullable<HocuspocusProvider["awareness"]>;
  };
}

function getUserColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return CURSOR_COLORS[Math.abs(hash) % CURSOR_COLORS.length];
}

export function WikiEditor({ projectId, pageId, initialContent, onContentChange }: WikiEditorProps) {
  const hocuspocusUrl = process.env.NEXT_PUBLIC_HOCUSPOCUS_URL ?? "ws://localhost:1234";
  const [collab, setCollab] = useState<{ doc: Y.Doc; provider: HocuspocusProvider } | null>(null);

  useEffect(() => {
    const doc = new Y.Doc();
    const provider = new HocuspocusProvider({
      url: hocuspocusUrl,
      name: `wiki-${pageId}`,
      document: doc,
      token: getAccessToken() ?? undefined,
      onAuthenticationFailed: () => {
        console.warn("Hocuspocus auth failed for wiki page");
      },
    });

    setCollab({ doc, provider });

    return () => {
      provider.destroy();
      doc.destroy();
    };
  }, [pageId, hocuspocusUrl]);

  if (!collab) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="size-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <CollaborativeEditor
      projectId={projectId}
      pageId={pageId}
      doc={collab.doc}
      provider={collab.provider}
      initialContent={initialContent}
      onContentChange={onContentChange}
    />
  );
}

function CollaborativeEditor({
  projectId,
  pageId,
  doc,
  provider,
  initialContent,
  onContentChange,
}: {
  projectId: bigint;
  pageId: string;
  doc: Y.Doc;
  provider: HocuspocusProvider;
  initialContent?: PartialBlock[];
  onContentChange?: (json: string) => void;
}) {
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const seededRef = useRef(false);
  const payload = getTokenPayload();
  const userName = normalizeCollaboratorName(payload?.name);

  const editor = useCreateBlockNote({
    collaboration: {
      provider: getCollaborationProvider(provider),
      fragment: doc.getXmlFragment("document-store"),
      user: {
        name: userName,
        color: getUserColor(userName),
      },
    },
    uploadFile: async (file: File) => {
      const buffer = await file.arrayBuffer();
      const res = await wikiClient.uploadWikiImage({
        projectId,
        pageId,
        filename: file.name,
        contentType: file.type,
        data: new Uint8Array(buffer),
      });
      return res.url;
    },
  });

  // Seed editor from initialContent when Yjs doc is empty (no yjs_state on server)
  useEffect(() => {
    if (seededRef.current || !initialContent || initialContent.length === 0) return;

    const fragment = doc.getXmlFragment("document-store");
    // Wait a tick for the provider sync, then check if doc is still empty
    const timer = setTimeout(() => {
      if (fragment.length === 0 && !seededRef.current) {
        seededRef.current = true;
        try {
          editor.replaceBlocks(editor.document, initialContent);
        } catch {
          // Editor may not be ready yet; ignore
        }
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [doc, editor, initialContent]);

  const handleChange = useCallback(() => {
    if (!onContentChange) return;
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      const json = JSON.stringify(editor.document);
      onContentChange(json);
    }, 2000);
  }, [editor, onContentChange]);

  return (
    <BlockNoteView
      editor={editor}
      theme="dark"
      onChange={handleChange}
    />
  );
}
