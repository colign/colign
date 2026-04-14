import * as Y from "yjs";

type PMMark = {
  type: string;
  attrs?: Record<string, unknown>;
};

export type PMNode = {
  type: string;
  attrs?: Record<string, unknown>;
  text?: string;
  marks?: PMMark[];
  content?: PMNode[];
};

export function isProseMirrorJSONContent(content: string): boolean {
  try {
    const parsed = JSON.parse(content) as PMNode;
    return parsed?.type === "doc";
  } catch {
    return false;
  }
}

export function proseMirrorJSONToYXmlFragment(
  doc: Y.Doc,
  fragment: Y.XmlFragment,
  content: PMNode,
): void {
  if (content.type !== "doc") {
    throw new Error(`expected ProseMirror doc node, got ${content.type}`);
  }

  for (const child of content.content ?? []) {
    const yNode = proseMirrorNodeToYNode(child);
    if (yNode) {
      fragment.insert(fragment.length, [yNode]);
    }
  }
}

export function yXmlFragmentToProseMirrorJSON(fragment: Y.XmlFragment): PMNode {
  const content: PMNode[] = [];

  fragment.forEach((item) => {
    const node = yNodeToProseMirror(item);
    if (node) {
      if (Array.isArray(node)) {
        content.push(...node);
      } else {
        content.push(node);
      }
    }
  });

  return { type: "doc", content };
}

function proseMirrorNodeToYNode(node: PMNode): Y.XmlElement | Y.XmlText | null {
  if (node.type === "text") {
    const text = new Y.XmlText();
    text.insert(0, node.text ?? "", marksToYAttrs(node.marks));
    return text;
  }

  if (node.type === "hardBreak") {
    return new Y.XmlElement("hardBreak");
  }

  if (node.type === "horizontalRule") {
    return new Y.XmlElement("horizontalRule");
  }

  // PageLink is an atomic inline content node
  if (node.type === "pageLink") {
    const el = new Y.XmlElement("pageLink");
    if (node.attrs) {
      for (const [key, value] of Object.entries(node.attrs)) {
        if (value !== undefined && value !== null) {
          el.setAttribute(key, String(value));
        }
      }
    }
    return el;
  }

  const element = new Y.XmlElement(proseMirrorTypeToYNode(node.type));
  for (const [key, value] of Object.entries(node.attrs ?? {})) {
    if (value !== undefined && value !== null) {
      element.setAttribute(key, String(value));
    }
  }

  for (const child of node.content ?? []) {
    const yChild = proseMirrorNodeToYNode(child);
    if (yChild) {
      element.insert(element.length, [yChild]);
    }
  }

  return element;
}

function yNodeToProseMirror(node: Y.XmlElement | Y.XmlText): PMNode | PMNode[] | null {
  if (node instanceof Y.XmlText) {
    return yTextToProseMirror(node);
  }

  // PageLink is an atomic inline content node
  if (node.nodeName === "pageLink") {
    const attrs = node.getAttributes();
    return { type: "pageLink", attrs };
  }

  const type = yNodeToProseMirrorType(node.nodeName);
  if (!type) {
    return null;
  }

  const attrs = node.getAttributes();
  const content: PMNode[] = [];
  node.forEach((child) => {
    const converted = yNodeToProseMirror(child as Y.XmlElement | Y.XmlText);
    if (!converted) return;
    if (Array.isArray(converted)) {
      content.push(...converted);
    } else {
      content.push(converted);
    }
  });

  const pmNode: PMNode = { type };
  if (Object.keys(attrs).length > 0) {
    pmNode.attrs = attrs;
  }
  if (content.length > 0) {
    pmNode.content = content;
  }
  return pmNode;
}

function yTextToProseMirror(text: Y.XmlText): PMNode[] {
  const nodes: PMNode[] = [];

  for (const op of text.toDelta()) {
    if (typeof op.insert !== "string" || op.insert.length === 0) {
      continue;
    }

    if (op.insert === "\n") {
      nodes.push({ type: "hardBreak" });
      continue;
    }

    const node: PMNode = { type: "text", text: op.insert };
    const marks = yAttrsToMarks(op.attributes as Record<string, unknown> | undefined);
    if (marks.length > 0) {
      node.marks = marks;
    }
    nodes.push(node);
  }

  return nodes;
}

