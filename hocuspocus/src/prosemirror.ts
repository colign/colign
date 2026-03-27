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
