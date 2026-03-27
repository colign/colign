import * as Y from "yjs";

/**
 * Converts simple HTML (from TipTap/markdownToHTML) into Y.js XmlFragment nodes.
 * Handles: headings, paragraphs, lists, code blocks, blockquotes, tables, and inline marks.
 */
export function htmlToYXmlFragment(
  doc: Y.Doc,
  fragment: Y.XmlFragment,
  html: string,
): void {
  const tokens = tokenize(html);
  let i = 0;

  // Merge adjacent list items into a single list
  while (i < tokens.length) {
    const token = tokens[i];

    if (token.type === "heading") {
      const el = new Y.XmlElement("heading");
      el.setAttribute("level", String(token.level));
      const text = new Y.XmlText();
      applyInlineFormatting(text, token.content);
      el.insert(0, [text]);
      fragment.insert(fragment.length, [el]);
      i++;
    } else if (token.type === "paragraph") {
      const el = new Y.XmlElement("paragraph");
      const text = new Y.XmlText();
      applyInlineFormatting(text, token.content);
      el.insert(0, [text]);
      fragment.insert(fragment.length, [el]);
      i++;
    } else if (token.type === "codeblock") {
      const el = new Y.XmlElement("codeBlock");
      const text = new Y.XmlText();
      text.insert(0, unescapeHtml(token.content));
      el.insert(0, [text]);
      fragment.insert(fragment.length, [el]);
      i++;
    } else if (token.type === "listitem") {
      // Collect consecutive list items of the same list type
      const listType = token.listType;
      const listEl = new Y.XmlElement(
        listType === "ol" ? "orderedList" : "bulletList",
      );

      while (i < tokens.length && tokens[i].type === "listitem" && tokens[i].listType === listType) {
        const li = new Y.XmlElement("listItem");
        const p = new Y.XmlElement("paragraph");
        const text = new Y.XmlText();
        applyInlineFormatting(text, tokens[i].content);
        p.insert(0, [text]);
        li.insert(0, [p]);
        listEl.insert(listEl.length, [li]);
        i++;
      }

      fragment.insert(fragment.length, [listEl]);
    } else if (token.type === "blockquote") {
      const el = new Y.XmlElement("blockquote");
      // Parse inner content (may contain <p> tags)
      const innerParagraphs = token.content.match(/<p[^>]*>([\s\S]*?)<\/p>/gi);
      if (innerParagraphs) {
        for (const pMatch of innerParagraphs) {
          const pContent = pMatch.replace(/^<p[^>]*>/i, "").replace(/<\/p>$/i, "");
          const p = new Y.XmlElement("paragraph");
          const text = new Y.XmlText();
          applyInlineFormatting(text, pContent);
          p.insert(0, [text]);
          el.insert(el.length, [p]);
        }
      } else {
        // No <p> wrapper — treat as single paragraph
        const p = new Y.XmlElement("paragraph");
        const text = new Y.XmlText();
        applyInlineFormatting(text, token.content);
        p.insert(0, [text]);
        el.insert(el.length, [p]);
      }
      fragment.insert(fragment.length, [el]);
      i++;
    } else if (token.type === "table") {
      const table = new Y.XmlElement("table");

      const headerRows = token.headerRows ?? [];
      const bodyRows = token.bodyRows ?? [];

      if (headerRows.length > 0) {
        for (const row of headerRows) {
          table.insert(table.length, [createTableRow(row, true)]);
        }
      }

      for (const row of bodyRows) {
        table.insert(table.length, [createTableRow(row, false)]);
      }

      fragment.insert(fragment.length, [table]);
      i++;
    } else {
      i++;
    }
  }
}

interface Token {
  type: "heading" | "paragraph" | "listitem" | "codeblock" | "blockquote" | "table";
  content: string;
  level?: number;
  listType?: "ul" | "ol";
  headerRows?: string[][];
  bodyRows?: string[][];
}

function tokenize(html: string): Token[] {
  const tokens: Token[] = [];
  // Match top-level HTML elements
  const tagRegex = /<(h[1-6]|p|ul|ol|pre|blockquote|table)([^>]*)>([\s\S]*?)<\/\1>/gi;
  let match: RegExpExecArray | null;

  while ((match = tagRegex.exec(html)) !== null) {
    const tag = match[1].toLowerCase();
    const content = match[3];

    if (tag.startsWith("h")) {
      const level = parseInt(tag[1], 10);
      tokens.push({
        type: "heading",
        content: content,
        level,
      });
    } else if (tag === "p") {
      tokens.push({
        type: "paragraph",
        content: content,
      });
    } else if (tag === "ul" || tag === "ol") {
      // Extract <li> items
      const liRegex = /<li>([\s\S]*?)<\/li>/gi;
      let liMatch: RegExpExecArray | null;
      while ((liMatch = liRegex.exec(content)) !== null) {
        tokens.push({
          type: "listitem",
          content: stripWrappingParagraph(liMatch[1]),
          listType: tag as "ul" | "ol",
        });
      }
    } else if (tag === "pre") {
      const codeMatch = content.match(/<code[^>]*>([\s\S]*?)<\/code>/i);
      tokens.push({
        type: "codeblock",
        content: codeMatch ? codeMatch[1] : stripTags(content),
      });
    } else if (tag === "blockquote") {
      tokens.push({
        type: "blockquote",
        content: content,
      });
    } else if (tag === "table") {
      tokens.push({
        type: "table",
        content,
        ...parseTable(content),
      });
    }
  }

  return tokens;
}

