import type { Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { getAccessToken, getRefreshToken, saveTokens, clearTokens } from "./auth";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";
import { createClient } from "@connectrpc/connect";

const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// Transport without interceptor for refresh calls (avoid infinite loop)
const plainTransport = createConnectTransport({
  baseUrl,
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

let refreshPromise: Promise<string | null> | null = null;

function isUnauthenticatedError(err: unknown): err is Error & { code: number } {
  return err instanceof Error && "code" in err && (err as { code: number }).code === 16;
}

async function refreshAccessToken(): Promise<string | null> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    const refreshToken = getRefreshToken();
    if (!refreshToken) {
      clearTokens();
      return null;
    }

    try {
      const refreshClient = createClient(AuthService, plainTransport);
      const res = await refreshClient.refreshToken({ refreshToken });
      saveTokens(res.accessToken, res.refreshToken);
      return res.accessToken;
    } catch {
      // Only clear when the token in storage is still the one that just failed.
      if (getRefreshToken() === refreshToken) {
        clearTokens();
      }
      return null;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

function isNotFoundError(err: unknown): boolean {
  return err instanceof Error && "code" in err && (err as { code: number }).code === 5;
}

const authInterceptor: Interceptor = (next) => async (req) => {
  const token = getAccessToken();
  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }

  try {
    return await next(req);
  } catch (err: unknown) {
    // If unauthorized, try refresh
    if (isUnauthenticatedError(err)) {
      const latestToken = getAccessToken();
      if (token && latestToken && latestToken !== token) {
        req.header.set("Authorization", `Bearer ${latestToken}`);
        return await next(req);
      }

      const refreshedToken = await refreshAccessToken();
      if (refreshedToken) {
        req.header.set("Authorization", `Bearer ${refreshedToken}`);
        return await next(req);
      }
    }
    // Redirect to projects list when resource not found
    if (isNotFoundError(err) && typeof window !== "undefined") {
      window.location.href = "/projects";
      return await next(req); // unreachable but satisfies return type
    }
    throw err;
  }
};

export const transport = createConnectTransport({
  baseUrl,
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
  interceptors: [authInterceptor],
});
