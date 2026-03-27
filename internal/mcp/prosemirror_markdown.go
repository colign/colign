package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type pmNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Marks   []pmMark       `json:"marks,omitempty"`
	Content []pmNode       `json:"content,omitempty"`
}

type pmMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

func exportDocumentToMarkdown(content string) (string, error) {
	if isProseMirrorJSONContent(content) {
		return proseMirrorJSONToMarkdown(content)
	}
	return htmlToMarkdown(content)
}

func isProseMirrorJSONContent(content string) bool {
	var node pmNode
	if err := json.Unmarshal([]byte(content), &node); err != nil {
		return false
	}
	return node.Type == "doc"
}

func proseMirrorJSONToMarkdown(content string) (string, error) {
	var doc pmNode
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return "", fmt.Errorf("failed to parse ProseMirror JSON: %w", err)
	}
	if doc.Type != "doc" {
		return "", fmt.Errorf("expected ProseMirror doc root, got %q", doc.Type)
	}

	renderer := pmMarkdownRenderer{}
	renderer.writeChildren(doc.Content, 0)
	return strings.TrimSpace(renderer.String()), nil
}

type pmMarkdownRenderer struct {
	b strings.Builder
}

func (r *pmMarkdownRenderer) String() string {
	return r.b.String()
}

func (r *pmMarkdownRenderer) writeChildren(nodes []pmNode, depth int) {
	for _, node := range nodes {
		r.writeNode(node, depth)
	}
}

func (r *pmMarkdownRenderer) writeNode(node pmNode, depth int) {
	switch node.Type {
	case "paragraph":
		text := r.inlineContent(node.Content)
		if text != "" {
			r.writeBlock(text)
		}
	case "heading":
		level := attrInt(node.Attrs, "level", 1)
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		r.writeBlock(strings.Repeat("#", level) + " " + r.inlineContent(node.Content))
	case "bulletList":
		r.writeList(node.Content, depth, false)
		r.ensureGap()
	case "orderedList":
		r.writeList(node.Content, depth, true)
		r.ensureGap()
	case "blockquote":
		text := strings.TrimSpace(r.blockText(node.Content, depth))
		if text == "" {
			return
		}
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				r.b.WriteString(">\n")
				continue
			}
			r.b.WriteString("> ")
			r.b.WriteString(line)
			r.b.WriteByte('\n')
		}
		r.b.WriteByte('\n')
	case "codeBlock":
		lang := attrString(node.Attrs, "language")
		r.b.WriteString("```")
		r.b.WriteString(lang)
		r.b.WriteByte('\n')
		r.b.WriteString(r.codeText(node.Content))
		if !strings.HasSuffix(r.b.String(), "\n") {
			r.b.WriteByte('\n')
		}
		r.b.WriteString("```\n\n")
	case "horizontalRule":
		r.writeBlock("---")
	case "table":
		r.writeTable(node)
	default:
		if text := strings.TrimSpace(r.blockText([]pmNode{node}, depth)); text != "" {
			r.writeBlock(text)
		}
	}
}

func (r *pmMarkdownRenderer) writeList(items []pmNode, depth int, ordered bool) {
	for idx, item := range items {
		if item.Type != "listItem" {
			continue
		}
		prefix := "- "
		if ordered {
			prefix = strconv.Itoa(idx+1) + ". "
		}
		indent := strings.Repeat("  ", depth)
		first := true
		for _, child := range item.Content {
			switch child.Type {
			case "paragraph":
				text := r.inlineContent(child.Content)
				if first {
					r.b.WriteString(indent)
					r.b.WriteString(prefix)
					r.b.WriteString(text)
					r.b.WriteByte('\n')
				} else {
					r.b.WriteString(indent)
					r.b.WriteString("  ")
					r.b.WriteString(text)
					r.b.WriteByte('\n')
				}
				first = false
			case "bulletList":
				if first {
					r.b.WriteString(indent)
					r.b.WriteString(prefix)
					r.b.WriteByte('\n')
					first = false
				}
				r.writeList(child.Content, depth+1, false)
			case "orderedList":
				if first {
					r.b.WriteString(indent)
					r.b.WriteString(prefix)
					r.b.WriteByte('\n')
					first = false
				}
				r.writeList(child.Content, depth+1, true)
			default:
				text := strings.TrimSpace(r.blockText([]pmNode{child}, depth+1))
				if text == "" {
					continue
				}
				if first {
					r.b.WriteString(indent)
					r.b.WriteString(prefix)
					r.b.WriteString(text)
					r.b.WriteByte('\n')
					first = false
				} else {
					r.b.WriteString(indent)
					r.b.WriteString("  ")
					r.b.WriteString(text)
					r.b.WriteByte('\n')
				}
			}
		}
		if first {
			r.b.WriteString(indent)
			r.b.WriteString(prefix)
			r.b.WriteByte('\n')
		}
	}
}

