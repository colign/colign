package wiki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"

	"github.com/gobenpark/colign/internal/ai"
	"github.com/gobenpark/colign/internal/aiconfig"
	"github.com/gobenpark/colign/internal/events"
	"github.com/gobenpark/colign/internal/models"
)

// AutoUpdateConsumer listens for stage_change events and triggers AI wiki updates
// when a change reaches the "approved" stage.
type AutoUpdateConsumer struct {
	db              *bun.DB
	wikiSvc         *Service
	aiSvc           *ai.Service
	aiConfigSvc     *aiconfig.Service
	hub             *events.Hub
	hocuspocusURL   string
	hocuspocusToken string
	ch              chan events.NotificationEvent
}

// AutoUpdateConsumerConfig holds the configuration for the consumer.
type AutoUpdateConsumerConfig struct {
	DB              *bun.DB
	WikiSvc         *Service
	AISvc           *ai.Service
	AIConfigSvc     *aiconfig.Service
	Hub             *events.Hub
	HocuspocusURL   string
	HocuspocusToken string
}

func NewAutoUpdateConsumer(cfg AutoUpdateConsumerConfig) *AutoUpdateConsumer {
	return &AutoUpdateConsumer{
		db:              cfg.DB,
		wikiSvc:         cfg.WikiSvc,
		aiSvc:           cfg.AISvc,
		aiConfigSvc:     cfg.AIConfigSvc,
		hub:             cfg.Hub,
		hocuspocusURL:   cfg.HocuspocusURL,
		hocuspocusToken: cfg.HocuspocusToken,
		ch:              make(chan events.NotificationEvent, 16),
	}
}

// Start begins consuming notification events in a goroutine.
func (c *AutoUpdateConsumer) Start(ctx context.Context) {
	c.hub.SubscribeNotifications(c.ch)
	go c.run(ctx)
}

func (c *AutoUpdateConsumer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-c.ch:
			if evt.Type != "stage_change" {
				continue
			}
			newStage, _ := evt.Metadata["new_stage"].(string)
			if newStage != "approved" {
				continue
			}
			c.handleApproved(ctx, evt)
		}
	}
}

func (c *AutoUpdateConsumer) handleApproved(ctx context.Context, evt events.NotificationEvent) {
	slog.InfoContext(ctx, "wiki auto-update triggered",
		slog.Int64("change_id", evt.ChangeID),
		slog.Int64("project_id", evt.ProjectID))

	// Resolve AI config
	cfg, err := c.resolveAIConfig(ctx, evt.ProjectID)
	if err != nil {
		slog.WarnContext(ctx, "wiki auto-update: no AI config", slog.String("error", err.Error()))
		return
	}

	// Collect change context
	input, err := c.collectContext(ctx, evt.ChangeID, evt.ProjectID)
	if err != nil {
		slog.WarnContext(ctx, "wiki auto-update: collect context failed", slog.String("error", err.Error()))
		return
	}

	// Generate wiki updates
	updates, err := c.aiSvc.GenerateWikiUpdate(ctx, cfg, input)
	if err != nil {
		slog.WarnContext(ctx, "wiki auto-update: AI generation failed", slog.String("error", err.Error()))
		return
	}

	if len(updates) == 0 {
		slog.InfoContext(ctx, "wiki auto-update: no updates needed", slog.Int64("change_id", evt.ChangeID))
		return
	}

	// Apply updates
	for _, u := range updates {
		switch u.Action {
		case "create":
			if err := c.applyCreate(ctx, evt.ProjectID, u); err != nil {
				slog.WarnContext(ctx, "wiki auto-update: create failed",
					slog.String("title", u.PageTitle), slog.String("error", err.Error()))
			}
		case "update":
			if err := c.applyUpdate(ctx, evt.ProjectID, u); err != nil {
				slog.WarnContext(ctx, "wiki auto-update: update failed",
					slog.String("page_id", u.PageID), slog.String("error", err.Error()))
			}
		}
	}

	slog.InfoContext(ctx, "wiki auto-update completed",
		slog.Int64("change_id", evt.ChangeID),
		slog.Int("updates", len(updates)))
}

