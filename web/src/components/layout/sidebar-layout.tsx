"use client";

import { useState, useEffect } from "react";
import { usePathname } from "next/navigation";
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "./app-sidebar";

const NO_SIDEBAR_PATHS = ["/auth", "/onboarding"];

export function SidebarLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [hasToken, setHasToken] = useState(false);

  useEffect(() => {
    setHasToken(!!localStorage.getItem("colign_access_token"));
  }, [pathname]);

  const hideSidebar = NO_SIDEBAR_PATHS.some((p) => pathname.startsWith(p)) || !hasToken;

  if (hideSidebar) {
    return <>{children}</>;
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        {/* Mobile hamburger */}
        <div className="flex items-center gap-2 border-b border-border/50 px-4 py-2 md:hidden">
          <SidebarTrigger />
          <span className="text-sm font-semibold">Colign</span>
        </div>
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
