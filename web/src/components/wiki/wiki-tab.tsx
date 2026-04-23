"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import dynamic from "next/dynamic";
import { wikiClient } from "@/lib/wiki";
import { showError, showSuccess } from "@/lib/toast";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import {
  FileText,
  Plus,
  ChevronRight,
  ChevronDown,
  MoreHorizontal,
  Trash2,
  FilePlus,
  GripVertical,
  Link2,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
  arrayMove,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
const WikiEditor = dynamic(
  () => import("@/components/wiki/wiki-editor").then((m) => m.WikiEditor),
  { ssr: false },
);
import type { WikiPage } from "@/gen/proto/wiki/v1/wiki_pb";

interface TreeNode {
  page: WikiPage;
  children: TreeNode[];
}

function normalizeWikiText(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function buildTree(pages: WikiPage[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  const roots: TreeNode[] = [];

  for (const page of pages) {
    map.set(page.id, { page, children: [] });
  }

  for (const page of pages) {
    const node = map.get(page.id)!;
    if (page.parentId && map.has(page.parentId)) {
      map.get(page.parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => a.page.sortOrder - b.page.sortOrder);
    for (const node of nodes) {
      sortNodes(node.children);
    }
  };
  sortNodes(roots);

  return roots;
}

export function WikiTab({ projectId }: { projectId: bigint }) {
  const { t } = useI18n();
  const searchParams = useSearchParams();
  const [pages, setPages] = useState<WikiPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPageId, setSelectedPageId] = useState<string | null>(null);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [deleteTarget, setDeleteTarget] = useState<WikiPage | null>(null);
  const lastPageParamRef = useRef<string | null>(null);
  const pageCacheRef = useRef<Map<string, WikiPage>>(new Map());

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor),
  );

  // Preload WikiEditor chunk as soon as wiki tab mounts
  // so it's ready when user clicks a page
  useEffect(() => {
    import("@/components/wiki/wiki-editor");
  }, []);

  const loadPages = useCallback(async () => {
    try {
      const res = await wikiClient.listWikiPages({ projectId });
      setPages(res.pages);
    } catch (err) {
      showError(t("toast.loadFailed"), err);
    } finally {
      setLoading(false);
    }
  }, [projectId, t]);

  useEffect(() => {
    loadPages();
  }, [loadPages]);

  // Auto-select wiki page from URL ?page= parameter (e.g. from document links)
  useEffect(() => {
    const pageParam = searchParams.get("page");
    if (!pageParam || pages.length === 0) return;
    if (pageParam === lastPageParamRef.current) return;
    lastPageParamRef.current = pageParam;
    // Match by title (case-insensitive, strip extension)
    const normalized = pageParam.replace(/\.md$/i, "").toLowerCase();
    const match = pages.find((p) => {
      const title = normalizeWikiText(p.title).toLowerCase();
      return title === normalized || title === pageParam.toLowerCase();
    });
    if (match) {
      setSelectedPageId(match.id);
      if (match.parentId) {
        setExpandedIds((prev) => new Set([...prev, match.parentId]));
      }
    }
  }, [searchParams, pages]);

  // Navigate to a page when clicking a [[Page Link]] in the editor
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (!detail?.pageId) return;
      setSelectedPageId(detail.pageId);
      const target = pages.find((p) => p.id === detail.pageId);
      if (target?.parentId) {
        setExpandedIds((prev) => new Set([...prev, target.parentId]));
      }
    };
    window.addEventListener("wiki-navigate", handler);
    return () => window.removeEventListener("wiki-navigate", handler);
  }, [pages]);


  const handleCreatePage = async (parentId?: string) => {
    try {
      const res = await wikiClient.createWikiPage({
        projectId,
        parentId: parentId ?? "",
        title: "",
      });
      await loadPages();
      setSelectedPageId(res.page!.id);
      if (parentId) {
        setExpandedIds((prev) => new Set([...prev, parentId]));
      }
    } catch (err) {
      showError(t("toast.createFailed"), err);
    }
  };

  const handleDeletePage = async () => {
    if (!deleteTarget) return;
    try {
      await wikiClient.deleteWikiPage({ projectId, pageId: deleteTarget.id });
      showSuccess(t("project.wikiDeleted"));
      if (selectedPageId === deleteTarget.id) {
        setSelectedPageId(null);
      }
      setDeleteTarget(null);
      await loadPages();
    } catch (err) {
      showError(t("toast.deleteFailed"), err);
    }
  };

  const toggleExpanded = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeId = active.id as string;
    const overId = over.id as string;

    // Find siblings of the dragged item
    const activePage = pages.find((p) => p.id === activeId);
    const overPage = pages.find((p) => p.id === overId);
    if (!activePage || !overPage) return;

    // Only reorder within the same parent
    if (activePage.parentId !== overPage.parentId) return;

    const siblings = pages
      .filter((p) => p.parentId === activePage.parentId)
      .sort((a, b) => a.sortOrder - b.sortOrder);

    const oldIndex = siblings.findIndex((s) => s.id === activeId);
    const newIndex = siblings.findIndex((s) => s.id === overId);
    if (oldIndex === -1 || newIndex === -1) return;

    const reordered = arrayMove(siblings, oldIndex, newIndex);

    // Optimistic update
    setPages((prev) => {
      const updated = [...prev];
      for (let i = 0; i < reordered.length; i++) {
        const idx = updated.findIndex((p) => p.id === reordered[i].id);
        if (idx !== -1) {
          updated[idx] = { ...updated[idx], sortOrder: i } as WikiPage;
        }
      }
      return updated;
    });

    // Server update
    try {
      await wikiClient.reorderWikiPages({
        projectId,
        pageId: activeId,
        parentId: activePage.parentId ?? "",
        sortOrder: newIndex,
      });
    } catch (err) {
      showError(t("toast.saveFailed"), err);
      await loadPages();
    }
  };

  const handlePageLoaded = useCallback((page: WikiPage) => {
    pageCacheRef.current.set(page.id, page);
  }, []);

  const tree = buildTree(pages);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="size-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <>
      <div className="flex gap-0">
        {/* Sidebar */}
        <div className="w-56 shrink-0 border-r border-border pr-4">
          <div className="sticky top-14 max-h-[calc(100vh-4rem)] overflow-y-auto overflow-x-hidden scrollbar-subtle">
            <Button
              variant="ghost"
              className="mb-3 w-full justify-start gap-1.5 text-sm text-muted-foreground hover:text-foreground"
              onClick={() => handleCreatePage()}
            >
              <Plus className="size-3.5" />
              {t("project.wikiNewPage")}
            </Button>

            {tree.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-lg border border-border px-4 py-12 text-center">
                <FileText className="mb-3 size-8 text-muted-foreground" />
                <p className="mb-1 text-sm font-medium text-foreground/80">
                  {t("project.wikiEmptyTitle")}
                </p>
                <p className="mb-4 text-xs text-muted-foreground">
                  {t("project.wikiEmptyDesc")}
                </p>
                <Button size="sm" onClick={() => handleCreatePage()}>
                  <Plus className="mr-1.5 size-3.5" />
                  {t("project.wikiCreateFirst")}
                </Button>
              </div>
            ) : (
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragEnd={handleDragEnd}
              >
                <SortableTreeLevel
                  nodes={tree}
                  depth={0}
                  selectedId={selectedPageId}
                  expandedIds={expandedIds}
                  onSelect={setSelectedPageId}
                  onToggle={toggleExpanded}
                  onCreateChild={(parentId) => handleCreatePage(parentId)}
                  onDelete={setDeleteTarget}
                />
              </DndContext>
            )}
          </div>
        </div>

        {/* Content Area */}
        <div className="min-w-0 flex-1 pl-4">
          {selectedPageId ? (
            <WikiPageContent
              projectId={projectId}
              pageId={selectedPageId}
              pageCache={pageCacheRef}
              onPageLoaded={handlePageLoaded}
              onTitleChange={(pageId, title) => {
                setPages((prev) =>
                  prev.map((p) => (p.id === pageId ? { ...p, title } as WikiPage : p)),
                );
              }}
            />
          ) : (
            <div className="flex items-center justify-center py-20">
              <div className="text-center">
                <FileText className="mx-auto mb-3 size-10 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">
                  {t("project.wikiEmptyDesc")}
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-destructive">{t("project.wikiDeleteTitle")}</DialogTitle>
            <DialogDescription>{t("project.wikiDeleteDesc")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeletePage}
            >
              {t("project.wikiDeleteConfirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function SortableTreeLevel({
  nodes,
  depth,
  selectedId,
  expandedIds,
  onSelect,
  onToggle,
  onCreateChild,
  onDelete,
}: {
  nodes: TreeNode[];
  depth: number;
  selectedId: string | null;
  expandedIds: Set<string>;
  onSelect: (id: string) => void;
  onToggle: (id: string) => void;
  onCreateChild: (parentId: string) => void;
  onDelete: (page: WikiPage) => void;
}) {
  return (
    <SortableContext
      items={nodes.map((n) => n.page.id)}
      strategy={verticalListSortingStrategy}
    >
      <div className="space-y-0.5">
        {nodes.map((node) => (
          <SortableTreeItem
            key={node.page.id}
            node={node}
            depth={depth}
            selectedId={selectedId}
            expandedIds={expandedIds}
            onSelect={onSelect}
            onToggle={onToggle}
            onCreateChild={onCreateChild}
            onDelete={onDelete}
          />
        ))}
      </div>
    </SortableContext>
  );
}

function SortableTreeItem({
  node,
  depth,
  selectedId,
  expandedIds,
  onSelect,
  onToggle,
  onCreateChild,
  onDelete,
}: {
  node: TreeNode;
  depth: number;
  selectedId: string | null;
  expandedIds: Set<string>;
  onSelect: (id: string) => void;
  onToggle: (id: string) => void;
  onCreateChild: (parentId: string) => void;
  onDelete: (page: WikiPage) => void;
}) {
  const { t } = useI18n();
  const hasChildren = node.children.length > 0;
  const isExpanded = expandedIds.has(node.page.id);
  const isSelected = selectedId === node.page.id;
  const maxDepth = 2;
  const pageIcon = normalizeWikiText(node.page.icon);
  const pageTitle = normalizeWikiText(node.page.title);

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: node.page.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div ref={setNodeRef} style={style}>
      <div
        className={`group flex items-center gap-1 rounded-md px-2 py-1.5 text-sm cursor-pointer transition-colors ${
          isSelected
            ? "bg-accent text-foreground"
            : "text-foreground/80 hover:bg-muted/50"
        }`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={() => onSelect(node.page.id)}
      >
        <button
          className="flex size-4 shrink-0 cursor-grab items-center justify-center rounded opacity-0 group-hover:opacity-60 active:cursor-grabbing"
          {...attributes}
          {...listeners}
          onClick={(e) => e.stopPropagation()}
        >
          <GripVertical className="size-3" />
        </button>

        {hasChildren ? (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggle(node.page.id);
            }}
            className="flex size-4 shrink-0 items-center justify-center rounded hover:bg-muted"
          >
            {isExpanded ? (
              <ChevronDown className="size-3" />
            ) : (
              <ChevronRight className="size-3" />
            )}
          </button>
        ) : (
          <span className="size-4 shrink-0" />
        )}

        {pageIcon ? (
          <span className="shrink-0 text-sm">{pageIcon}</span>
        ) : (
          <FileText className="size-3.5 shrink-0 text-muted-foreground" />
        )}

        <span className="truncate">
          {pageTitle || t("project.wikiUntitled")}
        </span>

        <TreeItemMenu
          depth={depth}
          maxDepth={maxDepth}
          onCreateChild={() => onCreateChild(node.page.id)}
          onDelete={() => onDelete(node.page)}
        />
      </div>

      {isExpanded && node.children.length > 0 && (
        <SortableTreeLevel
          nodes={node.children}
          depth={depth + 1}
          selectedId={selectedId}
          expandedIds={expandedIds}
          onSelect={onSelect}
          onToggle={onToggle}
          onCreateChild={onCreateChild}
          onDelete={onDelete}
        />
      )}
    </div>
  );
}