func (c *AutoUpdateConsumer) resolveAIConfig(ctx context.Context, projectID int64) (*aiconfig.AIConfig, error) {
	cfg, err := c.aiConfigSvc.ResolveForProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("no AI configuration found")
	}
	return cfg, nil
}

func (c *AutoUpdateConsumer) collectContext(ctx context.Context, changeID, projectID int64) (ai.WikiUpdateInput, error) {
	var input ai.WikiUpdateInput

	// Get change name
	err := c.db.NewSelect().
		TableExpr("changes").
		ColumnExpr("name").
		Where("id = ?", changeID).
		Scan(ctx, &input.ChangeName)
	if err != nil {
		return input, fmt.Errorf("get change: %w", err)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Proposal
	wg.Add(1)
	go func() {
		defer wg.Done()
		var content string
		err := c.db.NewSelect().
			TableExpr("documents").
			ColumnExpr("content").
			Where("change_id = ?", changeID).
			Where("type = 'proposal'").
			Scan(ctx, &content)
		if err == nil && content != "" {
			mu.Lock()
			input.Proposal = content
			mu.Unlock()
		}
	}()

	// Spec
	wg.Add(1)
	go func() {
		defer wg.Done()
		var content string
		err := c.db.NewSelect().
			TableExpr("documents").
			ColumnExpr("content").
			Where("change_id = ?", changeID).
			Where("type = 'spec'").
			Scan(ctx, &content)
		if err == nil && content != "" {
			mu.Lock()
			input.Spec = content
			mu.Unlock()
		}
	}()

	// Tasks
	wg.Add(1)
	go func() {
		defer wg.Done()
		var tasks []struct {
			Title  string `bun:"title"`
			Status string `bun:"status"`
		}
		err := c.db.NewSelect().
			TableExpr("tasks").
			Column("title", "status").
			Where("change_id = ?", changeID).
			OrderExpr("order_index ASC").
			Scan(ctx, &tasks)
		if err == nil && len(tasks) > 0 {
			var sb strings.Builder
			for _, t := range tasks {
				fmt.Fprintf(&sb, "- [%s] %s\n", t.Status, t.Title)
			}
			mu.Lock()
			input.Tasks = sb.String()
			mu.Unlock()
		}
	}()

	// Acceptance Criteria
	wg.Add(1)
	go func() {
		defer wg.Done()
		var criteria []models.AcceptanceCriteria
		err := c.db.NewSelect().
			Model(&criteria).
			Where("change_id = ?", changeID).
			Scan(ctx)
		if err == nil && len(criteria) > 0 {
			var sb strings.Builder
			for _, ac := range criteria {
				metStr := "unmet"
				if ac.Met {
					metStr = "met"
				}
				fmt.Fprintf(&sb, "Scenario: %s [%s]\n", ac.Scenario, metStr)
				for _, step := range ac.Steps {
					fmt.Fprintf(&sb, "  %s %s\n", step.Keyword, step.Text)
				}
			}
			mu.Lock()
			input.Acceptance = sb.String()
			mu.Unlock()
		}
	}()

	// Comments (last 10)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var comments []struct {
			UserName string `bun:"user_name"`
			Body     string `bun:"body"`
		}
		err := c.db.NewSelect().
			TableExpr("comments c").
			ColumnExpr("u.name AS user_name").
			ColumnExpr("c.body").
			Join("JOIN users u ON u.id = c.user_id").
			Where("c.change_id = ?", changeID).
			OrderExpr("c.created_at DESC").
			Limit(10).
			Scan(ctx, &comments)
		if err == nil && len(comments) > 0 {
			var sb strings.Builder
			for i := len(comments) - 1; i >= 0; i-- {
				fmt.Fprintf(&sb, "%s: %s\n", comments[i].UserName, comments[i].Body)
			}
			mu.Lock()
			input.Comments = sb.String()
			mu.Unlock()
		}
	}()

	// Existing wiki pages
	wg.Add(1)
	go func() {
		defer wg.Done()
		pages, err := c.wikiSvc.ListPages(ctx, projectID)
		if err == nil {
			titles := make([]string, len(pages))
			for i, p := range pages {
				titles[i] = p.Title
			}
			mu.Lock()
			input.ExistingPages = titles
			mu.Unlock()
		}
	}()

	wg.Wait()
	return input, nil
}

