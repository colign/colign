"use client";

import { useEffect } from "react";
import { useEvents } from "@/lib/events";
import { isDesktopApp } from "@/lib/push";
import { getTokenPayload } from "@/lib/auth";

/**
 * Bridges SSE notification events to native OS notifications in the Electron desktop app.
 * Renders nothing — purely a side-effect component.
 */
export function DesktopNotificationBridge() {
  const { on } = useEvents();

  useEffect(() => {
    if (typeof window === "undefined" || !isDesktopApp()) return;

    return on((event) => {
      if (!event.type.startsWith("notification_")) return;
      try {
        const data = JSON.parse(event.payload);
        const payload = getTokenPayload();
        if (!payload || data.userId !== payload.user_id) return;
        new Notification(data.title, {
          body: data.body,
          data: { url: data.url },
        });
      } catch {
        // Ignore parse errors
      }
    });
  }, [on]);

  return null;
}