func (r *pmMarkdownRenderer) writeTable(table pmNode) {
	rows := make([][]string, 0, len(table.Content))
	headerRows := 0
	for _, row := range table.Content {
		if row.Type != "tableRow" {
			continue
		}
		cells := make([]string, 0, len(row.Content))
		isHeader := true
		for _, cell := range row.Content {
			cells = append(cells, r.inlineContent(cell.Content))
			if cell.Type != "tableHeader" {
				isHeader = false
			}
		}
		if len(cells) == 0 {
			continue
		}
		if isHeader {
			headerRows++
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return
	}

	header := rows[0]
	body := rows[1:]
	if headerRows == 0 {
		body = rows[1:]
	}
	r.b.WriteString("| ")
	r.b.WriteString(strings.Join(header, " | "))
	r.b.WriteString(" |\n| ")
	r.b.WriteString(strings.Join(makeSeparatorRow(len(header)), " | "))
	r.b.WriteString(" |\n")
	for _, row := range body {
		r.b.WriteString("| ")
		r.b.WriteString(strings.Join(row, " | "))
		r.b.WriteString(" |\n")
	}
	r.b.WriteByte('\n')
}

func makeSeparatorRow(n int) []string {
	row := make([]string, n)
	for i := range row {
		row[i] = "---"
	}
	return row
}

func (r *pmMarkdownRenderer) inlineContent(nodes []pmNode) string {
	var b strings.Builder
	for _, node := range nodes {
		switch node.Type {
		case "text":
			text := node.Text
			for _, mark := range node.Marks {
				text = applyMark(text, mark)
			}
			b.WriteString(text)
		case "hardBreak":
			b.WriteString("  \n")
		default:
			b.WriteString(r.inlineContent(node.Content))
		}
	}
	return strings.TrimSpace(b.String())
}

func (r *pmMarkdownRenderer) codeText(nodes []pmNode) string {
	var b strings.Builder
	for _, node := range nodes {
		switch node.Type {
		case "text":
			b.WriteString(node.Text)
		case "hardBreak":
			b.WriteByte('\n')
		default:
			b.WriteString(r.codeText(node.Content))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (r *pmMarkdownRenderer) blockText(nodes []pmNode, depth int) string {
	clone := pmMarkdownRenderer{}
	clone.writeChildren(nodes, depth)
	return strings.TrimSpace(clone.String())
}

func (r *pmMarkdownRenderer) writeBlock(text string) {
	if text == "" {
		return
	}
	r.b.WriteString(text)
	r.b.WriteString("\n\n")
}

func (r *pmMarkdownRenderer) ensureGap() {
	if !strings.HasSuffix(r.b.String(), "\n\n") {
		r.b.WriteByte('\n')
	}
}

func applyMark(text string, mark pmMark) string {
	switch mark.Type {
	case "bold":
		return "**" + text + "**"
	case "italic":
		return "_" + text + "_"
	case "code":
		return "`" + text + "`"
	case "strike":
		return "~~" + text + "~~"
	default:
		return text
	}
}

func attrInt(attrs map[string]any, key string, fallback int) int {
	if attrs == nil {
		return fallback
	}
	switch v := attrs[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func attrString(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if value, ok := attrs[key].(string); ok {
		return value
	}
	return ""
}

func htmlToMarkdown(content string) (string, error) {
	root, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	var b strings.Builder
	renderHTMLNode(&b, root, 0)
	return strings.TrimSpace(b.String()), nil
}

func renderHTMLNode(b *strings.Builder, node *html.Node, depth int) {
	if node == nil {
		return
	}
	if node.Type == html.DocumentNode {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderHTMLNode(b, child, depth)
		}
		return
	}
	if node.Type == html.TextNode {
		text := strings.TrimSpace(node.Data)
		if text != "" {
			b.WriteString(text)
		}
		return
	}
	if node.Type != html.ElementNode {
		return
	}

	switch node.Data {
	case "p":
		text := strings.TrimSpace(innerHTMLText(node))
		if text != "" {
			b.WriteString(text)
			b.WriteString("\n\n")
		}
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := strings.TrimPrefix(node.Data, "h")
		lvl, _ := strconv.Atoi(level)
		b.WriteString(strings.Repeat("#", lvl))
		b.WriteByte(' ')
		b.WriteString(strings.TrimSpace(innerHTMLText(node)))
		b.WriteString("\n\n")
	case "ul":
		renderHTMLList(b, node, depth, false)
		b.WriteByte('\n')
	case "ol":
		renderHTMLList(b, node, depth, true)
		b.WriteByte('\n')
	case "pre":
		b.WriteString("```\n")
		b.WriteString(strings.TrimRight(innerHTMLText(node), "\n"))
		b.WriteString("\n```\n\n")
	case "blockquote":
		lines := strings.Split(strings.TrimSpace(innerHTMLText(node)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			b.WriteString("> ")
			b.WriteString(strings.TrimSpace(line))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	case "table":
		renderHTMLTable(b, node)
	default:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderHTMLNode(b, child, depth)
		}
	}
}

func renderHTMLList(b *strings.Builder, list *html.Node, depth int, ordered bool) {
	index := 1
	for child := list.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "li" {
			continue
		}
		b.WriteString(strings.Repeat("  ", depth))
		if ordered {
			b.WriteString(strconv.Itoa(index))
			b.WriteString(". ")
			index++
		} else {
			b.WriteString("- ")
		}
		b.WriteString(strings.TrimSpace(innerHTMLText(child)))
		b.WriteByte('\n')
	}
}

func renderHTMLTable(b *strings.Builder, table *html.Node) {
	rows := make([][]string, 0)
	for row := table.FirstChild; row != nil; row = row.NextSibling {
		collectHTMLRows(row, &rows)
	}
	if len(rows) == 0 {
		return
	}
	b.WriteString("| ")
	b.WriteString(strings.Join(rows[0], " | "))
	b.WriteString(" |\n| ")
	b.WriteString(strings.Join(makeSeparatorRow(len(rows[0])), " | "))
	b.WriteString(" |\n")
	for _, row := range rows[1:] {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
	}
	b.WriteByte('\n')
}

func collectHTMLRows(node *html.Node, rows *[][]string) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && node.Data == "tr" {
		cells := make([]string, 0)
		for cell := node.FirstChild; cell != nil; cell = cell.NextSibling {
			if cell.Type == html.ElementNode && (cell.Data == "th" || cell.Data == "td") {
				cells = append(cells, strings.TrimSpace(innerHTMLText(cell)))
			}
		}
		if len(cells) > 0 {
			*rows = append(*rows, cells)
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectHTMLRows(child, rows)
	}
}

func innerHTMLText(node *html.Node) string {
	var b bytes.Buffer
	walkHTMLText(&b, node)
	return normalizeWhitespace(b.String())
}

func walkHTMLText(b *bytes.Buffer, node *html.Node) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case html.TextNode:
			b.WriteString(child.Data)
		case html.ElementNode:
			switch child.Data {
			case "br":
				b.WriteByte('\n')
			case "code":
				b.WriteByte('`')
				walkHTMLText(b, child)
				b.WriteByte('`')
			case "strong", "b":
				b.WriteString("**")
				walkHTMLText(b, child)
				b.WriteString("**")
			case "em", "i":
				b.WriteByte('_')
				walkHTMLText(b, child)
				b.WriteByte('_')
			default:
				walkHTMLText(b, child)
			}
		}
	}
}

func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
