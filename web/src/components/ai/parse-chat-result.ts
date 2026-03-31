import type { ChatProposalResult, ChatACResult, ChatMessage } from "./types";

/**
 * Extracts a structured result (proposal or AC) from the AI's text response.
 * The AI outputs JSON inside ```json ... ``` code blocks.
 * Returns the result and the content with the code block removed.
 */
export function parseChatResult(content: string): {
  result: ChatMessage["result"] | undefined;
  cleanContent: string;
} {
  // Match ```json ... ``` blocks
  const codeBlockRegex = /```json\s*\n([\s\S]*?)\n```/;
  const match = content.match(codeBlockRegex);

  if (!match) {
    return { result: undefined, cleanContent: content };
  }

  const jsonStr = match[1].trim();
  const cleanContent = content.replace(match[0], "").trim();

  try {
    const parsed = JSON.parse(jsonStr);

    // Check if it's a proposal (object with "problem" key)
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed) && "problem" in parsed) {
      const proposal: ChatProposalResult = {
        problem: parsed.problem ?? "",
        scope: parsed.scope ?? "",
        outOfScope: parsed.outOfScope ?? "",
      };
      return { result: proposal, cleanContent };
    }

    // Check if it's AC (array with "scenario" key)
    if (Array.isArray(parsed) && parsed.length > 0 && "scenario" in parsed[0]) {
      const acs: ChatACResult[] = parsed.map((item: Record<string, unknown>) => ({
        scenario: (item.scenario as string) ?? "",
        steps: Array.isArray(item.steps)
          ? item.steps.map((s: Record<string, unknown>) => ({
              keyword: (s.keyword as string) ?? "",
              text: (s.text as string) ?? "",
            }))
          : [],
      }));
      return { result: acs, cleanContent };
    }
  } catch {
    // JSON parse failed — return as-is
  }

  return { result: undefined, cleanContent: content };
}
