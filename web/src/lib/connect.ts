import type { Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { getAccessToken, getRefreshToken, saveTokens, clearTokens } from "./auth";
import { AuthService } from "@/gen/proto/auth/v1/auth_pb";
import { createClient } from "@connectrpc/connect";

const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// Transport without interceptor for refresh calls (avoid infinite loop)
const plainTransport = createConnectTransport({ baseUrl });

const authInterceptor: Interceptor = (next) => async (req) => {
  const token = getAccessToken();
  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }

  try {
    return await next(req);
  } catch (err: unknown) {
    // If unauthorized, try refresh
    if (
      err instanceof Error &&
      "code" in err &&
      (err as { code: number }).code === 16 // Unauthenticated
    ) {
      const refreshToken = getRefreshToken();
      if (refreshToken) {
        try {
          const refreshClient = createClient(AuthService, plainTransport);
          const res = await refreshClient.refreshToken({ refreshToken });
          saveTokens(res.accessToken, res.refreshToken);

          // Retry original request with new token
          req.header.set("Authorization", `Bearer ${res.accessToken}`);
          return await next(req);
        } catch {
          clearTokens();
        }
      } else {
        clearTokens();
      }
    }
    throw err;
  }
};

export const transport = createConnectTransport({
  baseUrl,
  interceptors: [authInterceptor],
});
