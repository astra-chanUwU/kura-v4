package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/storage"
)

// sweepAbandonedUploads deletes .partial files older than an hour, once
// immediately and then every 15 minutes, until ctx is cancelled.
func sweepAbandonedUploads(ctx context.Context, dir string, log *slog.Logger) {
	sweep := func() {
		n, err := storage.SweepPartials(dir, time.Hour)
		if err != nil {
			log.Error("sweep partial uploads", "error", err)
			return
		}
		if n > 0 {
			log.Info("removed abandoned uploads", "count", n)
		}
	}
	sweep()
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}