function proseMirrorTypeToYNode(type: string): string {
  switch (type) {
    case "paragraph":
      return "paragraph";
    case "heading":
      return "heading";
    case "bulletList":
      return "bulletList";
    case "orderedList":
      return "orderedList";
    case "listItem":
      return "listItem";
    case "blockquote":
      return "blockquote";
    case "table":
      return "table";
    case "tableRow":
      return "tableRow";
    case "tableHeader":
      return "tableHeader";
    case "tableCell":
      return "tableCell";
    case "tableParagraph":
      return "tableParagraph";
    case "codeBlock":
      return "codeBlock";
    default:
      return type;
  }
}

function yNodeToProseMirrorType(type: string): string | null {
  switch (type) {
    case "paragraph":
      return "paragraph";
    case "heading":
      return "heading";
    case "bulletList":
      return "bulletList";
    case "orderedList":
      return "orderedList";
    case "listItem":
      return "listItem";
    case "blockquote":
      return "blockquote";
    case "table":
      return "table";
    case "tableRow":
      return "tableRow";
    case "tableHeader":
      return "tableHeader";
    case "tableCell":
      return "tableCell";
    case "tableParagraph":
      return "tableParagraph";
    case "codeBlock":
      return "codeBlock";
    case "hardBreak":
      return "hardBreak";
    case "horizontalRule":
      return "horizontalRule";
    default:
      return null;
  }
}

function marksToYAttrs(marks: PMMark[] | undefined): Record<string, unknown> {
  const attrs: Record<string, unknown> = {};

  for (const mark of marks ?? []) {
    switch (mark.type) {
      case "bold":
        attrs.bold = true;
        break;
      case "italic":
        attrs.italic = true;
        break;
      case "code":
        attrs.code = true;
        break;
      case "underline":
        attrs.underline = true;
        break;
      case "strike":
        attrs.strike = true;
        break;
      case "commentHighlight":
        attrs.commentHighlight = mark.attrs ?? {};
        break;
      default:
        break;
    }
  }

  return attrs;
}

function yAttrsToMarks(attrs: Record<string, unknown> | undefined): PMMark[] {
  if (!attrs) {
    return [];
  }

  const marks: PMMark[] = [];
  if (attrs.bold) marks.push({ type: "bold" });
  if (attrs.italic) marks.push({ type: "italic" });
  if (attrs.code) marks.push({ type: "code" });
  if (attrs.underline) marks.push({ type: "underline" });
  if (attrs.strike) marks.push({ type: "strike" });
  if (attrs.commentHighlight) {
    marks.push({ type: "commentHighlight", attrs: attrs.commentHighlight as Record<string, unknown> });
  }
  return marks;
}

// ---------------------------------------------------------------------------
// BlockNote ↔ ProseMirror JSON conversion
// ---------------------------------------------------------------------------
// BlockNote stores: blockGroup → blockContainer(id) → content node
// ProseMirror stores: flat doc → heading/paragraph/bulletList/orderedList/...

let bnIdCounter = 0;
function nextBnId(): string {
  return `bn-${Date.now().toString(36)}-${(bnIdCounter++).toString(36)}`;
}

/**
 * Check whether the Y.js fragment contains BlockNote structure
 * (first child is a blockGroup element).
 */
export function isBlockNoteFragment(fragment: Y.XmlFragment): boolean {
  if (fragment.length === 0) return false;
  const first = fragment.get(0);
  return first instanceof Y.XmlElement && first.nodeName === "blockGroup";
}

/**
 * Convert ProseMirror JSON → BlockNote Y.js structure.
 * Wraps each PM node in blockGroup → blockContainer.
 */
export function proseMirrorJSONToBlockNoteYFragment(
  doc: Y.Doc,
  fragment: Y.XmlFragment,
  content: PMNode,
): void {
  if (content.type !== "doc") {
    throw new Error(`expected ProseMirror doc node, got ${content.type}`);
  }

  const blockGroup = new Y.XmlElement("blockGroup");

  for (const child of content.content ?? []) {
    pmNodeToBlockNote(blockGroup, child);
  }

  fragment.insert(fragment.length, [blockGroup]);
}

