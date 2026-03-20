"use client";

import { useState } from "react";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical, User, Trash2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface TaskCardProps {
  task: {
    id: bigint;
    title: string;
    description: string;
    status: string;
    specRef: string;
    assigneeId?: bigint;
    assigneeName: string;
    orderIndex: number;
  };
  members: Array<{ userId: bigint; userName: string }>;
  onUpdate: (id: bigint, fields: Record<string, unknown>) => void;
  onDelete: (id: bigint) => void;
  isDragging?: boolean;
}

function getInitials(name: string): string {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((n) => n[0].toUpperCase())
    .join("");
}

export function TaskCard({ task, members, onUpdate, onDelete, isDragging }: TaskCardProps) {
  const { t } = useI18n();
  const [isExpanded, setIsExpanded] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(false);

  // Local state for each editable field
  const [title, setTitle] = useState(task.title);
  const [description, setDescription] = useState(task.description);
  const [status, setStatus] = useState(task.status);
  const [specRef, setSpecRef] = useState(task.specRef);
  const [assigneeId, setAssigneeId] = useState<string>(
    task.assigneeId !== undefined ? String(task.assigneeId) : "",
  );

  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({
    id: String(task.id),
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const isDone = task.status === "done";

  function handleBlurTitle() {
    if (title !== task.title) {
      onUpdate(task.id, { title });
    }
  }

  function handleBlurDescription() {
    if (description !== task.description) {
      onUpdate(task.id, { description });
    }
  }

  function handleStatusChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const newStatus = e.target.value;
    setStatus(newStatus);
    onUpdate(task.id, { status: newStatus });
  }

  function handleBlurSpecRef() {
    if (specRef !== task.specRef) {
      onUpdate(task.id, { specRef });
    }
  }

  function handleAssigneeChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const value = e.target.value;
    setAssigneeId(value);
    onUpdate(task.id, {
      assigneeId: value === "" ? null : BigInt(value),
    });
  }

  function handleDeleteClick() {
    if (deleteConfirm) {
      onDelete(task.id);
    } else {
      setDeleteConfirm(true);
    }
  }

  function handleCardClick() {
    if (!isExpanded) {
      setIsExpanded(true);
    }
  }

  function handleCollapse(e: React.MouseEvent) {
    e.stopPropagation();
    setIsExpanded(false);
    setDeleteConfirm(false);
  }

  const statusOptions = [
    { value: "todo", label: t("tasks.statusTodo") },
    { value: "in_progress", label: t("tasks.statusInProgress") },
    { value: "done", label: t("tasks.statusDone") },
  ];

  if (isExpanded) {
    return (
      <div
        ref={setNodeRef}
        style={style}
        className="rounded-lg border border-border bg-card p-3 shadow-sm"
      >
        {/* Drag handle row */}
        <div className="mb-2 flex items-center gap-2">
          <button
            {...attributes}
            {...listeners}
            className="flex min-h-[44px] min-w-[44px] cursor-grab items-center justify-center rounded text-muted-foreground hover:text-foreground active:cursor-grabbing"
            tabIndex={-1}
          >
            <GripVertical className="h-4 w-4" />
          </button>
          <button
            onClick={handleCollapse}
            className="ml-auto text-xs text-muted-foreground hover:text-foreground transition-colors duration-200"
          >
            {t("common.done")}
          </button>
        </div>

        {/* Title */}
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onBlur={handleBlurTitle}
          placeholder={t("tasks.titlePlaceholder")}
          className="mb-2 w-full rounded border border-border bg-transparent px-2 py-1.5 text-sm font-medium outline-none focus:border-primary transition-colors duration-200"
        />

        {/* Description */}
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          onBlur={handleBlurDescription}
          placeholder={t("tasks.descriptionPlaceholder")}
          rows={3}
          className="mb-2 w-full resize-none rounded border border-border bg-transparent px-2 py-1.5 text-sm text-muted-foreground outline-none focus:border-primary transition-colors duration-200"
        />

        {/* Status */}
        <div className="mb-2">
          <label className="mb-1 block text-xs font-medium text-muted-foreground">
            Status
          </label>
          <select
            value={status}
            onChange={handleStatusChange}
            className="w-full rounded border border-border bg-card px-2 py-1.5 text-sm outline-none focus:border-primary transition-colors duration-200 min-h-[44px]"
          >
            {statusOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        {/* Spec Ref */}
        <div className="mb-2">
          <label className="mb-1 block text-xs font-medium text-muted-foreground">
            Spec Ref
          </label>
          <input
            value={specRef}
            onChange={(e) => setSpecRef(e.target.value)}
            onBlur={handleBlurSpecRef}
            placeholder={t("tasks.specRefPlaceholder")}
            className="w-full rounded border border-border bg-transparent px-2 py-1.5 text-sm outline-none focus:border-primary transition-colors duration-200"
          />
        </div>

        {/* Assignee */}
        <div className="mb-3">
          <label className="mb-1 block text-xs font-medium text-muted-foreground">
            {t("tasks.assignee")}
          </label>
          <select
            value={assigneeId}
            onChange={handleAssigneeChange}
            className="w-full rounded border border-border bg-card px-2 py-1.5 text-sm outline-none focus:border-primary transition-colors duration-200 min-h-[44px]"
          >
            <option value="">{t("tasks.unassigned")}</option>
            {members.map((m) => (
              <option key={String(m.userId)} value={String(m.userId)}>
                {m.userName}
              </option>
            ))}
          </select>
        </div>

        {/* Delete */}
        <button
          onClick={handleDeleteClick}
          onBlur={() => setDeleteConfirm(false)}
          className="flex min-h-[44px] w-full items-center justify-center gap-1.5 rounded border border-destructive/30 px-2 py-1.5 text-sm text-destructive hover:bg-destructive/10 transition-colors duration-200"
        >
          <Trash2 className="h-4 w-4" />
          {deleteConfirm ? t("tasks.deleteConfirm") : t("common.delete")}
        </button>
      </div>
    );
  }

  // Collapsed state
  return (
    <div
      ref={setNodeRef}
      style={style}
      onClick={handleCardClick}
      className={[
        "group cursor-pointer rounded-lg border border-border bg-card p-3 shadow-sm hover:border-border/80 hover:shadow-md transition-all duration-200",
        isDone ? "opacity-70" : "",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <div className="flex items-start gap-2">
        {/* Drag handle */}
        <button
          {...attributes}
          {...listeners}
          onClick={(e) => e.stopPropagation()}
          className="mt-0.5 flex min-h-[44px] min-w-[44px] cursor-grab items-center justify-center rounded text-muted-foreground opacity-0 group-hover:opacity-100 hover:text-foreground active:cursor-grabbing transition-opacity duration-200"
          tabIndex={-1}
        >
          <GripVertical className="h-4 w-4" />
        </button>

        {/* Content */}
        <div className="min-w-0 flex-1">
          <p
            className={[
              "mb-1 text-sm font-medium leading-snug",
              isDone ? "line-through text-muted-foreground" : "",
            ]
              .filter(Boolean)
              .join(" ")}
          >
            {task.title}
          </p>

          {task.description && (
            <p className="mb-2 line-clamp-2 text-xs text-muted-foreground leading-relaxed">
              {task.description}
            </p>
          )}

          <div className="flex items-center gap-2 flex-wrap">
            {task.specRef && (
              <span className="inline-flex items-center rounded-full border border-border/60 bg-muted/50 px-2 py-0.5 text-xs text-muted-foreground">
                {task.specRef}
              </span>
            )}

            {task.assigneeName ? (
              <div className="ml-auto flex min-h-[28px] min-w-[28px] items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
                {getInitials(task.assigneeName)}
              </div>
            ) : (
              <div className="ml-auto flex min-h-[28px] min-w-[28px] items-center justify-center rounded-full border border-dashed border-border text-muted-foreground">
                <User className="h-3 w-3" />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
