import { Mark, mergeAttributes } from "@tiptap/core";

export interface CommentHighlightOptions {
  HTMLAttributes: Record<string, string>;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    commentHighlight: {
      setCommentHighlight: (attrs: { commentId: string }) => ReturnType;
      unsetCommentHighlight: (commentId: string) => ReturnType;
    };
  }
}

export const CommentHighlight = Mark.create<CommentHighlightOptions>({
  name: "commentHighlight",

  addOptions() {
    return {
      HTMLAttributes: {},
    };
  },

  addAttributes() {
    return {
      commentId: {
        default: null,
        parseHTML: (element) => element.getAttribute("data-comment-id"),
        renderHTML: (attributes) => {
          if (!attributes.commentId) return {};
          return { "data-comment-id": attributes.commentId };
        },
      },
    };
  },

  parseHTML() {
    return [{ tag: "span[data-comment-id]" }];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "span",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        class: "comment-highlight",
      }),
      0,
    ];
  },

  addCommands() {
    return {
      setCommentHighlight:
        (attrs) =>
        ({ commands }) => {
          return commands.setMark(this.name, attrs);
        },
      unsetCommentHighlight:
        (commentId) =>
        ({ tr, state }) => {
          const { doc } = state;
          doc.descendants((node, pos) => {
            node.marks.forEach((mark) => {
              if (
                mark.type.name === this.name &&
                mark.attrs.commentId === commentId
              ) {
                tr.removeMark(pos, pos + node.nodeSize, mark);
              }
            });
          });
          return true;
        },
    };
  },
});
