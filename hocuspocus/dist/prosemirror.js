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
