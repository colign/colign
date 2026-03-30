import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";

const transport = createConnectTransport({
  baseUrl: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080",
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

export const authClient = createClient(AuthService, transport);

export const AUTH_CHANGED_EVENT = "colign:auth-changed";

const COOKIE_ACCESS = "colign_access_token";

/**
 * Save the access token cookie. The refresh token is HttpOnly
 * and set by the server via Set-Cookie headers.
 */
export function saveTokens(accessToken: string, _refreshToken?: string) {
  setCookie(COOKIE_ACCESS, accessToken);
  dispatchAuthChanged();
}

export function getAccessToken(): string | null {
  return getCookie(COOKIE_ACCESS);
}

/**
 * Clear the access token cookie and call the server logout
 * endpoint to clear the HttpOnly refresh token cookie.
 * Callers that navigate after logout should await this function.
 */
export async function clearTokens() {
  clearCookie(COOKIE_ACCESS);
  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  try {
    await fetch(`${apiUrl}/api/auth/logout`, {
      method: "POST",
      credentials: "include",
    });
  } catch {
    // Network error — cookie cleared client-side is enough to block new requests
  }
  dispatchAuthChanged();
}

export function isLoggedIn(): boolean {
  return !!getAccessToken();
}

interface JWTPayload {
  user_id: number;
  email: string;
  name: string;
  org_id: number;
}

export function getTokenPayload(): JWTPayload | null {
  const token = getAccessToken();
  if (!token) return null;
  try {
    const base64 = token.split(".")[1];
    const json = atob(base64);
    return JSON.parse(json);
  } catch {
    return null;
  }
}

function setCookie(name: string, value: string) {
  if (typeof document === "undefined") return;
  const parts = [
    `${name}=${encodeURIComponent(value)}`,
    "Path=/",
    `Max-Age=${60 * 60 * 24 * 30}`,
    "SameSite=Lax",
  ];
  const domain = deriveCookieDomain(window.location.hostname);
  if (domain) {
    parts.push(`Domain=${domain}`);
  }
  if (window.location.protocol === "https:") {
    parts.push("Secure");
  }
  document.cookie = parts.join("; ");
}

function clearCookie(name: string) {
  if (typeof document === "undefined") return;
  const parts = [`${name}=`, "Path=/", "Max-Age=0", "SameSite=Lax"];
  const domain = deriveCookieDomain(window.location.hostname);
  if (domain) {
    parts.push(`Domain=${domain}`);
  }
  if (window.location.protocol === "https:") {
    parts.push("Secure");
  }
  document.cookie = parts.join("; ");
}

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

function dispatchAuthChanged() {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}

function deriveCookieDomain(hostname: string): string {
  if (!hostname || hostname === "localhost") return "";
  const parts = hostname.split(".");
  if (parts.length < 2) return "";
  return `.${parts.slice(-2).join(".")}`;
}