function TreeItemMenu({
  depth,
  maxDepth,
  onCreateChild,
  onDelete,
}: {
  depth: number;
  maxDepth: number;
  onCreateChild: () => void;
  onDelete: () => void;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open]);

  return (
    <div className="relative ml-auto shrink-0" ref={menuRef}>
      <button
        onClick={(e) => {
          e.stopPropagation();
          setOpen((v) => !v);
        }}
        className="flex size-5 items-center justify-center rounded opacity-0 hover:bg-muted group-hover:opacity-100"
      >
        <MoreHorizontal className="size-3" />
      </button>
      {open && (
        <div className="absolute right-0 top-full z-50 mt-1 w-40 rounded-md border border-border bg-popover py-1 shadow-md">
          {depth < maxDepth && (
            <button
              className="flex w-full items-center gap-2 px-3 py-1.5 text-sm hover:bg-muted"
              onClick={(e) => {
                e.stopPropagation();
                setOpen(false);
                onCreateChild();
              }}
            >
              <FilePlus className="size-3.5" />
              {t("project.wikiAddSubpage")}
            </button>
          )}
          <button
            className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-destructive hover:bg-muted"
            onClick={(e) => {
              e.stopPropagation();
              setOpen(false);
              onDelete();
            }}
          >
            <Trash2 className="size-3.5" />
            {t("project.wikiDeleteConfirm")}
          </button>
        </div>
      )}
    </div>
  );
}

