"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { createClient } from "@connectrpc/connect";
import {
  LayoutDashboard,
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
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { CreateOrganizationDialog } from "@/components/organization/create-organization-dialog";
import { CreateProjectDialog } from "@/components/project/create-project-dialog";
import { useOrg } from "@/lib/org-context";
import { projectClient } from "@/lib/project";
import { notificationClient } from "@/lib/notification";
import { NOTIFICATIONS_UPDATED_EVENT } from "@/lib/notification-events";
import { useEvents } from "@/lib/events";
import { useI18n } from "@/lib/i18n";
import { toProjectPath, toProjectRef } from "@/lib/project-ref";
import { clearTokens, getTokenPayload } from "@/lib/auth";
import { showError } from "@/lib/toast";
import { transport } from "@/lib/connect";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";

interface Project {
  id: bigint;
  name: string;
  slug: string;
}

const meClient = createClient(AuthService, transport);

export function AppSidebar() {
  const pathname = usePathname();
  const { toggleSidebar, state } = useSidebar();
  const { on } = useEvents();
  const { t } = useI18n();
  const { currentOrg, orgs, switchOrg } = useOrg();
  const [projects, setProjects] = useState<Project[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [orgMenuOpen, setOrgMenuOpen] = useState(false);
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const payload = typeof window !== "undefined" ? getTokenPayload() : null;
  const userEmail = payload?.email ?? "";
  const [profile, setProfile] = useState<{ name: string; email: string; avatarUrl: string }>({
    name: payload?.name || userEmail.split("@")[0],
    email: userEmail,
    avatarUrl: "",
  });
  const userName = profile.name || userEmail.split("@")[0];

  useEffect(() => {
    meClient
      .me({})
      .then((res) => {
        setProfile({
          name: res.name || payload?.name || userEmail.split("@")[0],
          email: res.email || userEmail,
          avatarUrl: res.avatarUrl || "",
        });
      })
      .catch((err) => showError("Failed to load projects", err));
  }, [payload?.name, userEmail]);

  useEffect(() => {
    const handleProfileUpdated = (event: Event) => {
      const detail = (event as CustomEvent<{ name: string; email: string; avatarUrl: string }>)
        .detail;
      if (!detail) return;
      setProfile(detail);
    };
    window.addEventListener("colign:profile-updated", handleProfileUpdated);
    return () => window.removeEventListener("colign:profile-updated", handleProfileUpdated);
  }, []);

  useEffect(() => {
    function fetchUnread() {
      notificationClient
        .getUnreadCount({})
        .then((res) => setUnreadCount(res.count))
        .catch(() => {});
    }

    fetchUnread();
    const handleNotificationsUpdated = () => fetchUnread();
    window.addEventListener(NOTIFICATIONS_UPDATED_EVENT, handleNotificationsUpdated);
    const interval = setInterval(fetchUnread, 30000);
    return () => {
      window.removeEventListener(NOTIFICATIONS_UPDATED_EVENT, handleNotificationsUpdated);
      clearInterval(interval);
    };
  }, [currentOrg, pathname]);

  useEffect(() => {
    return on((event) => {
      if (event.type.startsWith("notification_")) {
        notificationClient
          .getUnreadCount({})
          .then((res) => setUnreadCount(res.count))
          .catch(() => {});
      }
    });
  }, [on]);

  useEffect(() => {
    async function loadProjects() {
      try {
        const res = await projectClient.listProjects({});
        setProjects(res.projects.map((p) => ({ id: p.id, name: p.name, slug: p.slug })));
      } catch (err) {
        showError("Failed to load projects", err);
      }
    }
    loadProjects();
  }, [currentOrg, pathname]);

  const openSearch = () => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true }));
  };

  const navItems = [
    { label: t("sidebar.dashboard"), href: "/dashboard", icon: LayoutDashboard },
    { label: t("sidebar.inbox"), href: "/inbox", icon: Inbox },
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
              <div className="flex min-w-0 flex-col gap-0.5 leading-none">
                <span className="truncate font-semibold">Colign</span>
                {currentOrg && (
                  <span className="truncate text-xs text-muted-foreground">{currentOrg.name}</span>
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
                  {item.href === "/inbox" && unreadCount > 0 && (
                    <SidebarMenuBadge className="top-1.5 right-1 bg-red-500 text-white peer-hover/menu-button:text-white peer-data-active/menu-button:text-white group-data-[collapsible=icon]:absolute group-data-[collapsible=icon]:top-1 group-data-[collapsible=icon]:right-1 group-data-[collapsible=icon]:flex group-data-[collapsible=icon]:h-2.5 group-data-[collapsible=icon]:w-2.5 group-data-[collapsible=icon]:min-w-0 group-data-[collapsible=icon]:rounded-full group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:items-center group-data-[collapsible=icon]:justify-center">
                      <span className="group-data-[collapsible=icon]:hidden">
                        {unreadCount > 99 ? "99+" : unreadCount}
                      </span>
                    </SidebarMenuBadge>
                  )}
                </SidebarMenuItem>
              ))}
              <SidebarMenuItem>
                <SidebarMenuButton tooltip={`${t("sidebar.search")} (⌘K)`} onClick={openSearch}>
                  <Search className="size-4" />
                  <span>{t("sidebar.search")}</span>
                  <kbd className="ml-auto text-[10px] text-muted-foreground tracking-widest">
                    ⌘K
                  </kbd>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarSeparator />

        {/* Projects */}
        <SidebarGroup>
          <SidebarGroupLabel
            className="cursor-pointer hover:text-sidebar-foreground"
            render={<Link href="/projects" />}
          >
            {t("sidebar.projects")}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {projects.map((project) => (
                <SidebarMenuItem key={String(project.id)}>
                  <SidebarMenuButton
                    isActive={pathname.startsWith(`/projects/${toProjectRef(project)}`)}
                    tooltip={project.name}
                    render={<Link href={toProjectPath(project)} />}
                  >
                    <FolderKanban className="size-4" />
                    <span className="truncate">{project.name}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
              <SidebarMenuItem>
                <CreateProjectDialog onCreated={() => {}}>
                  <SidebarMenuButton tooltip={t("sidebar.newProject")}>
                    <Plus className="size-4" />
                    <span className="text-muted-foreground">{t("sidebar.newProject")}</span>
                  </SidebarMenuButton>
                </CreateProjectDialog>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip={state === "expanded" ? t("sidebar.collapse") : t("sidebar.expand")}
              onClick={toggleSidebar}
            >
              {state === "expanded" ? (
                <ChevronsLeft className="size-4" />
              ) : (
                <ChevronsRight className="size-4" />
              )}
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
          {currentOrg && (
            <SidebarMenuItem>
              <div className="relative">
                <SidebarMenuButton
                  tooltip={t("sidebar.switchOrg")}
                  onClick={() => setOrgMenuOpen(!orgMenuOpen)}
                >
                  <ArrowLeftRight className="size-4" />
                  <span className="truncate">{currentOrg.name}</span>
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
                    <div className="my-1 h-px bg-border" />
                    <CreateOrganizationDialog
                      onCreated={() => setOrgMenuOpen(false)}
                      triggerClassName="flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
                    />
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
                <Avatar>
                  <AvatarImage src={profile.avatarUrl} alt={userName} />
                  <AvatarFallback className="bg-emerald-600 text-xs font-bold uppercase text-white">
                    {userName.charAt(0) || "?"}
                  </AvatarFallback>
                </Avatar>
                <div className="flex min-w-0 flex-col gap-0.5 leading-none">
                  <span className="truncate text-sm font-medium">{userName}</span>
                  <span className="truncate text-xs text-muted-foreground">{profile.email}</span>
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
                      onClick={async () => {
                        setProfileMenuOpen(false);
                        await clearTokens();
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
