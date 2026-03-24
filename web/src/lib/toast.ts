import { toast } from "sonner";

/**
 * Extract a user-friendly message from ConnectRPC or generic errors.
 */
function extractMessage(err: unknown): string {
  if (err instanceof Error) {
    // ConnectRPC errors have a `message` field with useful info
    return err.message;
  }
  if (typeof err === "string") return err;
  return "";
}

export function showError(fallbackMessage: string, err?: unknown) {
  const detail = err ? extractMessage(err) : "";
  toast.error(fallbackMessage, {
    description: detail || undefined,
  });
}

export function showSuccess(message: string, description?: string) {
  toast.success(message, {
    description,
  });
}

export function showInfo(message: string, description?: string) {
  toast.info(message, {
    description,
  });
}
