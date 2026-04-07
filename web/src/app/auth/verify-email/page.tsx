"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useI18n } from "@/lib/i18n";
import { authClient, getTokenPayload } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CheckCircle2, XCircle, Loader2 } from "lucide-react";

export default function VerifyEmailPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useI18n();
  const token = searchParams.get("token");

  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [resendStatus, setResendStatus] = useState<"idle" | "sending" | "sent">("idle");

  useEffect(() => {
    if (!token) {
      setStatus("error");
      return;
    }

    authClient.verifyEmail({ token }).then(
      () => setStatus("success"),
      () => setStatus("error"),
    );
  }, [token]);

  function handleContinue() {
    const pendingInvite = sessionStorage.getItem("pending_invite_token");
    if (pendingInvite) {
      sessionStorage.removeItem("pending_invite_token");
      router.push(`/invite/${pendingInvite}`);
      return;
    }

    // Try opening desktop app via deep link, fallback to web
    window.location.href = "colign://auth/verified";
    setTimeout(() => {
      router.push("/projects");
    }, 1500);
  }

  async function handleResend() {
    if (resendStatus === "sending") return;
    const payload = getTokenPayload();
    if (!payload?.email) return;
    setResendStatus("sending");
    try {
      await authClient.resendVerificationEmail({ email: payload.email });
      setResendStatus("sent");
    } catch {
      setResendStatus("idle");
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md border-border/50 bg-card/50 backdrop-blur-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-lg">
            {status === "loading" && t("auth.verifyingEmail")}
            {status === "success" && t("auth.emailVerified")}
            {status === "error" && t("auth.verifyEmailFailed")}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4">
          {status === "loading" && <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />}
          {status === "success" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-green-500" />
              <Button className="w-full cursor-pointer" onClick={handleContinue}>
                {t("auth.continueToProjects")}
              </Button>
            </>
          )}
          {status === "error" && (
            <>
              <XCircle className="h-8 w-8 text-destructive" />
              <Button
                variant="outline"
                className="w-full cursor-pointer"
                disabled={resendStatus === "sending"}
                onClick={handleResend}
              >
                {resendStatus === "sending" && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {resendStatus === "sent" ? t("auth.resendSuccess") : t("auth.resendVerification")}
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
