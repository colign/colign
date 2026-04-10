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
const Y = __importStar(require("yjs"));
const html_to_yjs_1 = require("./html-to-yjs");
function convert(html) {
    const doc = new Y.Doc();
    const fragment = doc.getXmlFragment("default");
    (0, html_to_yjs_1.htmlToYXmlFragment)(doc, fragment, html);
    return doc;
}
function fragmentToString(fragment) {
    const items = [];
    fragment.forEach((item) => {
        if (item instanceof Y.XmlElement) {
            items.push(`${item.nodeName}(${JSON.stringify(item.getAttributes())})[${elementChildren(item)}]`);
        }
        else if (item instanceof Y.XmlText) {
            items.push(`text("${item.toString()}")`);
        }
    });
    return items.join(", ");
}
function elementChildren(el) {
    const items = [];
    el.forEach((child) => {
        if (child instanceof Y.XmlElement) {
            items.push(`${child.nodeName}(${JSON.stringify(child.getAttributes())})[${elementChildren(child)}]`);
        }
        else if (child instanceof Y.XmlText) {
            items.push(`text("${child.toString()}")`);
        }
    });
    return items.join(", ");
}
// Test heading conversion
{
    const doc = convert("<h1>Title</h1><h2>Subtitle</h2><h3>Section</h3>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 3, `Expected 3 elements, got ${fragment.length}`);
    const h1 = fragment.get(0);
    console.assert(h1.nodeName === "heading", `Expected heading, got ${h1.nodeName}`);
    console.assert(h1.getAttribute("level") === "1", `Expected level 1, got ${h1.getAttribute("level")}`);
    const h2 = fragment.get(1);
    console.assert(h2.getAttribute("level") === "2", `Expected level 2`);
    const h3 = fragment.get(2);
    console.assert(h3.getAttribute("level") === "3", `Expected level 3`);
    console.log("PASS: headings");
}
// Test paragraph conversion
{
    const doc = convert("<p>Hello world</p><p>Second paragraph</p>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 2, `Expected 2 paragraphs, got ${fragment.length}`);
    const p1 = fragment.get(0);
    console.assert(p1.nodeName === "paragraph", `Expected paragraph, got ${p1.nodeName}`);
    console.log("PASS: paragraphs");
}
// Test bullet list merging
{
    // markdownToHTML produces separate <ul> for each item, but they should be merged
    const doc = convert("<ul><li>Item 1</li></ul><ul><li>Item 2</li></ul><ul><li>Item 3</li></ul>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 bulletList (merged), got ${fragment.length}`);
    const list = fragment.get(0);
    console.assert(list.nodeName === "bulletList", `Expected bulletList, got ${list.nodeName}`);
    let listItemCount = 0;
    list.forEach(() => listItemCount++);
    console.assert(listItemCount === 3, `Expected 3 list items, got ${listItemCount}`);
    console.log("PASS: bullet list merging");
}
// Test mixed content
{
    const doc = convert("<h2>Design</h2><p>Overview text</p><ul><li>Step 1</li><li>Step 2</li></ul><p>Conclusion</p>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 4, `Expected 4 elements (h2, p, ul, p), got ${fragment.length}`);
    const el0 = fragment.get(0);
    const el1 = fragment.get(1);
    const el2 = fragment.get(2);
    const el3 = fragment.get(3);
    console.assert(el0.nodeName === "heading", `[0] Expected heading`);
    console.assert(el1.nodeName === "paragraph", `[1] Expected paragraph`);
    console.assert(el2.nodeName === "bulletList", `[2] Expected bulletList`);
    console.assert(el3.nodeName === "paragraph", `[3] Expected paragraph`);
    console.log("PASS: mixed content");
}
// Test code block conversion
{
    const doc = convert('<h2>Example</h2><pre><code class="language-tsx">const x = 1;\nconsole.log(x);</code></pre>');
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 2, `Expected 2 elements (h2, codeBlock), got ${fragment.length}`);
    const codeBlock = fragment.get(1);
    console.assert(codeBlock.nodeName === "codeBlock", `Expected codeBlock, got ${codeBlock.nodeName}`);
    const codeText = codeBlock.get(0);
    console.assert(codeText.toString() === "const x = 1;\nconsole.log(x);", `Expected code block text to round-trip, got ${JSON.stringify(codeText.toString())}`);
    console.log("PASS: code blocks");
}
// Test inline formatting preserved in heading and list item
{
    const doc = convert("<h2>Hello <code>world()</code></h2><ol><li><strong>Step</strong> <code>one()</code></li></ol>");
    const fragment = doc.getXmlFragment("default");
    const heading = fragment.get(0);
    const list = fragment.get(1);
    const listItemParagraph = list.get(0).get(0);
    const headingText = heading.get(0);
    const listItemText = listItemParagraph.get(0);
    const headingDelta = headingText.toDelta();
    const listDelta = listItemText.toDelta();
    console.assert(headingDelta.some((op) => typeof op.insert === "string" && op.attributes?.code), "Expected heading inline code formatting to be preserved");
    console.assert(listDelta.some((op) => typeof op.insert === "string" && op.attributes?.bold), "Expected list item bold formatting to be preserved");
    console.assert(listDelta.some((op) => typeof op.insert === "string" && op.attributes?.code), "Expected list item inline code formatting to be preserved");
    console.log("PASS: inline formatting");
}
// Test blockquote conversion
{
    const doc = convert("<blockquote><p>This is a quote</p></blockquote>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 blockquote, got ${fragment.length}`);
    const bq = fragment.get(0);
    console.assert(bq.nodeName === "blockquote", `Expected blockquote, got ${bq.nodeName}`);
    const p = bq.get(0);
    console.assert(p.nodeName === "paragraph", `Expected paragraph inside blockquote, got ${p.nodeName}`);
    const text = p.get(0);
    console.assert(text.toString() === "This is a quote", `Expected 'This is a quote', got '${text.toString()}'`);
    console.log("PASS: blockquote");
}
// Test blockquote with multiple paragraphs
{
    const doc = convert("<blockquote><p>First line</p><p>Second line</p></blockquote>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 blockquote, got ${fragment.length}`);
    const bq = fragment.get(0);
    let childCount = 0;
    bq.forEach(() => childCount++);
    console.assert(childCount === 2, `Expected 2 paragraphs inside blockquote, got ${childCount}`);
    console.log("PASS: blockquote with multiple paragraphs");
}
// Test blockquote without <p> wrapper
{
    const doc = convert("<blockquote>Plain quoted text</blockquote>");
    const fragment = doc.getXmlFragment("default");
    const bq = fragment.get(0);
    console.assert(bq.nodeName === "blockquote", `Expected blockquote`);
    const p = bq.get(0);
    console.assert(p.nodeName === "paragraph", `Expected paragraph inside blockquote`);
    console.log("PASS: blockquote without p wrapper");
}
// Test blockquote with real marked.parse() output (contains newlines)
{
    const doc = convert("<blockquote>\n<p>Quoted via marked</p>\n</blockquote>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 blockquote, got ${fragment.length}`);
    const bq = fragment.get(0);
    console.assert(bq.nodeName === "blockquote", `Expected blockquote, got ${bq.nodeName}`);
    const p = bq.get(0);
    console.assert(p.nodeName === "paragraph", `Expected paragraph inside blockquote`);
    const text = p.get(0);
    console.assert(text.toString() === "Quoted via marked", `Expected 'Quoted via marked', got '${text.toString()}'`);
    console.log("PASS: blockquote with marked.parse() newlines");
}
// Test blockquote with inline formatting
{
    const doc = convert("<blockquote><p>This is <strong>important</strong> text</p></blockquote>");
    const fragment = doc.getXmlFragment("default");
    const bq = fragment.get(0);
    const p = bq.get(0);
    const text = p.get(0);
    const delta = text.toDelta();
    console.assert(delta.some((op) => op.attributes?.bold && op.insert === "important"), "Expected bold formatting preserved in blockquote");
    console.assert(delta.some((op) => !op.attributes?.bold && op.insert.includes("text")), "Expected plain text after bold to NOT inherit bold");
    console.log("PASS: blockquote with inline formatting");
}
// Test blockquote in mixed content
{
    const doc = convert("<h2>Title</h2><blockquote><p>A quote</p></blockquote><p>After quote</p>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 3, `Expected 3 elements (h2, blockquote, p), got ${fragment.length}`);
    const el0 = fragment.get(0);
    const el1 = fragment.get(1);
    const el2 = fragment.get(2);
    console.assert(el0.nodeName === "heading", `[0] Expected heading, got ${el0.nodeName}`);
    console.assert(el1.nodeName === "blockquote", `[1] Expected blockquote, got ${el1.nodeName}`);
    console.assert(el2.nodeName === "paragraph", `[2] Expected paragraph, got ${el2.nodeName}`);
    console.log("PASS: blockquote in mixed content");
}
// Test empty input
{
    const doc = convert("");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 0, `Expected 0 elements for empty input`);
    console.log("PASS: empty input");
}
// Test HTML entity unescaping
{
    const doc = convert("<p>A &amp; B &lt; C</p>");
    const fragment = doc.getXmlFragment("default");
    const p = fragment.get(0);
    let textContent = "";
    p.forEach((child) => {
        if (child instanceof Y.XmlText) {
            textContent = child.toString();
        }
    });
    console.assert(textContent === "A & B < C", `Expected 'A & B < C', got '${textContent}'`);
    console.log("PASS: HTML entity unescaping");
}
// Test listItem has paragraph child (TipTap requirement)
{
    const doc = convert("<ul><li>Item</li></ul>");
    const fragment = doc.getXmlFragment("default");
    const list = fragment.get(0);
    const listItem = list.get(0);
    console.assert(listItem.nodeName === "listItem", `Expected listItem`);
    const paragraph = listItem.get(0);
    console.assert(paragraph.nodeName === "paragraph", `Expected paragraph inside listItem, got ${paragraph.nodeName}`);
    console.log("PASS: listItem contains paragraph");
}
// Test table conversion (inline HTML)
{
    const doc = convert("<table><thead><tr><th>Stage</th><th>Owner</th></tr></thead><tbody><tr><td>Design</td><td>Ben</td></tr></tbody></table>");
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 table, got ${fragment.length}`);
    const table = fragment.get(0);
    console.assert(table.nodeName === "table", `Expected table, got ${table.nodeName}`);
    const headerRow = table.get(0);
    const bodyRow = table.get(1);
    console.assert(headerRow.nodeName === "tableRow", `Expected tableRow, got ${headerRow.nodeName}`);
    console.assert(bodyRow.nodeName === "tableRow", `Expected tableRow, got ${bodyRow.nodeName}`);
    const headerCell = headerRow.get(0);
    const bodyCell = bodyRow.get(0);
    console.assert(headerCell.nodeName === "tableHeader", `Expected tableHeader, got ${headerCell.nodeName}`);
    console.assert(bodyCell.nodeName === "tableCell", `Expected tableCell, got ${bodyCell.nodeName}`);
    // BlockNote requires "tableParagraph" (not "paragraph") inside table cells
    const headerParagraph = headerCell.get(0);
    const bodyParagraph = bodyCell.get(0);
    console.assert(headerParagraph.nodeName === "tableParagraph", `Expected tableParagraph inside header cell, got ${headerParagraph.nodeName}`);
    console.assert(bodyParagraph.nodeName === "tableParagraph", `Expected tableParagraph inside body cell, got ${bodyParagraph.nodeName}`);
    const headerText = headerParagraph.get(0);
    const bodyText = bodyParagraph.get(0);
    console.assert(headerText.toString() === "Stage", `Expected 'Stage', got '${headerText.toString()}'`);
    console.assert(bodyText.toString() === "Design", `Expected 'Design', got '${bodyText.toString()}'`);
    console.log("PASS: table conversion");
}
// Test table conversion with Goldmark GFM output (newlines between tags)
{
    const goldmarkHtml = `<table>
<thead>
<tr>
<th>원칙</th>
<th>설명</th>
</tr>
</thead>
<tbody>
<tr>
<td><strong>Phase 1/2 호환</strong></td>
<td>Phase 1은 규칙 기반</td>
</tr>
</tbody>
</table>`;
    const doc = convert(goldmarkHtml);
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 table from Goldmark HTML, got ${fragment.length}`);
    const table = fragment.get(0);
    console.assert(table.nodeName === "table", `Expected table, got ${table.nodeName}`);
    let rowCount = 0;
    table.forEach(() => rowCount++);
    console.assert(rowCount === 2, `Expected 2 table rows, got ${rowCount}`);
    const headerRow = table.get(0);
    const bodyRow = table.get(1);
    // Verify header cell
    const headerCell = headerRow.get(0);
    console.assert(headerCell.nodeName === "tableHeader", `Expected tableHeader, got ${headerCell.nodeName}`);
    const headerParagraph = headerCell.get(0);
    console.assert(headerParagraph.nodeName === "tableParagraph", `Expected tableParagraph in Goldmark header cell, got ${headerParagraph.nodeName}`);
    const headerText = headerParagraph.get(0);
    console.assert(headerText.toString() === "원칙", `Expected '원칙', got '${headerText.toString()}'`);
    // Verify body cell with bold formatting
    const bodyCell = bodyRow.get(0);
    console.assert(bodyCell.nodeName === "tableCell", `Expected tableCell, got ${bodyCell.nodeName}`);
    const bodyParagraph = bodyCell.get(0);
    console.assert(bodyParagraph.nodeName === "tableParagraph", `Expected tableParagraph in Goldmark body cell, got ${bodyParagraph.nodeName}`);
    const bodyText = bodyParagraph.get(0);
    const delta = bodyText.toDelta();
    console.assert(delta.some((op) => op.attributes?.bold && op.insert === "Phase 1/2 호환"), "Expected bold formatting preserved in Goldmark table cell");
    console.log("PASS: table conversion (Goldmark GFM format)");
}
// Test table with multiple body rows
{
    const multiRowHtml = `<table>
<thead>
<tr>
<th>Decision</th>
<th>Status</th>
</tr>
</thead>
<tbody>
<tr>
<td>Event Sourcing</td>
<td><strong>Approved</strong></td>
</tr>
<tr>
<td>CQRS</td>
<td>Proposed</td>
</tr>
<tr>
<td>DDD</td>
<td><em>In Review</em></td>
</tr>
</tbody>
</table>`;
    const doc = convert(multiRowHtml);
    const fragment = doc.getXmlFragment("default");
    console.assert(fragment.length === 1, `Expected 1 table, got ${fragment.length}`);
    const table = fragment.get(0);
    let rowCount = 0;
    table.forEach(() => rowCount++);
    console.assert(rowCount === 4, `Expected 4 rows (1 header + 3 body), got ${rowCount}`);
    // Verify all cells use tableParagraph
    for (let r = 0; r < rowCount; r++) {
        const row = table.get(r);
        row.forEach((cell) => {
            if (cell instanceof Y.XmlElement) {
                const para = cell.get(0);
                console.assert(para.nodeName === "tableParagraph", `Expected tableParagraph in row ${r}, got ${para.nodeName}`);
            }
        });
    }
    console.log("PASS: table with multiple body rows");
}
// ==========================================================================
// BlockNote format tests (htmlToBlockNoteFragment)
// ==========================================================================
function convertBN(html) {
    const doc = new Y.Doc();
    const fragment = doc.getXmlFragment("document-store");
    (0, html_to_yjs_1.htmlToBlockNoteFragment)(doc, fragment, html);
    return doc;
}
// BlockNote: basic structure — blockGroup → blockContainer → content
{
    const doc = convertBN("<h2>Title</h2><p>Hello</p>");
    const fragment = doc.getXmlFragment("document-store");
    console.assert(fragment.length === 1, `BN: Expected 1 blockGroup, got ${fragment.length}`);
    const blockGroup = fragment.get(0);
    console.assert(blockGroup.nodeName === "blockGroup", `BN: Expected blockGroup, got ${blockGroup.nodeName}`);
    let containerCount = 0;
    blockGroup.forEach(() => containerCount++);
    console.assert(containerCount === 2, `BN: Expected 2 blockContainers, got ${containerCount}`);
    const bc0 = blockGroup.get(0);
    const bc1 = blockGroup.get(1);
    console.assert(bc0.nodeName === "blockContainer", `BN: Expected blockContainer, got ${bc0.nodeName}`);
    console.assert(bc1.nodeName === "blockContainer", `BN: Expected blockContainer, got ${bc1.nodeName}`);
    console.assert(bc0.getAttribute("id") !== undefined, `BN: blockContainer must have id`);
    const heading = bc0.get(0);
    const paragraph = bc1.get(0);
    console.assert(heading.nodeName === "heading", `BN: Expected heading content, got ${heading.nodeName}`);
    console.assert(paragraph.nodeName === "paragraph", `BN: Expected paragraph content, got ${paragraph.nodeName}`);
    console.log("PASS: BlockNote basic structure");
}
// BlockNote: code block with language attribute
{
    const doc = convertBN('<pre><code class="language-go">func main() {}</code></pre>');
    const fragment = doc.getXmlFragment("document-store");
    const blockGroup = fragment.get(0);
    const container = blockGroup.get(0);
    console.assert(container.nodeName === "blockContainer", `BN code: Expected blockContainer`);
    const codeBlock = container.get(0);
    console.assert(codeBlock.nodeName === "codeBlock", `BN code: Expected codeBlock, got ${codeBlock.nodeName}`);
    console.assert(codeBlock.getAttribute("language") === "go", `BN code: Expected language=go, got ${codeBlock.getAttribute("language")}`);
    const codeText = codeBlock.get(0);
    console.assert(codeText.toString() === "func main() {}", `BN code: Expected code text, got ${codeText.toString()}`);
    console.log("PASS: BlockNote code block");
}
// BlockNote: code block without language defaults to "text"
{
    const doc = convertBN("<pre><code>plain code</code></pre>");
    const fragment = doc.getXmlFragment("document-store");
    const blockGroup = fragment.get(0);
    const codeBlock = blockGroup.get(0).get(0);
    console.assert(codeBlock.getAttribute("language") === "text", `BN code: Expected language=text for no-lang code block`);
    console.log("PASS: BlockNote code block default language");
}
// BlockNote: lists become individual blockContainers with bulletListItem/numberedListItem
{
    const doc = convertBN("<ul><li>A</li><li>B</li></ul><ol><li>One</li></ol>");
    const fragment = doc.getXmlFragment("document-store");
    const blockGroup = fragment.get(0);
    let count = 0;
    blockGroup.forEach(() => count++);
    console.assert(count === 3, `BN list: Expected 3 blockContainers, got ${count}`);
    const bc0 = blockGroup.get(0);
    const bc1 = blockGroup.get(1);
    const bc2 = blockGroup.get(2);
    const li0 = bc0.get(0);
    const li1 = bc1.get(0);
    const li2 = bc2.get(0);
    console.assert(li0.nodeName === "bulletListItem", `BN list: Expected bulletListItem, got ${li0.nodeName}`);
    console.assert(li1.nodeName === "bulletListItem", `BN list: Expected bulletListItem, got ${li1.nodeName}`);
    console.assert(li2.nodeName === "numberedListItem", `BN list: Expected numberedListItem, got ${li2.nodeName}`);
    console.log("PASS: BlockNote lists");
}
// BlockNote: blockquote → "quote"
{
    const doc = convertBN("<blockquote><p>Quoted text</p></blockquote>");
    const fragment = doc.getXmlFragment("document-store");
    const blockGroup = fragment.get(0);
    const container = blockGroup.get(0);
    const quote = container.get(0);
    console.assert(quote.nodeName === "quote", `BN quote: Expected "quote", got ${quote.nodeName}`);
    console.log("PASS: BlockNote blockquote → quote");
}
// BlockNote: table inside blockContainer
{
    const doc = convertBN("<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>");
    const fragment = doc.getXmlFragment("document-store");
    const blockGroup = fragment.get(0);
    const container = blockGroup.get(0);
    console.assert(container.nodeName === "blockContainer", `BN table: Expected blockContainer`);
    const table = container.get(0);
    console.assert(table.nodeName === "table", `BN table: Expected table, got ${table.nodeName}`);
    console.log("PASS: BlockNote table");
}
// BlockNote: mixed content — full document
{
    const doc = convertBN('<h2>Title</h2><p>Text</p><pre><code class="language-sql">SELECT 1;</code></pre>' +
        "<ul><li>Item</li></ul><blockquote><p>Quote</p></blockquote>" +
        "<table><thead><tr><th>Col</th></tr></thead><tbody><tr><td>Val</td></tr></tbody></table>");
    const fragment = doc.getXmlFragment("document-store");
    const blockGroup = fragment.get(0);
    let count = 0;
    blockGroup.forEach(() => count++);
    // heading, paragraph, codeBlock, bulletListItem, quote, table = 6
    console.assert(count === 6, `BN mixed: Expected 6 blockContainers, got ${count}`);
    const types = [];
    blockGroup.forEach((bc) => {
        if (bc instanceof Y.XmlElement) {
            const content = bc.get(0);
            types.push(content.nodeName);
        }
    });
    console.assert(types[0] === "heading", `BN mixed[0]: Expected heading, got ${types[0]}`);
    console.assert(types[1] === "paragraph", `BN mixed[1]: Expected paragraph, got ${types[1]}`);
    console.assert(types[2] === "codeBlock", `BN mixed[2]: Expected codeBlock, got ${types[2]}`);
    console.assert(types[3] === "bulletListItem", `BN mixed[3]: Expected bulletListItem, got ${types[3]}`);
    console.assert(types[4] === "quote", `BN mixed[4]: Expected quote, got ${types[4]}`);
    console.assert(types[5] === "table", `BN mixed[5]: Expected table, got ${types[5]}`);
    console.log("PASS: BlockNote mixed content");
}
console.log("\nAll tests passed!");
