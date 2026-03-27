import * as Y from "yjs";
import { htmlToYXmlFragment } from "./html-to-yjs";
import {
  proseMirrorJSONToYXmlFragment,
  yXmlFragmentToProseMirrorJSON,
  type PMNode,
} from "./prosemirror";

function toFragment(content: string): Y.XmlFragment {
  const doc = new Y.Doc();
  const fragment = doc.getXmlFragment("default");
  htmlToYXmlFragment(doc, fragment, content);
  return fragment;
}

{
  const fragment = toFragment("<h2>Design</h2><p>Hello <code>world()</code></p><pre><code class=\"language-go\">fmt.Println(1)</code></pre>");
  const json = yXmlFragmentToProseMirrorJSON(fragment);

  console.assert(json.type === "doc", "expected doc root");
  console.assert(json.content?.[0]?.type === "heading", "expected heading node");
  console.assert(json.content?.[1]?.type === "paragraph", "expected paragraph node");
  console.assert(json.content?.[2]?.type === "codeBlock", "expected codeBlock node");

  console.log("PASS: Y.js -> ProseMirror JSON");
}

{
  const doc = new Y.Doc();
  const fragment = doc.getXmlFragment("default");
  const content: PMNode = {
    type: "doc",
    content: [
      { type: "heading", attrs: { level: 2 }, content: [{ type: "text", text: "API" }] },
      {
        type: "paragraph",
        content: [
          { type: "text", text: "Use " },
          { type: "text", text: "client()", marks: [{ type: "code" }] },
        ],
      },
    ],
  };

  proseMirrorJSONToYXmlFragment(doc, fragment, content);
  console.assert(fragment.length === 2, `expected 2 top-level nodes, got ${fragment.length}`);

  const paragraph = fragment.get(1) as Y.XmlElement;
  const text = paragraph.get(0) as Y.XmlText;
  console.assert(
    text.toDelta().some((op: { insert?: unknown; attributes?: { code?: boolean } }) => typeof op.insert === "string" && op.attributes?.code),
    "expected code mark to be preserved",
  );

  console.log("PASS: ProseMirror JSON -> Y.js");
}