func (c *AutoUpdateConsumer) applyCreate(ctx context.Context, projectID int64, u ai.WikiPageUpdate) error {
	page, err := c.wikiSvc.CreatePage(ctx, projectID, nil, u.PageTitle, 0)
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}

	return c.syncContent(ctx, projectID, page.ID, u.Content)
}

func (c *AutoUpdateConsumer) applyUpdate(ctx context.Context, projectID int64, u ai.WikiPageUpdate) error {
	pageID, err := uuid.Parse(u.PageID)
	if err != nil {
		return fmt.Errorf("parse page id: %w", err)
	}

	// Update title if provided
	if u.PageTitle != "" {
		if _, err := c.wikiSvc.UpdatePage(ctx, projectID, pageID, &u.PageTitle, nil, nil, nil); err != nil {
			return fmt.Errorf("update title: %w", err)
		}
	}

	return c.syncContent(ctx, projectID, pageID, u.Content)
}

// syncContent resolves wiki links, converts markdown to HTML, sends to Hocuspocus,
// and persists content_text + page links.
func (c *AutoUpdateConsumer) syncContent(ctx context.Context, projectID int64, pageID uuid.UUID, content string) error {
	// Resolve [[Page Title]] links
	resolvedContent, targetIDs := c.resolveWikiLinks(ctx, projectID, content)

	// Convert markdown to HTML
	html, err := mdToHTML(resolvedContent)
	if err != nil {
		return fmt.Errorf("convert markdown: %w", err)
	}

	// Send to Hocuspocus
	if c.hocuspocusURL != "" {
		documentName := fmt.Sprintf("wiki-%s", pageID.String())
		if err := c.sendToHocuspocus(documentName, html); err != nil {
			slog.WarnContext(ctx, "hocuspocus update failed", slog.String("page_id", pageID.String()), slog.String("error", err.Error()))
		}
	}

	// Sync links (always call to clear stale links when all removed)
	if err := c.wikiSvc.SyncLinks(ctx, projectID, pageID, targetIDs); err != nil {
		slog.WarnContext(ctx, "sync links failed", slog.String("page_id", pageID.String()), slog.String("error", err.Error()))
	}

	// Persist content_text
	if _, err := c.wikiSvc.UpdatePage(ctx, projectID, pageID, nil, nil, nil, &content); err != nil {
		slog.WarnContext(ctx, "persist content_text failed", slog.String("page_id", pageID.String()), slog.String("error", err.Error()))
	}

	return nil
}

var wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

func (c *AutoUpdateConsumer) resolveWikiLinks(ctx context.Context, projectID int64, content string) (string, []uuid.UUID) {
	if !strings.Contains(content, "[[") {
		return content, nil
	}

	pages, err := c.wikiSvc.ListPages(ctx, projectID)
	if err != nil {
		return content, nil
	}

	titleToPage := make(map[string]*models.WikiPage, len(pages))
	for _, p := range pages {
		titleToPage[strings.ToLower(p.Title)] = p
	}

	var targetIDs []uuid.UUID
	resolved := wikiLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		title := match[2 : len(match)-2]
		page, ok := titleToPage[strings.ToLower(title)]
		if !ok {
			return match
		}
		targetIDs = append(targetIDs, page.ID)
		return fmt.Sprintf(`<span class="wiki-page-link" data-page-id="%s" data-page-title="%s">%s</span>`,
			page.ID.String(), page.Title, page.Title)
	})

	return resolved, targetIDs
}

func mdToHTML(md string) (string, error) {
	var buf bytes.Buffer
	parser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)
	if err := parser.Convert([]byte(md), &buf); err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}
	return buf.String(), nil
}

func (c *AutoUpdateConsumer) sendToHocuspocus(documentName, htmlContent string) error {
	payload := map[string]string{
		"document_name": documentName,
		"content":       htmlContent,
		"fragment":      "document-store",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.hocuspocusURL+"/api/documents", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.hocuspocusToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("hocuspocus request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hocuspocus returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
