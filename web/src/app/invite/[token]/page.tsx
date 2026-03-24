"use client";

import { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import { orgClient } from "@/lib/organization";
import { isLoggedIn } from "@/lib/auth";
import { showError } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type InvitationInfo = {
  orgName: string;
  email: string;
  role: string;
};

export default function InvitePage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const router = useRouter();
  const [invitation, setInvitation] = useState<InvitationInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [accepting, setAccepting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    async function fetchInvitation() {
      try {
        const res = await orgClient.getInvitation({ token });
        setInvitation({
          orgName: res.invitation?.organization?.name ?? "Unknown",
          email: res.invitation?.email ?? "",
          role: res.invitation?.role ?? "member",
        });
      } catch (err) {
        showError("Failed to load invitation", err);
        setError("This invitation is invalid or has expired.");
      } finally {
        setLoading(false);
      }
    }
    fetchInvitation();
  }, [token]);

  async function handleAccept() {
    if (!isLoggedIn()) {
      // Save invite token and redirect to auth
      sessionStorage.setItem("pending_invite_token", token);
      router.push("/auth");
      return;
    }

    setAccepting(true);
    setError("");
    try {
      await orgClient.acceptInvitation({ token });
      router.push("/projects");
    } catch (err: unknown) {
      showError("Failed to accept invitation", err);
      setError(err instanceof Error ? err.message : "Failed to accept invitation");
    } finally {
      setAccepting(false);
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold tracking-tight">
            Co<span className="text-primary">lign</span>
          </h1>
        </div>

        <Card className="border-border/50 bg-card/50 backdrop-blur-sm">
          <CardHeader className="text-center">
            <CardTitle className="text-xl">Organization Invitation</CardTitle>
            {invitation && (
              <CardDescription className="text-base">
                You&apos;ve been invited to join <strong>{invitation.orgName}</strong> as a{" "}
                {invitation.role}.
              </CardDescription>
            )}
          </CardHeader>
          <CardContent className="space-y-4">
            {error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-center">
                <p className="text-sm text-destructive">{error}</p>
              </div>
            )}

            {invitation && !error && (
              <>
                <div className="rounded-lg bg-accent/50 p-4 text-center text-sm text-muted-foreground">
                  Invitation for <strong className="text-foreground">{invitation.email}</strong>
                </div>

                <Button
                  onClick={handleAccept}
                  disabled={accepting}
                  className="w-full cursor-pointer"
                  size="lg"
                >
                  {accepting
                    ? "Joining..."
                    : isLoggedIn()
                      ? `Join ${invitation.orgName}`
                      : "Sign in to accept"}
                </Button>

                {!isLoggedIn() && (
                  <p className="text-center text-xs text-muted-foreground">
                    You need to sign in or create an account to accept this invitation.
                  </p>
                )}
              </>
            )}

            {!invitation && error && (
              <Button
                variant="outline"
                onClick={() => router.push("/auth")}
                className="w-full cursor-pointer"
              >
                Go to Sign In
              </Button>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
