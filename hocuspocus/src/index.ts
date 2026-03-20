import { Hocuspocus } from "@hocuspocus/server";
import { Pool } from "pg";
import * as Y from "yjs";
import * as crypto from "crypto";

const pool = new Pool({
  connectionString:
    process.env.DATABASE_URL ??
    "postgres://postgres:postgres@localhost:5432/colign",
});

const JWT_SECRET = process.env.JWT_SECRET ?? "dev-secret-change-in-production";

function verifyJwt(token: string): { user_id: number; email: string; name: string } | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;

    // Verify signature
    const header = parts[0];
    const payload = parts[1];
    const signature = parts[2];
    const expected = crypto
      .createHmac("sha256", JWT_SECRET)
      .update(`${header}.${payload}`)
      .digest("base64url");

    if (signature !== expected) return null;

    const decoded = JSON.parse(Buffer.from(payload, "base64url").toString());

    // Check expiry
    if (decoded.exp && decoded.exp < Date.now() / 1000) return null;

    return decoded;
  } catch {
    return null;
  }
}

const server = new Hocuspocus({
  port: Number(process.env.PORT ?? 1234),

  async onAuthenticate({ token }) {
    const payload = verifyJwt(token);
    if (!payload) {
      throw new Error("Unauthorized");
    }
    return { user: payload };
  },

  async onLoadDocument({ documentName, document }) {
    // documentName format: "change-{id}-{docType}"
    const parts = documentName.split("-");
    if (parts.length < 3) return;

    const changeId = parts[1];
    const docType = parts.slice(2).join("-");

    try {
      const result = await pool.query(
        "SELECT content FROM documents WHERE change_id = $1 AND type = $2 LIMIT 1",
        [changeId, docType],
      );

      if (result.rows.length > 0 && result.rows[0].content) {
        // Apply existing HTML content to Y.js doc
        // The content will be synced to connecting clients
        const yXmlFragment = document.getXmlFragment("default");
        if (yXmlFragment.length === 0) {
          // Store HTML as metadata so clients can initialize from it
          const yMeta = document.getMap("meta");
          yMeta.set("initialHtml", result.rows[0].content);
        }
      }
    } catch (err) {
      console.error("Failed to load document:", err);
    }
  },

  async onStoreDocument({ documentName, document }) {
    // Save Y.js document content back to DB
    const parts = documentName.split("-");
    if (parts.length < 3) return;

    const changeId = parts[1];
    const docType = parts.slice(2).join("-");

    try {
      // Get HTML from Y.js XML fragment
      const yXmlFragment = document.getXmlFragment("default");
      const content = yXmlFragmentToHtml(yXmlFragment);

      if (!content) return;

      // Upsert document
      await pool.query(
        `INSERT INTO documents (change_id, type, content, version)
         VALUES ($1, $2, $3, 1)
         ON CONFLICT (change_id, type)
         DO UPDATE SET content = $3, version = documents.version + 1, updated_at = NOW()`,
        [changeId, docType, content],
      );
    } catch (err) {
      console.error("Failed to store document:", err);
    }
  },

  async onConnect() {
    console.log("Client connected");
  },

  async onDisconnect() {
    console.log("Client disconnected");
  },
});

function yXmlFragmentToHtml(fragment: Y.XmlFragment): string {
  let html = "";
  fragment.forEach((item) => {
    if (item instanceof Y.XmlElement) {
      html += xmlElementToHtml(item);
    } else if (item instanceof Y.XmlText) {
      html += item.toString();
    }
  });
  return html;
}

function xmlElementToHtml(element: Y.XmlElement): string {
  const tag = element.nodeName;
  const attrs = element.getAttributes();
  let attrStr = "";
  for (const [key, value] of Object.entries(attrs)) {
    attrStr += ` ${key}="${value}"`;
  }

  let inner = "";
  element.forEach((child) => {
    if (child instanceof Y.XmlElement) {
      inner += xmlElementToHtml(child);
    } else if (child instanceof Y.XmlText) {
      inner += child.toString();
    }
  });

  if (["br", "hr", "img"].includes(tag)) {
    return `<${tag}${attrStr} />`;
  }
  return `<${tag}${attrStr}>${inner}</${tag}>`;
}

server.listen();
console.log(`Hocuspocus listening on port ${process.env.PORT ?? 1234}`);
