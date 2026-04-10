"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.isProseMirrorJSONContent = isProseMirrorJSONContent;
exports.proseMirrorJSONToYXmlFragment = proseMirrorJSONToYXmlFragment;
exports.yXmlFragmentToProseMirrorJSON = yXmlFragmentToProseMirrorJSON;
exports.isBlockNoteFragment = isBlockNoteFragment;
exports.proseMirrorJSONToBlockNoteYFragment = proseMirrorJSONToBlockNoteYFragment;
exports.yBlockNoteFragmentToProseMirrorJSON = yBlockNoteFragmentToProseMirrorJSON;
const Y = __importStar(require("yjs"));
function isProseMirrorJSONContent(content) {
    try {
        const parsed = JSON.parse(content);
        return parsed?.type === "doc";
    }
    catch {
        return false;
    }
}
function proseMirrorJSONToYXmlFragment(doc, fragment, content) {
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
function yXmlFragmentToProseMirrorJSON(fragment) {
    const content = [];
    fragment.forEach((item) => {
        const node = yNodeToProseMirror(item);
        if (node) {
            if (Array.isArray(node)) {
                content.push(...node);
            }
            else {
                content.push(node);
            }
        }
    });
    return { type: "doc", content };
}
function proseMirrorNodeToYNode(node) {
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
function yNodeToProseMirror(node) {
    if (node instanceof Y.XmlText) {
        return yTextToProseMirror(node);
    }
    const type = yNodeToProseMirrorType(node.nodeName);
    if (!type) {
        return null;
    }
    const attrs = node.getAttributes();
    const content = [];
    node.forEach((child) => {
        const converted = yNodeToProseMirror(child);
        if (!converted)
            return;
        if (Array.isArray(converted)) {
            content.push(...converted);
        }
        else {
            content.push(converted);
        }
    });
    const pmNode = { type };
    if (Object.keys(attrs).length > 0) {
        pmNode.attrs = attrs;
    }
    if (content.length > 0) {
        pmNode.content = content;
    }
    return pmNode;
}
function yTextToProseMirror(text) {
    const nodes = [];
    for (const op of text.toDelta()) {
        if (typeof op.insert !== "string" || op.insert.length === 0) {
            continue;
        }
        if (op.insert === "\n") {
            nodes.push({ type: "hardBreak" });
            continue;
        }
        const node = { type: "text", text: op.insert };
        const marks = yAttrsToMarks(op.attributes);
        if (marks.length > 0) {
            node.marks = marks;
        }
        nodes.push(node);
    }
    return nodes;
}
function proseMirrorTypeToYNode(type) {
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
function yNodeToProseMirrorType(type) {
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
function marksToYAttrs(marks) {
    const attrs = {};
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
function yAttrsToMarks(attrs) {
    if (!attrs) {
        return [];
    }
    const marks = [];
    if (attrs.bold)
        marks.push({ type: "bold" });
    if (attrs.italic)
        marks.push({ type: "italic" });
    if (attrs.code)
        marks.push({ type: "code" });
    if (attrs.underline)
        marks.push({ type: "underline" });
    if (attrs.strike)
        marks.push({ type: "strike" });
    if (attrs.commentHighlight) {
        marks.push({ type: "commentHighlight", attrs: attrs.commentHighlight });
    }
    return marks;
}
// ---------------------------------------------------------------------------
// BlockNote ↔ ProseMirror JSON conversion
// ---------------------------------------------------------------------------
// BlockNote stores: blockGroup → blockContainer(id) → content node
// ProseMirror stores: flat doc → heading/paragraph/bulletList/orderedList/...
let bnIdCounter = 0;
function nextBnId() {
    return `bn-${Date.now().toString(36)}-${(bnIdCounter++).toString(36)}`;
}
/**
 * Check whether the Y.js fragment contains BlockNote structure
 * (first child is a blockGroup element).
 */
function isBlockNoteFragment(fragment) {
    if (fragment.length === 0)
        return false;
    const first = fragment.get(0);
    return first instanceof Y.XmlElement && first.nodeName === "blockGroup";
}
/**
 * Convert ProseMirror JSON → BlockNote Y.js structure.
 * Wraps each PM node in blockGroup → blockContainer.
 */
function proseMirrorJSONToBlockNoteYFragment(doc, fragment, content) {
    if (content.type !== "doc") {
        throw new Error(`expected ProseMirror doc node, got ${content.type}`);
    }
    const blockGroup = new Y.XmlElement("blockGroup");
    for (const child of content.content ?? []) {
        pmNodeToBlockNote(blockGroup, child);
    }
    fragment.insert(fragment.length, [blockGroup]);
}
function pmNodeToBlockNote(blockGroup, node) {
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
                        if (yChild)
                            quote.insert(quote.length, [yChild]);
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
function makeBnContainer() {
    const container = new Y.XmlElement("blockContainer");
    container.setAttribute("id", nextBnId());
    return container;
}
/** Copy listItem's paragraph children's text content into a BlockNote list item element */
function copyListItemContent(listItem, target) {
    for (const child of listItem.content ?? []) {
        if (child.type === "paragraph") {
            for (const textNode of child.content ?? []) {
                const yChild = proseMirrorNodeToYNode(textNode);
                if (yChild)
                    target.insert(target.length, [yChild]);
            }
        }
    }
}
/** Convert PM table node to BlockNote table with tableParagraph inside cells */
function pmTableToBlockNote(table) {
    const tableEl = new Y.XmlElement("table");
    for (const [key, value] of Object.entries(table.attrs ?? {})) {
        if (value !== undefined && value !== null) {
            tableEl.setAttribute(key, String(value));
        }
    }
    for (const row of table.content ?? []) {
        const rowEl = new Y.XmlElement("tableRow");
        for (const cell of row.content ?? []) {
            const cellEl = new Y.XmlElement(cell.type === "tableHeader" ? "tableHeader" : "tableCell");
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
                        if (yChild)
                            tp.insert(tp.length, [yChild]);
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
function yBlockNoteFragmentToProseMirrorJSON(fragment) {
    const content = [];
    fragment.forEach((item) => {
        if (item instanceof Y.XmlElement && item.nodeName === "blockGroup") {
            content.push(...extractBlocksFromGroup(item));
        }
        else {
            // Fallback for non-BlockNote nodes
            const node = yNodeToProseMirror(item);
            if (node) {
                if (Array.isArray(node))
                    content.push(...node);
                else
                    content.push(node);
            }
        }
    });
    return { type: "doc", content };
}
function extractBlocksFromGroup(blockGroup) {
    const result = [];
    let currentListType = null;
    let currentListItems = [];
    const flushList = () => {
        if (currentListType && currentListItems.length > 0) {
            result.push({ type: currentListType, content: currentListItems });
            currentListItems = [];
            currentListType = null;
        }
    };
    blockGroup.forEach((container) => {
        if (!(container instanceof Y.XmlElement) || container.nodeName !== "blockContainer")
            return;
        const contentNode = container.get(0);
        if (!contentNode || !(contentNode instanceof Y.XmlElement))
            return;
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
        }
        else {
            flushList();
            if (nodeName === "quote") {
                const textContent = extractChildContent(contentNode);
                result.push({
                    type: "blockquote",
                    content: [{ type: "paragraph", content: textContent }],
                });
            }
            else if (nodeName === "table") {
                result.push(bnTableToProseMirror(contentNode));
            }
            else {
                // heading, paragraph, codeBlock — pass through
                const node = yNodeToProseMirror(contentNode);
                if (node) {
                    if (Array.isArray(node))
                        result.push(...node);
                    else
                        result.push(node);
                }
            }
        }
    });
    flushList();
    return result;
}
function extractChildContent(node) {
    const content = [];
    node.forEach((child) => {
        const converted = yNodeToProseMirror(child);
        if (converted) {
            if (Array.isArray(converted))
                content.push(...converted);
            else
                content.push(converted);
        }
    });
    return content;
}
/** Convert BlockNote table Y.js node to ProseMirror JSON (tableParagraph → paragraph) */
function bnTableToProseMirror(tableNode) {
    const rows = [];
    tableNode.forEach((row) => {
        if (!(row instanceof Y.XmlElement) || row.nodeName !== "tableRow")
            return;
        const cells = [];
        row.forEach((cell) => {
            if (!(cell instanceof Y.XmlElement))
                return;
            const cellType = cell.nodeName === "tableHeader" ? "tableHeader" : "tableCell";
            const cellAttrs = cell.getAttributes();
            const cellContent = [];
            cell.forEach((child) => {
                if (child instanceof Y.XmlElement && (child.nodeName === "tableParagraph" || child.nodeName === "paragraph")) {
                    const textContent = [];
                    child.forEach((textChild) => {
                        const converted = yNodeToProseMirror(textChild);
                        if (converted) {
                            if (Array.isArray(converted))
                                textContent.push(...converted);
                            else
                                textContent.push(converted);
                        }
                    });
                    cellContent.push({ type: "paragraph", content: textContent });
                }
            });
            const pmCell = { type: cellType };
            if (Object.keys(cellAttrs).length > 0)
                pmCell.attrs = cellAttrs;
            if (cellContent.length > 0)
                pmCell.content = cellContent;
            cells.push(pmCell);
        });
        rows.push({ type: "tableRow", content: cells });
    });
    const attrs = tableNode.getAttributes();
    const pmTable = { type: "table", content: rows };
    if (Object.keys(attrs).length > 0)
        pmTable.attrs = attrs;
    return pmTable;
}