function WikiPageContent({
  projectId,
  pageId,
  pageCache,
  onPageLoaded,
  onTitleChange,
}: {
  projectId: bigint;
  pageId: string;
  pageCache: React.RefObject<Map<string, WikiPage>>;
  onPageLoaded: (page: WikiPage) => void;
  onTitleChange: (pageId: string, title: string) => void;
}) {
  const { t } = useI18n();
  const [prevPageId, setPrevPageId] = useState(pageId);
  const [fetchedPage, setFetchedPage] = useState<WikiPage | null>(null);
  const [title, setTitle] = useState("");
  const [editorReady, setEditorReady] = useState(false);

  // Handle page switch during render — no unmount, instant state reset
  const cachedPage = pageCache.current?.get(pageId);
  if (pageId !== prevPageId) {
    setPrevPageId(pageId);
    setFetchedPage(null);
    setEditorReady(false);
    setTitle(cachedPage ? normalizeWikiText(cachedPage.title) : "");
  }

  // Derive current page: fresh fetch > cache > null
  const page = (fetchedPage?.id === pageId ? fetchedPage : null) ?? cachedPage ?? null;
  const titleLoading = !page;

  // Fetch fresh data on page change
  useEffect(() => {
    let cancelled = false;
    wikiClient
      .getWikiPage({ projectId, pageId })
      .then((res) => {
        if (!cancelled && res.page) {
          setFetchedPage(res.page);
          setTitle(normalizeWikiText(res.page.title));
          onPageLoaded(res.page);
        }
      })
      .catch((err) => {
        if (!cancelled) showError(t("toast.loadFailed"), err);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, pageId]);

  // Sync title from cache on initial mount
  useEffect(() => {
    if (cachedPage && !title) {
      setTitle(normalizeWikiText(cachedPage.title));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleTitleBlur = async () => {
    if (!page || title === page.title) return;
    try {
      await wikiClient.updateWikiPage({
        projectId,
        pageId: page.id,
        title,
        icon: "",
      });
      onTitleChange(page.id, title);
    } catch (err) {
      showError(t("toast.saveFailed"), err);
    }
  };

  const handleContentChange = useCallback(
    async (json: string) => {
      try {
        await wikiClient.updateWikiPage({
          projectId,
          pageId,
          title: "",
          icon: "",
          contentJson: json,
          contentText: "",
        });

        // Extract page link IDs and sync to wiki_page_links table
        const targetIds = extractPageLinkIds(json);
        await wikiClient.syncLinks({
          projectId,
          sourcePageId: pageId,
          targetPageIds: targetIds,
        });
      } catch (err) {
        showError(t("toast.saveFailed"), err);
      }
    },
    [pageId, projectId, t],
  );

  const initialBlocks = useMemo(() => {
    if (!page?.contentJson) return undefined;
    try {
      const parsed = JSON.parse(page.contentJson);
      if (Array.isArray(parsed)) return parsed;
      // Handle ProseMirror doc wrapper: { type: "doc", content: [...] }
      if (parsed && typeof parsed === "object" && Array.isArray(parsed.content)) {
        return parsed.content;
      }
      return undefined;
    } catch {
      return undefined;
    }
  }, [page?.contentJson]);

  return (
    <div className="min-w-0">
      {titleLoading ? (
        <div className="mb-4 h-9 w-48 animate-pulse rounded bg-muted" />
      ) : (
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onBlur={handleTitleBlur}
          onKeyDown={(e) => {
            if (e.key === "Enter") e.currentTarget.blur();
          }}
          placeholder={t("project.wikiUntitled")}
          className="mb-4 w-full bg-transparent text-[22px] font-semibold leading-[1.3] outline-none placeholder:text-muted-foreground"
        />
      )}
      {page ? (
        <>
          {/* Lightweight static preview — no BlockNote, renders instantly */}
          {!editorReady && initialBlocks && initialBlocks.length > 0 && (
            <StaticContentPreview blocks={initialBlocks} />
          )}
          {/* Collaborative editor — lazy loaded via dynamic import, hidden until sync */}
          <div className={!editorReady && initialBlocks && initialBlocks.length > 0 ? "h-0 overflow-hidden" : ""}>
            <WikiEditor
              key={pageId}
              projectId={projectId}
              pageId={pageId}
              initialContent={initialBlocks}
              onContentChange={handleContentChange}
              onReady={() => setEditorReady(true)}
            />
          </div>
          <BacklinksPanel
            projectId={projectId}
            pageId={pageId}
            onNavigate={(id) => {
              window.dispatchEvent(
                new CustomEvent("wiki-navigate", { detail: { pageId: id } }),
              );
            }}
          />
        </>
      ) : (
        <div className="flex items-center justify-center py-20">
          <div className="size-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      )}
    </div>
  );
}

function StaticContentPreview({ blocks }: { blocks: unknown[] }) {
  return (
    <div className="wiki-static-preview">
      {blocks.map((block, i) => (
        <StaticBlock key={i} block={block} />
      ))}
    </div>
  );
}

function renderInlineContent(items: unknown[]): React.ReactNode[] {
  return items.map((item, i) => {
    const c = item as Record<string, unknown>;
    if (c.type === "text") {
      const styles = (c.styles || {}) as Record<string, boolean>;
      let el: React.ReactNode = c.text as string;
      if (styles.bold) el = <strong>{el}</strong>;
      if (styles.italic) el = <em>{el}</em>;
      if (styles.code) el = <code className="rounded bg-muted px-1 py-0.5 text-sm">{el}</code>;
      if (styles.strikethrough) el = <s>{el}</s>;
      if (styles.underline) el = <u>{el}</u>;
      return <span key={i}>{el}</span>;
    }
    if (c.type === "pageLink") {
      const lp = (c.props || {}) as Record<string, string>;
      return <span key={i} className="wiki-page-link">{lp.pageTitle || "Untitled"}</span>;
    }
    if (c.type === "link") {
      const content = c.content as unknown[] | undefined;
      return <a key={i} className="text-primary underline">{content ? renderInlineContent(content) : (c.href as string)}</a>;
    }
    return null;
  });
}

function StaticBlock({ block }: { block: unknown }) {
  const b = block as Record<string, unknown>;
  const type = (b.type as string) || "paragraph";
  const props = (b.props || {}) as Record<string, unknown>;
  const content = b.content as unknown[] | undefined;
  const children = b.children as unknown[] | undefined;

  const inline = content && content.length > 0 ? renderInlineContent(content) : null;
  const childBlocks = children && children.length > 0 ? (
    <div className="ml-6">
      {children.map((child, i) => <StaticBlock key={i} block={child} />)}
    </div>
  ) : null;

  switch (type) {
    case "heading": {
      const level = (props.level as number) || 1;
      if (level === 1) return <><h1 className="mb-2 mt-6 text-3xl font-bold">{inline}</h1>{childBlocks}</>;
      if (level === 2) return <><h2 className="mb-2 mt-5 text-2xl font-bold">{inline}</h2>{childBlocks}</>;
      return <><h3 className="mb-2 mt-4 text-xl font-bold">{inline}</h3>{childBlocks}</>;
    }
    case "bulletListItem":
      return <><div className="my-0.5 flex gap-2"><span className="shrink-0">•</span><span>{inline}</span></div>{childBlocks}</>;
    case "numberedListItem":
      return <><div className="my-0.5">{inline}</div>{childBlocks}</>;
    case "image":
      return <><img src={props.url as string} alt={(props.caption as string) || ""} className="my-2 max-w-full rounded" />{childBlocks}</>;
    case "table": {
      const tableContent = b.content as Record<string, unknown> | undefined;
      const rows = (tableContent?.rows as unknown[]) || [];
      return (
        <table className="my-2 w-full border-collapse text-sm">
          <tbody>
            {rows.map((row, ri) => {
              const cells = ((row as Record<string, unknown>).cells as unknown[][]) || [];
              return (
                <tr key={ri} className="border-b border-border">
                  {cells.map((cell, ci) => (
                    <td key={ci} className="border border-border px-3 py-1.5">
                      {renderInlineContent(cell)}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      );
    }
    case "horizontalRule":
      return <hr className="my-4 border-border" />;
    default:
      if (!inline) return <div className="h-[1.5em]" />;
      return <><p className="my-1 leading-relaxed">{inline}</p>{childBlocks}</>;
  }
}

/**
 * Recursively extract all pageLink pageId values from BlockNote JSON.
 */
function extractPageLinkIds(json: string): string[] {
  const ids = new Set<string>();
  try {
    const blocks = JSON.parse(json);
    const walk = (obj: unknown) => {
      if (!obj || typeof obj !== "object") return;
      if (Array.isArray(obj)) {
        for (const item of obj) walk(item);
        return;
      }
      const rec = obj as Record<string, unknown>;
      if (rec.type === "pageLink" && rec.props) {
        const pageId = (rec.props as Record<string, unknown>).pageId;
        if (typeof pageId === "string" && pageId) ids.add(pageId);
      }
      for (const val of Object.values(rec)) walk(val);
    };
    walk(blocks);
  } catch {
    // malformed JSON — return empty
  }
  return Array.from(ids);
}

function BacklinksPanel({
  projectId,
  pageId,
  onNavigate,
}: {
  projectId: bigint;
  pageId: string;
  onNavigate: (pageId: string) => void;
}) {
  const { t } = useI18n();
  const [backlinks, setBacklinks] = useState<
    Array<{ sourcePageId: string; sourcePageTitle: string; sourcePageIcon: string }>
  >([]);

  useEffect(() => {
    wikiClient
      .getBacklinks({ projectId, pageId })
      .then((res) =>
        setBacklinks(
          res.links.map((l) => ({
            sourcePageId: l.sourcePageId,
            sourcePageTitle: l.sourcePageTitle || t("project.wikiUntitled"),
            sourcePageIcon: l.sourcePageIcon || "",
          })),
        ),
      )
      .catch(() => {});
  }, [projectId, pageId, t]);

  if (backlinks.length === 0) return null;

  return (
    <div className="mt-8 border-t border-border pt-4">
      <div className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        <Link2 className="size-3" />
        {t("project.wikiBacklinks")}
      </div>
      <div className="space-y-1">
        {backlinks.map((link) => (
          <button
            key={link.sourcePageId}
            onClick={() => onNavigate(link.sourcePageId)}
            className="block w-full text-left text-sm text-primary hover:underline"
          >
            {link.sourcePageIcon ? `${link.sourcePageIcon} ` : ""}
            {link.sourcePageTitle}
          </button>
        ))}
      </div>
    </div>
  );
}
