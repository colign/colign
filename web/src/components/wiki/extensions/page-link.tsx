import { createInlineContentSpec } from "@blocknote/core";

/**
 * BlockNote custom inline content for wiki page links.
 * Renders as an atomic [[Page Title]] link that navigates to the target page.
 */
export const PageLink = createInlineContentSpec(
  {
    type: "pageLink" as const,
    propSchema: {
      pageId: { default: "" },
      pageTitle: { default: "" },
    },
    content: "none",
  },
  {
    render: (inlineContent) => {
      const dom = document.createElement("span");
      dom.classList.add("wiki-page-link");
      dom.setAttribute("data-page-id", inlineContent.props.pageId);
      dom.setAttribute("data-page-title", inlineContent.props.pageTitle);
      dom.textContent = inlineContent.props.pageTitle || "Untitled";
      dom.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        window.dispatchEvent(
          new CustomEvent("wiki-navigate", {
            detail: { pageId: inlineContent.props.pageId },
          }),
        );
      });
      return { dom };
    },
    parse: (element) => {
      const pageId = element.getAttribute("data-page-id");
      if (pageId && element.classList.contains("wiki-page-link")) {
        return {
          pageId,
          pageTitle: element.getAttribute("data-page-title") || element.textContent || "",
        };
      }
      return undefined;
    },
    toExternalHTML: (inlineContent) => {
      const dom = document.createElement("span");
      dom.classList.add("wiki-page-link");
      dom.setAttribute("data-page-id", inlineContent.props.pageId);
      dom.setAttribute("data-page-title", inlineContent.props.pageTitle);
      dom.textContent = inlineContent.props.pageTitle || "Untitled";
      return { dom };
    },
  },
);
