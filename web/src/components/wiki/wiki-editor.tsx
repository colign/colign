"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useCreateBlockNote, SuggestionMenuController } from "@blocknote/react";
import { BlockNoteView } from "@blocknote/shadcn";
import {
  BlockNoteSchema,
  defaultInlineContentSpecs,
  defaultStyleSpecs,
  defaultBlockSpecs,
} from "@blocknote/core";
import type { PartialBlock } from "@blocknote/core";
import { HocuspocusProvider } from "@hocuspocus/provider";
import * as Y from "yjs";
import { wikiClient } from "@/lib/wiki";
import { getAccessToken, getTokenPayload } from "@/lib/auth";
import { PageLink } from "./extensions/page-link";
import "@blocknote/shadcn/style.css";

const wikiSchema = BlockNoteSchema.create({
  blockSpecs: defaultBlockSpecs,
  inlineContentSpecs: {
    ...defaultInlineContentSpecs,
    pageLink: PageLink,
  },
  styleSpecs: defaultStyleSpecs,
});

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

  const collab = useMemo(() => {
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
    return { doc, provider };
  }, [pageId, hocuspocusUrl]);

  useEffect(() => {
    return () => {
      collab.provider.destroy();
      collab.doc.destroy();
    };
  }, [collab]);

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
    schema: wikiSchema,
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

  // Cache page list for [[ suggestion menu
  const [wikiPages, setWikiPages] = useState<
    Array<{ id: string; title: string; icon: string }>
  >([]);
  const pagesFetchedRef = useRef(false);

  useEffect(() => {
    if (pagesFetchedRef.current) return;
    pagesFetchedRef.current = true;
    wikiClient
      .listWikiPages({ projectId })
      .then((res) => {
        setWikiPages(
          res.pages.map((p) => ({
            id: p.id,
            title: p.title || "Untitled",
            icon: p.icon || "",
          })),
        );
      })
      .catch(() => {});
  }, [projectId]);

  // Seed editor from initialContent when Yjs doc is empty (no yjs_state on server)
  useEffect(() => {
    if (seededRef.current || !initialContent || initialContent.length === 0) return;

    const fragment = doc.getXmlFragment("document-store");

    const trySeed = () => {
      if (seededRef.current) return;
      seededRef.current = true;
      // Seed when fragment is empty OR when the editor document has no meaningful content
      // (BlockNote may write a default empty paragraph before sync completes)
      const isDocEmpty = fragment.length === 0 || editor.document.every(
        (block) => {
          const content = block.content;
          if (!content || !Array.isArray(content)) return true;
          return content.length === 0 ||
            content.every((inline: { type?: string; text?: string }) =>
              inline.type !== "text" || !inline.text?.trim(),
            );
        },
      );
      if (isDocEmpty) {
        try {
          editor.replaceBlocks(editor.document, initialContent);
        } catch {
          // Editor may not be ready yet; ignore
        }
      }
    };

    // If provider already synced, seed immediately
    if (provider.isSynced) {
      trySeed();
      return;
    }

    // Wait for provider sync event
    const onSynced = () => trySeed();
    provider.on("synced", onSynced);

    // Fallback if server is unreachable
    const fallback = setTimeout(trySeed, 2000);

    return () => {
      provider.off("synced", onSynced);
      clearTimeout(fallback);
    };
  }, [doc, editor, initialContent, provider]);

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
      className="wiki-blocknote-editor"
    >
      <SuggestionMenuController
        triggerCharacter="[["
        getItems={async (query) => {
          return wikiPages
            .filter((p) => p.id !== pageId)
            .filter(
              (p) =>
                !query ||
                p.title.toLowerCase().includes(query.toLowerCase()),
            )
            .map((p) => ({
              title: `${p.icon ? p.icon + " " : ""}${p.title}`,
              onItemClick: () => {
                editor.insertInlineContent([
                  {
                    type: "pageLink",
                    props: { pageId: p.id, pageTitle: p.title },
                  },
                  " ",
                ]);
              },
            }));
        }}
      />
    </BlockNoteView>
  );
}
