"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Home,
  Inbox,
  Search,
  Plus,
  Settings,
  ChevronsLeft,
  ChevronsRight,
  ArrowLeftRight,
  FolderKanban,
  Palette,
  LogOut,
  UserCog,
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import { useOrg } from "@/lib/org-context";
import { projectClient } from "@/lib/project";
import { useI18n } from "@/lib/i18n";
import { clearTokens, getTokenPayload } from "@/lib/auth";

interface Project {
  id: bigint;
  name: string;
  slug: string;
}

export function AppSidebar() {
  const pathname = usePathname();
  const { toggleSidebar, state } = useSidebar();
  const { t } = useI18n();
  const { currentOrg, orgs, switchOrg } = useOrg();
  const [projects, setProjects] = useState<Project[]>([]);
  const [orgMenuOpen, setOrgMenuOpen] = useState(false);
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const payload = typeof window !== "undefined" ? getTokenPayload() : null;
  const userEmail = payload?.email ?? "";
  const userName = payload?.name || userEmail.split("@")[0];

  useEffect(() => {
    async function loadProjects() {
      try {
        const res = await projectClient.listProjects({});
        setProjects(
          res.projects.map((p) => ({ id: p.id, name: p.name, slug: p.slug })),
        );
      } catch {
        // not loaded
      }
    }
    loadProjects();
  }, [currentOrg, pathname]);

  const navItems = [
    { label: t("sidebar.home"), href: "/home", icon: Home },
    { label: t("sidebar.inbox"), href: "/inbox", icon: Inbox },
    { label: t("sidebar.search"), href: "/search", icon: Search },
  ];

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" tooltip="Colign" render={<Link href="/projects" />}>
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <span className="text-sm font-bold">C</span>
              </div>
              <div className="flex flex-col gap-0.5 leading-none">
                <span className="font-semibold">Colign</span>
                {currentOrg && (
                  <span className="text-xs text-muted-foreground">{currentOrg.name}</span>
                )}
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        {/* Navigation */}
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
                    isActive={pathname === item.href}
                    tooltip={item.label}
                    render={<Link href={item.href} />}
                  >
                    <item.icon className="size-4" />
                    <span>{item.label}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarSeparator />

        {/* Projects */}
        <SidebarGroup>
          <SidebarGroupLabel>{t("sidebar.projects")}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {projects.map((project) => (
                <SidebarMenuItem key={String(project.id)}>
                  <SidebarMenuButton
                    isActive={pathname.startsWith(`/projects/${project.slug}`)}
                    tooltip={project.name}
                    render={<Link href={`/projects/${project.slug}`} />}
                  >
                    <FolderKanban className="size-4" />
                    <span>{project.name}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
              <SidebarMenuItem>
                <SidebarMenuButton tooltip={t("sidebar.newProject")} render={<Link href="/projects/new" />}>
                  <Plus className="size-4" />
                  <span className="text-muted-foreground">{t("sidebar.newProject")}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip={state === "expanded" ? t("sidebar.collapse") : t("sidebar.expand")} onClick={toggleSidebar}>
              {state === "expanded" ? <ChevronsLeft className="size-4" /> : <ChevronsRight className="size-4" />}
              <span>{state === "expanded" ? t("sidebar.collapse") : t("sidebar.expand")}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={pathname.startsWith("/settings")}
              tooltip={t("sidebar.settings")}
              render={<Link href="/settings" />}
            >
              <Settings className="size-4" />
              <span>{t("sidebar.settings")}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          {currentOrg && orgs.length > 1 && (
            <SidebarMenuItem>
              <div className="relative">
                <SidebarMenuButton
                  tooltip={t("sidebar.switchOrg")}
                  onClick={() => setOrgMenuOpen(!orgMenuOpen)}
                >
                  <ArrowLeftRight className="size-4" />
                  <span>{currentOrg.name}</span>
                </SidebarMenuButton>
                {orgMenuOpen && (
                  <div className="absolute bottom-full left-0 z-50 mb-1 w-full rounded-md border border-border bg-popover p-1 shadow-md">
                    {orgs.map((org) => (
                      <button
                        key={String(org.id)}
                        onClick={() => {
                          setOrgMenuOpen(false);
                          if (org.id !== currentOrg.id) switchOrg(org.id);
                        }}
                        className={`flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm ${
                          org.id === currentOrg.id ? "bg-accent" : "hover:bg-accent"
                        }`}
                      >
                        {org.name}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </SidebarMenuItem>
          )}
          <SidebarMenuItem>
            <div className="relative">
              <SidebarMenuButton
                size="lg"
                tooltip={t("sidebar.profile")}
                onClick={() => setProfileMenuOpen(!profileMenuOpen)}
              >
                <div className="flex aspect-square size-8 items-center justify-center rounded-full bg-emerald-600 text-white text-xs font-bold uppercase">
                  {userName.charAt(0) || "?"}
                </div>
                <div className="flex flex-col gap-0.5 leading-none">
                  <span className="text-sm font-medium">{userName}</span>
                  <span className="text-xs text-muted-foreground">{userEmail}</span>
                </div>
              </SidebarMenuButton>
              {profileMenuOpen && (
                <>
                  <div className="fixed inset-0 z-40" onClick={() => setProfileMenuOpen(false)} />
                  <div className="absolute bottom-full left-0 z-50 mb-2 w-56 rounded-lg border border-border bg-popover p-1.5 shadow-xl">
                    <Link
                      href="/settings"
                      onClick={() => setProfileMenuOpen(false)}
                      className="flex w-full cursor-pointer items-center gap-2.5 rounded-md px-3 py-2 text-sm text-foreground/80 transition-colors hover:bg-accent hover:text-foreground"
                    >
                      <UserCog className="size-4 text-muted-foreground" />
                      {t("sidebar.profileSettings")}
                    </Link>
                    <Link
                      href="/settings"
                      onClick={() => setProfileMenuOpen(false)}
                      className="flex w-full cursor-pointer items-center gap-2.5 rounded-md px-3 py-2 text-sm text-foreground/80 transition-colors hover:bg-accent hover:text-foreground"
                    >
                      <Palette className="size-4 text-muted-foreground" />
                      {t("sidebar.appearance")}
                    </Link>
                    <div className="my-1 h-px bg-border" />
                    <button
                      onClick={() => {
                        setProfileMenuOpen(false);
                        clearTokens();
                        window.location.href = "/auth";
                      }}
                      className="flex w-full cursor-pointer items-center gap-2.5 rounded-md px-3 py-2 text-sm text-destructive transition-colors hover:bg-destructive/10"
                    >
                      <LogOut className="size-4" />
                      {t("sidebar.logOut")}
                    </button>
                  </div>
                </>
              )}
            </div>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
