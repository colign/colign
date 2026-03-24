"use client";

import { I18nProvider } from "@/lib/i18n";
import { OrgProvider } from "@/lib/org-context";
import { EventProvider } from "@/lib/events";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarLayout } from "@/components/layout/sidebar-layout";
import { CommandPalette } from "@/components/command-palette";
import { Toaster } from "sonner";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <I18nProvider>
      <OrgProvider>
        <EventProvider>
          <TooltipProvider>
            <SidebarLayout>{children}</SidebarLayout>
            <CommandPalette />
            <Toaster
              position="bottom-right"
              toastOptions={{
                className: "!bg-card !text-foreground !border-border",
              }}
              closeButton
            />
          </TooltipProvider>
        </EventProvider>
      </OrgProvider>
    </I18nProvider>
  );
}