function parseTable(html: string): { headerRows: string[][]; bodyRows: string[][] } {
  const headerRows: string[][] = [];
  const bodyRows: string[][] = [];
  const rowRegex = /<tr[^>]*>([\s\S]*?)<\/tr>/gi;
  let rowMatch: RegExpExecArray | null;

  while ((rowMatch = rowRegex.exec(html)) !== null) {
    const rowHtml = rowMatch[1];
    const headerCells = extractCells(rowHtml, "th");
    if (headerCells.length > 0) {
      headerRows.push(headerCells);
      continue;
    }

    const bodyCells = extractCells(rowHtml, "td");
    if (bodyCells.length > 0) {
      bodyRows.push(bodyCells);
    }
  }

  return { headerRows, bodyRows };
}

function extractCells(rowHtml: string, tagName: "th" | "td"): string[] {
  const cells: string[] = [];
  const cellRegex = new RegExp(`<${tagName}[^>]*>([\\s\\S]*?)<\\/${tagName}>`, "gi");
  let cellMatch: RegExpExecArray | null;

  while ((cellMatch = cellRegex.exec(rowHtml)) !== null) {
    cells.push(cellMatch[1].trim());
  }

  return cells;
}

function createTableRow(cells: string[], isHeader: boolean): Y.XmlElement {
  const row = new Y.XmlElement("tableRow");

  for (const cellHtml of cells) {
    const cell = new Y.XmlElement(isHeader ? "tableHeader" : "tableCell");
    const paragraph = new Y.XmlElement("paragraph");
    const text = new Y.XmlText();
    applyInlineFormatting(text, stripWrappingParagraph(cellHtml));
    paragraph.insert(0, [text]);
    cell.insert(0, [paragraph]);
    row.insert(row.length, [cell]);
  }

  return row;
}

function stripTags(html: string): string {
  return html.replace(/<[^>]+>/g, "");
}

function stripWrappingParagraph(html: string): string {
  return html
    .replace(/^<p[^>]*>/i, "")
    .replace(/<\/p>$/i, "")
    .trim();
}

/**
 * Applies inline formatting (bold, italic, code) from simple HTML to Y.XmlText.
 * For plain text without formatting, just inserts the text directly.
 */
function applyInlineFormatting(xmlText: Y.XmlText, html: string): void {
  // Simple case: no inline formatting tags
  if (!/<(strong|em|code|b|i|u|s)[\s>]/i.test(html)) {
    xmlText.insert(0, unescapeHtml(html));
    return;
  }

  // Parse inline formatting
  const segments = parseInlineHtml(html);
  let offset = 0;
  for (const seg of segments) {
    const attrs: Record<string, boolean> = {};
    if (seg.bold) attrs.bold = true;
    if (seg.italic) attrs.italic = true;
    if (seg.code) attrs.code = true;
    if (seg.underline) attrs.underline = true;
    if (seg.strike) attrs.strike = true;

    const text = unescapeHtml(seg.text);
    xmlText.insert(offset, text, attrs);
    offset += text.length;
  }
}

interface InlineSegment {
  text: string;
  bold?: boolean;
  italic?: boolean;
  code?: boolean;
  underline?: boolean;
  strike?: boolean;
}

function parseInlineHtml(html: string): InlineSegment[] {
  const segments: InlineSegment[] = [];
  let remaining = html;

  while (remaining.length > 0) {
    // Find the next inline tag
    const tagMatch = remaining.match(
      /<(strong|em|code|b|i|u|s)>([\s\S]*?)<\/\1>/i,
    );

    if (!tagMatch || tagMatch.index === undefined) {
      // No more tags, rest is plain text
      if (remaining) {
        segments.push({ text: stripTags(remaining) });
      }
      break;
    }

    // Add text before the tag
    if (tagMatch.index > 0) {
      const before = remaining.substring(0, tagMatch.index);
      if (before) {
        segments.push({ text: stripTags(before) });
      }
    }

    const tag = tagMatch[1].toLowerCase();
    const content = tagMatch[2];
    const seg: InlineSegment = { text: stripTags(content) };

    if (tag === "strong" || tag === "b") seg.bold = true;
    if (tag === "em" || tag === "i") seg.italic = true;
    if (tag === "code") seg.code = true;
    if (tag === "u") seg.underline = true;
    if (tag === "s") seg.strike = true;

    segments.push(seg);
    remaining = remaining.substring(tagMatch.index + tagMatch[0].length);
  }

  return segments;
}

function unescapeHtml(text: string): string {
  return text
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'");
}
