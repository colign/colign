"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { isLoggedIn } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";

export default function AuthCallbackPage() {
  const router = useRouter();
  const { t } = useI18n();

  useEffect(() => {
    // Cookies are set by the server before redirect — just check and navigate.
    if (isLoggedIn()) {
      const pendingInvite = sessionStorage.getItem("pending_invite_token");
      if (pendingInvite) {
        sessionStorage.removeItem("pending_invite_token");
        router.push(`/invite/${pendingInvite}`);
      } else {
        router.push("/");
      }
    } else {
      router.push("/auth");
    }
  }, [router]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p>{t("common.authenticating")}</p>
    </div>
  );
}
