"use client";

import { useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { saveTokens } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";

export default function AuthCallbackPage() {
  const router = useRouter();
  const { t } = useI18n();
  const searchParams = useSearchParams();

  useEffect(() => {
    const accessToken = searchParams.get("access_token");
    const refreshToken = searchParams.get("refresh_token");

    if (accessToken && refreshToken) {
      saveTokens(accessToken, refreshToken);
      // Check for pending invitation
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
  }, [searchParams, router]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p>{t("common.authenticating")}</p>
    </div>
  );
}