function pmNodeToBlockNote(blockGroup: Y.XmlElement, node: PMNode): void {
  switch (node.type) {
    case "heading":
    case "paragraph":
    case "codeBlock": {
      const container = makeBnContainer();
      const yNode = proseMirrorNodeToYNode(node);
      if (yNode) {
        container.insert(0, [yNode]);
        blockGroup.insert(blockGroup.length, [container]);
      }
      break;
    }
    case "table": {
      const container = makeBnContainer();
      // Convert table with tableParagraph inside cells
      const tableEl = pmTableToBlockNote(node);
      container.insert(0, [tableEl]);
      blockGroup.insert(blockGroup.length, [container]);
      break;
    }
    case "bulletList": {
      for (const item of node.content ?? []) {
        const container = makeBnContainer();
        const li = new Y.XmlElement("bulletListItem");
        copyListItemContent(item, li);
        container.insert(0, [li]);
        blockGroup.insert(blockGroup.length, [container]);
      }
      break;
    }
    case "orderedList": {
      for (const item of node.content ?? []) {
        const container = makeBnContainer();
        const li = new Y.XmlElement("numberedListItem");
        copyListItemContent(item, li);
        container.insert(0, [li]);
        blockGroup.insert(blockGroup.length, [container]);
      }
      break;
    }
    case "blockquote": {
      for (const child of node.content ?? []) {
        if (child.type === "paragraph") {
          const container = makeBnContainer();
          const quote = new Y.XmlElement("quote");
          for (const textNode of child.content ?? []) {
            const yChild = proseMirrorNodeToYNode(textNode);
            if (yChild) quote.insert(quote.length, [yChild]);
          }
          container.insert(0, [quote]);
          blockGroup.insert(blockGroup.length, [container]);
        }
      }
      break;
    }
    case "horizontalRule": {
      // Skip — BlockNote doesn't support horizontal rules as blocks
      break;
    }
    default: {
      // Fallback: wrap as-is
      const container = makeBnContainer();
      const yNode = proseMirrorNodeToYNode(node);
      if (yNode) {
        container.insert(0, [yNode]);
        blockGroup.insert(blockGroup.length, [container]);
      }
      break;
    }
  }
}

function makeBnContainer(): Y.XmlElement {
  const container = new Y.XmlElement("blockContainer");
  container.setAttribute("id", nextBnId());
  return container;
}

/** Copy listItem's paragraph children's text content into a BlockNote list item element */
function copyListItemContent(listItem: PMNode, target: Y.XmlElement): void {
  for (const child of listItem.content ?? []) {
    if (child.type === "paragraph") {
      for (const textNode of child.content ?? []) {
        const yChild = proseMirrorNodeToYNode(textNode);
        if (yChild) target.insert(target.length, [yChild]);
      }
    }
  }
}

/** Convert PM table node to BlockNote table with tableParagraph inside cells */
function pmTableToBlockNote(table: PMNode): Y.XmlElement {
  const tableEl = new Y.XmlElement("table");
  for (const [key, value] of Object.entries(table.attrs ?? {})) {
    if (value !== undefined && value !== null) {
      tableEl.setAttribute(key, String(value));
    }
  }

  for (const row of table.content ?? []) {
    const rowEl = new Y.XmlElement("tableRow");
    for (const cell of row.content ?? []) {
      const cellEl = new Y.XmlElement(
        cell.type === "tableHeader" ? "tableHeader" : "tableCell",
      );
      for (const [key, value] of Object.entries(cell.attrs ?? {})) {
        if (value !== undefined && value !== null) {
          cellEl.setAttribute(key, String(value));
        }
      }
      // Convert paragraph → tableParagraph inside cells
      for (const cellChild of cell.content ?? []) {
        if (cellChild.type === "paragraph" || cellChild.type === "tableParagraph") {
          const tp = new Y.XmlElement("tableParagraph");
          for (const textNode of cellChild.content ?? []) {
            const yChild = proseMirrorNodeToYNode(textNode);
            if (yChild) tp.insert(tp.length, [yChild]);
          }
          cellEl.insert(cellEl.length, [tp]);
        }
      }
      rowEl.insert(rowEl.length, [cellEl]);
    }
    tableEl.insert(tableEl.length, [rowEl]);
  }

  return tableEl;
}

/**
 * Convert BlockNote Y.js structure → ProseMirror JSON.
 * Unwraps blockGroup/blockContainer and groups consecutive list items.
 */
