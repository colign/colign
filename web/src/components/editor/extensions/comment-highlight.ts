import { createStyleSpec } from "@blocknote/core";

/**
 * BlockNote custom style for inline comment highlights.
 * The string value is the commentId.
 */
export const CommentHighlight = createStyleSpec(
  {
    type: "commentHighlight",
    propSchema: "string",
  },
  {
    render: (value) => {
      const span = document.createElement("span");
      span.classList.add("comment-highlight");
      span.setAttribute("data-comment-id", value);
      return { dom: span, contentDOM: span };
    },
    parse: (element) => {
      const commentId = element.getAttribute("data-comment-id");
      if (commentId) return commentId;
      if (element.classList.contains("comment-highlight")) return "";
      return undefined;
    },
  },
);
