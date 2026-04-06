"use client";

import { useCallback, useEffect, useRef, useState } from "react";
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
} from "lucide-react";
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "@/components/ui/resizable";
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
import { WikiEditor } from "@/components/wiki/wiki-editor";
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
  const [pages, setPages] = useState<WikiPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPageId, setSelectedPageId] = useState<string | null>(null);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [deleteTarget, setDeleteTarget] = useState<WikiPage | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor),
  );

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

  const tree = buildTree(pages);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="size-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <ResizablePanelGroup orientation="horizontal" className="min-h-[600px] w-full min-w-0" style={{ height: "calc(100vh - 22rem)" }}>
      {/* Sidebar */}
      <ResizablePanel defaultSize="25%" minSize="15%" maxSize="45%" className="min-w-0 overflow-y-auto overflow-x-hidden">
        <div className="mb-3 flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
            {t("project.wiki")}
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            onClick={() => handleCreatePage()}
          >
            <Plus className="size-3.5" />
          </Button>
        </div>

        {tree.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border/60 px-4 py-12 text-center">
            <FileText className="mb-3 size-8 text-muted-foreground/40" />
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
      </ResizablePanel>

      <ResizableHandle withHandle className="mx-1 hover:bg-primary/20 transition-colors" />

      {/* Content Area */}
      <ResizablePanel defaultSize="75%" className="min-w-0 pl-3">
        <div className="h-full overflow-y-auto overflow-x-hidden rounded-xl border border-border/40 bg-card/50 scrollbar-subtle">
          {selectedPageId ? (
            <WikiPageContent
              projectId={projectId}
              pageId={selectedPageId}
              onTitleChange={(pageId, title) => {
                setPages((prev) =>
                  prev.map((p) => (p.id === pageId ? { ...p, title } as WikiPage : p)),
                );
              }}
            />
          ) : (
            <div className="flex h-full items-center justify-center py-20">
              <div className="text-center">
                <FileText className="mx-auto mb-3 size-10 text-muted-foreground/30" />
                <p className="text-sm text-muted-foreground">
                  {t("project.wikiEmptyDesc")}
                </p>
              </div>
            </div>
          )}
        </div>
      </ResizablePanel>

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
    </ResizablePanelGroup>
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
            ? "bg-primary/10 text-primary"
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
          <FileText className="size-3.5 shrink-0 text-muted-foreground/60" />
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
  onTitleChange,
}: {
  projectId: bigint;
  pageId: string;
  onTitleChange: (pageId: string, title: string) => void;
}) {
  const { t } = useI18n();
  const [page, setPage] = useState<WikiPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [title, setTitle] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    wikiClient
      .getWikiPage({ projectId, pageId })
      .then((res) => {
        if (!cancelled) {
          setPage(res.page!);
          setTitle(normalizeWikiText(res.page?.title));
          setLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          showError(t("toast.loadFailed"), err);
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, pageId, t]);

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
      if (!page) return;
      try {
        await wikiClient.updateWikiPage({
          projectId,
          pageId: page.id,
          title: "",
          icon: "",
          contentJson: json,
          contentText: "",
        });
      } catch (err) {
        showError(t("toast.saveFailed"), err);
      }
    },
    [page, projectId, t],
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="size-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  if (!page) return null;

  const initialBlocks = page.contentJson
    ? (() => {
        try {
          const parsed = JSON.parse(page.contentJson);
          return Array.isArray(parsed) ? parsed : undefined;
        } catch {
          return undefined;
        }
      })()
    : undefined;

  return (
    <div className="min-w-0 p-6">
      <input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onBlur={handleTitleBlur}
        onKeyDown={(e) => {
          if (e.key === "Enter") e.currentTarget.blur();
        }}
        placeholder={t("project.wikiUntitled")}
        className="mb-4 w-full bg-transparent text-2xl font-bold outline-none placeholder:text-muted-foreground/40"
      />
      <WikiEditor
        key={pageId}
        projectId={projectId}
        pageId={pageId}
        initialContent={initialBlocks}
        onContentChange={handleContentChange}
      />
    </div>
  );
}
