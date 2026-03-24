import { getAccessToken, getRefreshToken, saveTokens, clearTokens } from "./auth";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface SectionChunk {
  section: string;
  chunk: string;
}

export interface GeneratedAC {
  scenario: string;
  steps: { keyword: string; text: string }[];
}

let refreshPromise: Promise<string | null> | null = null;

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
      const plainTransport = createConnectTransport({
        baseUrl: API_BASE,
        fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
      });
      const refreshClient = createClient(AuthService, plainTransport);
      const res = await refreshClient.refreshToken({ refreshToken });
      saveTokens(res.accessToken, res.refreshToken);
      return res.accessToken;
    } catch {
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

// Helper: fetch with automatic token refresh on 401
async function fetchWithAuth(
  url: string,
  init: RequestInit
): Promise<Response> {
  const token = getAccessToken();
  const headers = new Headers(init.headers);

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  let res = await fetch(url, {
    ...init,
    headers,
    credentials: "include",
  });

  // If unauthorized, try refresh
  if (res.status === 401) {
    const latestToken = getAccessToken();
    if (token && latestToken && latestToken !== token) {
      headers.set("Authorization", `Bearer ${latestToken}`);
      res = await fetch(url, {
        ...init,
        headers,
        credentials: "include",
      });
    } else {
      const refreshedToken = await refreshAccessToken();
      if (refreshedToken) {
        headers.set("Authorization", `Bearer ${refreshedToken}`);
        res = await fetch(url, {
          ...init,
          headers,
          credentials: "include",
        });
      }
    }
  }

  return res;
}

// Streaming proposal generation via SSE
export async function* streamProposal(
  changeId: number | bigint,
  description: string,
  signal?: AbortSignal
): AsyncGenerator<SectionChunk> {
  const res = await fetchWithAuth(`${API_BASE}/api/ai/generate-proposal`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ changeId: Number(changeId), description }),
    signal,
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || "Failed to generate proposal");
  }

  const reader = res.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";

    for (const line of lines) {
      if (line.startsWith("data: ")) {
        const data = line.slice(6);
        if (data === "[DONE]") return;
        try {
          yield JSON.parse(data) as SectionChunk;
        } catch {
          // skip malformed lines
        }
      }
    }
  }
}

// Batch AC generation
export async function generateAC(
  changeId: number | bigint
): Promise<GeneratedAC[]> {
  const res = await fetchWithAuth(`${API_BASE}/api/ai/generate-ac`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ changeId: Number(changeId) }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || "Failed to generate AC");
  }

  return res.json();
}
