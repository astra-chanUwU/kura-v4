package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Queue interface {
	Claim(context.Context, []string) (model.Job, error)
	Complete(context.Context, model.Job) error
	Fail(context.Context, model.Job, string) error
}
type Handler func(context.Context, json.RawMessage) error

// Run bounds concurrency and only claims registered kinds. Jobs are delivered
// at least once; handlers must be idempotent and honor cancellation. The 45s
// deadline is shorter than the 60s database lease. Longer jobs need renewal.
func Run(ctx context.Context, queue Queue, handlers map[string]Handler, concurrency int, log *slog.Logger) {
	kinds := make([]string, 0, len(handlers))
	for kind := range handlers {
		kinds = append(kinds, kind)
	}
	if len(kinds) == 0 {
		<-ctx.Done()
		return
	}
	var wg sync.WaitGroup
	for range max(1, concurrency) {
		wg.Go(func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				job, err := queue.Claim(ctx, kinds)
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				if err != nil {
					log.Error("claim job", "error", err)
					continue
				}
				jobCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
				err = invoke(jobCtx, handlers[job.Kind], job.Payload)
				if err == nil {
					err = jobCtx.Err()
				}
				cancel()
				finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err != nil {
					log.Error("job failed", "id", job.ID, "kind", job.Kind, "error", err)
					// Store a bounded public-safe reason; detailed errors stay in logs.
					err = queue.Fail(finishCtx, job, "job execution failed")
				} else {
					err = queue.Complete(finishCtx, job)
				}
				finishCancel()
				if err != nil {
					log.Error("save job outcome", "id", job.ID, "error", err)
				}
			}
		})
	}
	wg.Wait()
}

func invoke(ctx context.Context, handler Handler, payload json.RawMessage) (err error) {
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("job handler panicked")
		}
	}()
	return handler(ctx, payload)
}
