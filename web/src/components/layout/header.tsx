"use client";

import { useRef, useState, useEffect } from "react";
import Image from "next/image";
import Link from "next/link";
import { Github } from "lucide-react";
import { useOrg } from "@/lib/org-context";

interface Breadcrumb {
  label: string;
  href?: string;
  editable?: boolean;
  editablePrefix?: string;
  editableValue?: string;
  onSave?: (value: string) => void;
}

interface HeaderProps {
  breadcrumbs?: Breadcrumb[];
  actions?: React.ReactNode;
}

export function Header({ breadcrumbs = [], actions }: HeaderProps) {
  const { currentOrg, orgs, switchOrg } = useOrg();
  const [orgMenuOpen, setOrgMenuOpen] = useState(false);
  const orgMenuRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (orgMenuRef.current && !orgMenuRef.current.contains(e.target as Node)) {
        setOrgMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <header className="sticky top-0 z-30 border-b border-border/50 bg-background/80 backdrop-blur-md">
      <div className="flex items-center justify-between px-6 py-3">
        <div className="flex items-center gap-3">
          <Link href="/projects" className="flex items-center">
            <Image
              src="/logo.png"
              alt="Colign"
              width={24}
              height={24}
              priority
              className="hidden dark:block"
            />
            <Image
              src="/logo-dark.png"
              alt="Colign"
              width={24}
              height={24}
              priority
              className="block dark:hidden"
            />
          </Link>

          {/* Org Switcher */}
          {currentOrg && (
            <>
              <svg
                className="h-3.5 w-3.5 text-muted-foreground/40"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8.25 4.5l7.5 7.5-7.5 7.5"
                />
              </svg>
              <div className="relative" ref={orgMenuRef}>
                <button
                  onClick={() => setOrgMenuOpen(!orgMenuOpen)}
                  className="cursor-pointer flex items-center gap-1.5 rounded-lg px-2 py-1 text-sm font-medium text-foreground/80 transition-colors hover:bg-accent hover:text-foreground"
                >
                  <svg
                    className="h-3.5 w-3.5 text-muted-foreground"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={1.5}
                      d="M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21"
                    />
                  </svg>
                  {currentOrg.name}
                  {orgs.length > 1 && (
                    <svg
                      className="h-3 w-3 text-muted-foreground/50"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M19.5 8.25l-7.5 7.5-7.5-7.5"
                      />
                    </svg>
                  )}
                </button>
                {orgMenuOpen && orgs.length > 1 && (
                  <div className="absolute left-0 top-9 z-50 w-56 rounded-xl border border-border/50 bg-popover p-1.5 shadow-xl animate-in fade-in slide-in-from-top-2 duration-150">
                    {orgs.map((org) => (
                      <button
                        key={String(org.id)}
                        onClick={() => {
                          setOrgMenuOpen(false);
                          if (org.id !== currentOrg.id) {
                            switchOrg(org.id);
                          }
                        }}
                        className={`flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors ${
                          org.id === currentOrg.id
                            ? "bg-accent text-foreground"
                            : "text-foreground/80 hover:bg-accent"
                        }`}
                      >
                        <svg
                          className="h-3.5 w-3.5 text-muted-foreground"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={1.5}
                            d="M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21"
                          />
                        </svg>
                        <span className="truncate">{org.name}</span>
                        {org.id === currentOrg.id && (
                          <svg
                            className="ml-auto h-3.5 w-3.5 text-primary"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M4.5 12.75l6 6 9-13.5"
                            />
                          </svg>
                        )}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}

          {breadcrumbs.map((crumb, i) => (
            <div key={i} className="flex items-center gap-3">
              <svg
                className="h-3.5 w-3.5 text-muted-foreground/40"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8.25 4.5l7.5 7.5-7.5 7.5"
                />
              </svg>
              {crumb.href ? (
                <Link
                  href={crumb.href}
                  className="text-sm text-muted-foreground transition-colors duration-200 hover:text-foreground"
                >
                  {crumb.label}
                </Link>
              ) : crumb.editable && crumb.onSave ? (
                <EditableBreadcrumb
                  label={crumb.label}
                  prefix={crumb.editablePrefix}
                  editableValue={crumb.editableValue}
                  onSave={crumb.onSave}
                />
              ) : (
                <span className="text-sm font-medium">{crumb.label}</span>
              )}
            </div>
          ))}
        </div>

        <div className="flex items-center gap-2">
          <a
            href="https://github.com/colign/colign"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <Github className="h-4 w-4" />
          </a>
          {actions}
        </div>
      </div>
    </header>
  );
}

function EditableBreadcrumb({
  label,
  prefix,
  editableValue,
  onSave,
}: {
  label: string;
  prefix?: string;
  editableValue?: string;
  onSave: (value: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  const actualValue = editableValue ?? label;
  const [value, setValue] = useState(actualValue);
  const inputRef = useRef<HTMLInputElement>(null);
  const committedRef = useRef(false);

  useEffect(() => {
    setValue(editableValue ?? label);
  }, [editableValue, label]);

  useEffect(() => {
    if (editing) {
      committedRef.current = false;
      inputRef.current?.select();
    }
  }, [editing]);

  function commit() {
    if (committedRef.current) return;
    committedRef.current = true;
    const trimmed = value.trim();
    if (trimmed && trimmed !== actualValue) {
      onSave(trimmed);
    } else {
      setValue(actualValue);
    }
    setEditing(false);
  }

  if (editing) {
    return (
      <span className="inline-flex items-center gap-1.5">
        {prefix && <span className="text-sm text-muted-foreground">{prefix}</span>}
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") commit();
            if (e.key === "Escape") {
              setValue(actualValue);
              setEditing(false);
            }
          }}
          className="rounded-md border border-primary/50 bg-transparent px-1.5 py-0.5 text-sm font-medium text-foreground outline-none"
        />
      </span>
    );
  }

  return (
    <button
      onClick={() => setEditing(true)}
      className="cursor-pointer rounded-md px-1.5 py-0.5 text-sm font-medium transition-colors hover:bg-accent"
    >
      {label}
    </button>
  );
}
