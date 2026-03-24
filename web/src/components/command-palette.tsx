"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  FolderKanban,
  GitBranch,
  CheckSquare,
  LayoutDashboard,
  Inbox,
  Settings,
} from "lucide-react";
import { projectClient } from "@/lib/project";
import { showError } from "@/lib/toast";
import { useI18n } from "@/lib/i18n";

interface SearchResult {
  type: string;
  id: bigint;
  title: string;
  subtitle: string;
  slug: string;
  projectId: bigint;
}

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const router = useRouter();
  const { t } = useI18n();

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const search = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([]);
      return;
    }
    setLoading(true);
    try {
      const res = await projectClient.search({ query: q });
      setResults(
        res.results.map((r) => ({
          type: r.type,
          id: r.id,
          title: r.title,
          subtitle: r.subtitle,
          slug: r.slug,
          projectId: r.projectId,
        })),
      );
    } catch (err) {
      showError("Search failed", err);
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => search(query), 200);
    return () => clearTimeout(timer);
  }, [query, search]);

  const handleSelect = (result: SearchResult) => {
    setOpen(false);
    setQuery("");
    if (result.type === "project") {
      router.push(`/projects/${result.slug}`);
    } else if (result.type === "change") {
      router.push(`/projects/${result.slug}/changes/${result.id}`);
    } else if (result.type === "task") {
      router.push(`/projects/${result.slug}/changes/${result.id}`);
    }
  };

  const projects = results.filter((r) => r.type === "project");
  const changes = results.filter((r) => r.type === "change");
  const tasks = results.filter((r) => r.type === "task");

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder={t("search.placeholder")} value={query} onValueChange={setQuery} />
      <CommandList>
        {query.trim() === "" ? (
          <CommandGroup heading={t("search.quickLinks")}>
            <CommandItem
              onSelect={() => {
                setOpen(false);
                router.push("/dashboard");
              }}
            >
              <LayoutDashboard className="mr-2 size-4 text-muted-foreground" />
              {t("sidebar.dashboard")}
            </CommandItem>
            <CommandItem
              onSelect={() => {
                setOpen(false);
                router.push("/inbox");
              }}
            >
              <Inbox className="mr-2 size-4 text-muted-foreground" />
              {t("sidebar.inbox")}
            </CommandItem>
            <CommandItem
              onSelect={() => {
                setOpen(false);
                router.push("/settings");
              }}
            >
              <Settings className="mr-2 size-4 text-muted-foreground" />
              {t("sidebar.settings")}
            </CommandItem>
          </CommandGroup>
        ) : (
          <>
            {!loading && results.length === 0 && (
              <CommandEmpty>{t("search.noResults")}</CommandEmpty>
            )}
            {projects.length > 0 && (
              <CommandGroup heading={t("search.projects")}>
                {projects.map((r) => (
                  <CommandItem key={`p-${r.id}`} onSelect={() => handleSelect(r)}>
                    <FolderKanban className="mr-2 size-4 text-muted-foreground" />
                    <span className="flex-1">{r.title}</span>
                    <span className="text-xs text-muted-foreground">{r.subtitle}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
            {changes.length > 0 && (
              <CommandGroup heading={t("search.changes")}>
                {changes.map((r) => (
                  <CommandItem key={`c-${r.id}`} onSelect={() => handleSelect(r)}>
                    <GitBranch className="mr-2 size-4 text-muted-foreground" />
                    <span className="flex-1">{r.title}</span>
                    <span className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                      {r.subtitle}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
            {tasks.length > 0 && (
              <CommandGroup heading={t("search.tasks")}>
                {tasks.map((r) => (
                  <CommandItem key={`t-${r.id}`} onSelect={() => handleSelect(r)}>
                    <CheckSquare className="mr-2 size-4 text-muted-foreground" />
                    <span className="flex-1">{r.title}</span>
                    <span className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                      {r.subtitle}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </>
        )}
      </CommandList>
    </CommandDialog>
  );
}

export function useCommandPalette() {
  return {
    open: () => {
      document.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true }));
    },
  };
}
