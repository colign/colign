package archive

import (
	"context"
	"log/slog"
	"time"

	"github.com/gobenpark/colign/internal/models"
)

const cronInterval = 24 * time.Hour

// StartCron starts a background goroutine that periodically scans for changes
// eligible for auto-archive. It stops when ctx is cancelled.
func (s *Service) StartCron(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cronInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("archive cron stopped")
				return
			case <-ticker.C:
				s.runAutoArchiveScan(ctx)
			}
		}
	}()
	slog.Info("archive cron started", "interval", cronInterval)
}

func (s *Service) runAutoArchiveScan(ctx context.Context) {
	var changes []models.Change
	err := s.db.NewSelect().Model(&changes).
		Where("stage = ?", models.StageReady).
		Where("archived_at IS NULL").
		Scan(ctx)
	if err != nil {
		slog.Error("archive cron: failed to list ready changes", "error", err)
		return
	}

	for _, change := range changes {
		archived, err := s.EvaluateAutoArchive(ctx, change.ID)
		if err != nil {
			slog.Error("archive cron: evaluation failed", "error", err, "change_id", change.ID)
			continue
		}
		if archived {
			slog.Info("archive cron: auto-archived change", "change_id", change.ID)
		}
	}
}
