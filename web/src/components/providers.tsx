"use client";

import { ThemeProvider } from "@/lib/theme-context";
import { I18nProvider } from "@/lib/i18n";
import { OrgProvider } from "@/lib/org-context";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarLayout } from "@/components/layout/sidebar-layout";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <I18nProvider>
        <OrgProvider>
          <TooltipProvider>
            <SidebarLayout>{children}</SidebarLayout>
          </TooltipProvider>
        </OrgProvider>
      </I18nProvider>
    </ThemeProvider>
  );
}
