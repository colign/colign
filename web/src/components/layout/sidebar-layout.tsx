"use client";

import { useState, useEffect, useCallback } from "react";
import Image from "next/image";
import { usePathname, useRouter } from "next/navigation";
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "./app-sidebar";
import { clearTokens, getAccessToken } from "@/lib/auth";
import { showError } from "@/lib/toast";
import { createClient, ConnectError, Code } from "@connectrpc/connect";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";
import { transport } from "@/lib/connect";

const NO_SIDEBAR_PATHS = ["/auth", "/onboarding"];

export function SidebarLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [hasToken, setHasToken] = useState(false);
  const [verified, setVerified] = useState(false);

  const verifySession = useCallback(async () => {
    const token = getAccessToken();
    if (!token) {
      setHasToken(false);
      setVerified(true);
      if (!NO_SIDEBAR_PATHS.some((p) => pathname.startsWith(p))) {
        router.replace("/auth");
      }
      return;
    }

    setHasToken(true);

    // Skip verification on auth pages
    if (NO_SIDEBAR_PATHS.some((p) => pathname.startsWith(p))) {
      setVerified(true);
      return;
    }

    try {
      const meClient = createClient(AuthService, transport);
      await meClient.me({});
      setVerified(true);
    } catch (err) {
      // Only redirect on auth-specific errors (bad token, user not found)
      if (
        err instanceof ConnectError &&
        (err.code === Code.Unauthenticated || err.code === Code.NotFound)
      ) {
        clearTokens();
        setHasToken(false);
        router.replace("/auth");
      }
      // Network errors, server errors, etc. — notify and proceed
      if (!(err instanceof ConnectError && (err.code === Code.Unauthenticated || err.code === Code.NotFound))) {
        showError("Failed to load data", err);
      }
      setVerified(true);
    }
  }, [pathname, router]);

  // Verify on mount and pathname change
  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      if (cancelled) return;
      await verifySession();
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [verifySession]);

  // Verify on window focus (tab switch back)
  useEffect(() => {
    const handleFocus = () => {
      verifySession();
    };
    window.addEventListener("focus", handleFocus);
    return () => window.removeEventListener("focus", handleFocus);
  }, [verifySession]);

  const hideSidebar = NO_SIDEBAR_PATHS.some((p) => pathname.startsWith(p)) || !hasToken;

  // Show nothing until verified to prevent flash
  if (!verified) {
    return null;
  }

  if (hideSidebar) {
    return <>{children}</>;
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="overflow-x-hidden">
        {/* Mobile hamburger */}
        <div className="flex items-center gap-2 border-b border-border/50 px-4 py-2 md:hidden">
          <SidebarTrigger />
          <Image src="/logo.png" alt="Colign" width={80} height={20} className="h-5 w-auto" />
        </div>
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