export function yBlockNoteFragmentToProseMirrorJSON(fragment: Y.XmlFragment): PMNode {
  const content: PMNode[] = [];

  fragment.forEach((item) => {
    if (item instanceof Y.XmlElement && item.nodeName === "blockGroup") {
      content.push(...extractBlocksFromGroup(item));
    } else {
      // Fallback for non-BlockNote nodes
      const node = yNodeToProseMirror(item);
      if (node) {
        if (Array.isArray(node)) content.push(...node);
        else content.push(node);
      }
    }
  });

  return { type: "doc", content };
}

function extractBlocksFromGroup(blockGroup: Y.XmlElement): PMNode[] {
  const result: PMNode[] = [];
  let currentListType: string | null = null;
  let currentListItems: PMNode[] = [];

  const flushList = () => {
    if (currentListType && currentListItems.length > 0) {
      result.push({ type: currentListType, content: currentListItems });
      currentListItems = [];
      currentListType = null;
    }
  };

  blockGroup.forEach((container) => {
    if (!(container instanceof Y.XmlElement) || container.nodeName !== "blockContainer") return;

    const contentNode = container.get(0) as Y.XmlElement | undefined;
    if (!contentNode || !(contentNode instanceof Y.XmlElement)) return;

    const nodeName = contentNode.nodeName;

    if (nodeName === "bulletListItem" || nodeName === "numberedListItem") {
      const listType = nodeName === "bulletListItem" ? "bulletList" : "orderedList";
      if (currentListType !== listType) {
        flushList();
        currentListType = listType;
      }
      const textContent = extractChildContent(contentNode);
      currentListItems.push({
        type: "listItem",
        content: [{ type: "paragraph", content: textContent }],
      });
    } else {
      flushList();

      if (nodeName === "quote") {
        const textContent = extractChildContent(contentNode);
        result.push({
          type: "blockquote",
          content: [{ type: "paragraph", content: textContent }],
        });
      } else if (nodeName === "table") {
        result.push(bnTableToProseMirror(contentNode));
      } else {
        // heading, paragraph, codeBlock — pass through
        const node = yNodeToProseMirror(contentNode);
        if (node) {
          if (Array.isArray(node)) result.push(...node);
          else result.push(node);
        }
      }
    }
  });

  flushList();
  return result;
}

function extractChildContent(node: Y.XmlElement): PMNode[] {
  const content: PMNode[] = [];
  node.forEach((child) => {
    const converted = yNodeToProseMirror(child as Y.XmlElement | Y.XmlText);
    if (converted) {
      if (Array.isArray(converted)) content.push(...converted);
      else content.push(converted);
    }
  });
  return content;
}

/** Convert BlockNote table Y.js node to ProseMirror JSON (tableParagraph → paragraph) */
function bnTableToProseMirror(tableNode: Y.XmlElement): PMNode {
  const rows: PMNode[] = [];
  tableNode.forEach((row) => {
    if (!(row instanceof Y.XmlElement) || row.nodeName !== "tableRow") return;
    const cells: PMNode[] = [];
    row.forEach((cell) => {
      if (!(cell instanceof Y.XmlElement)) return;
      const cellType = cell.nodeName === "tableHeader" ? "tableHeader" : "tableCell";
      const cellAttrs = cell.getAttributes();
      const cellContent: PMNode[] = [];
      cell.forEach((child) => {
        if (child instanceof Y.XmlElement && (child.nodeName === "tableParagraph" || child.nodeName === "paragraph")) {
          const textContent: PMNode[] = [];
          child.forEach((textChild) => {
            const converted = yNodeToProseMirror(textChild as Y.XmlElement | Y.XmlText);
            if (converted) {
              if (Array.isArray(converted)) textContent.push(...converted);
              else textContent.push(converted);
            }
          });
          cellContent.push({ type: "paragraph", content: textContent });
        }
      });
      const pmCell: PMNode = { type: cellType };
      if (Object.keys(cellAttrs).length > 0) pmCell.attrs = cellAttrs;
      if (cellContent.length > 0) pmCell.content = cellContent;
      cells.push(pmCell);
    });
    rows.push({ type: "tableRow", content: cells });
  });

  const attrs = tableNode.getAttributes();
  const pmTable: PMNode = { type: "table", content: rows };
  if (Object.keys(attrs).length > 0) pmTable.attrs = attrs;
  return pmTable;
}
