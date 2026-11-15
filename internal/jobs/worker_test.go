package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type scriptQueue struct {
	mu          sync.Mutex
	claims      int
	job         model.Job
	claimErr    error
	completeErr error
	failErr     error
	completed   []model.Job
	failed      []model.Job
}

func (q *scriptQueue) Claim(context.Context, []string) (model.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.claims++
	if q.job.ID != 0 && q.claims == 1 {
		return q.job, nil
	}
	if q.claimErr != nil {
		return model.Job{}, q.claimErr
	}
	return model.Job{}, model.ErrEmpty
}

func (q *scriptQueue) Complete(_ context.Context, job model.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.completed = append(q.completed, job)
	return q.completeErr
}

func (q *scriptQueue) Fail(_ context.Context, job model.Job, _ string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.failed = append(q.failed, job)
	return q.failErr
}

func (q *scriptQueue) snapshot() (claims int, completed, failed []model.Job) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.claims, append([]model.Job(nil), q.completed...), append([]model.Job(nil), q.failed...)
}

func runUntil(t *testing.T, ctx context.Context, cancel context.CancelFunc, q *scriptQueue, ready func() bool, log *slog.Logger, handler Handler) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		Run(ctx, q, map[string]Handler{"post.derive": handler}, 1, log)
	}()
	require.Eventually(t, ready, 5*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop")
	}
}

func TestRunDoesNotTreatDatabaseOutageAsEmptyQueue(t *testing.T) {
	outage := errors.New("database unavailable")
	q := &scriptQueue{claimErr: fmt.Errorf("dial postgres: %w", outage)}
	var calls int
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	runUntil(t, ctx, cancel, q, func() bool {
		claims, _, _ := q.snapshot()
		return claims > 0
	}, log, func(context.Context, json.RawMessage) error {
		calls++
		return nil
	})
	_, completed, failed := q.snapshot()
	require.Zero(t, calls)
	require.Empty(t, completed)
	require.Empty(t, failed)
	require.Contains(t, buf.String(), "claim job")
	require.Contains(t, buf.String(), "database unavailable")
}

func TestRunIdlesWhenQueueIsEmpty(t *testing.T) {
	q := &scriptQueue{claimErr: fmt.Errorf("idle: %w", model.ErrEmpty)}
	var calls int
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	runUntil(t, ctx, cancel, q, func() bool {
		claims, _, _ := q.snapshot()
		return claims > 0
	}, log, func(context.Context, json.RawMessage) error {
		calls++
		return nil
	})
	require.Zero(t, calls)
	require.NotContains(t, buf.String(), "claim job")
}

func TestRunStaleWorkerCannotFinishJobItNoLongerOwns(t *testing.T) {
	owned := model.Job{ID: 9, Kind: "post.derive", Token: "lease-a", Payload: json.RawMessage(`{"post_id":"p"}`)}
	stale := errors.New("stale lease")
	q := &scriptQueue{job: owned, completeErr: stale}
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	runUntil(t, ctx, cancel, q, func() bool {
		_, completed, _ := q.snapshot()
		return len(completed) > 0
	}, log, func(context.Context, json.RawMessage) error { return nil })
	_, completed, failed := q.snapshot()
	require.Equal(t, []model.Job{owned}, completed)
	require.Empty(t, failed)
	require.Contains(t, buf.String(), "save job outcome")
	require.Contains(t, buf.String(), "stale lease")
}

func TestRunStaleFailureKeepsTheLeaseToken(t *testing.T) {
	owned := model.Job{ID: 9, Kind: "post.derive", Token: "lease-a", Payload: json.RawMessage(`{}`)}
	q := &scriptQueue{job: owned, failErr: errors.New("stale lease")}
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	runUntil(t, ctx, cancel, q, func() bool {
		_, _, failed := q.snapshot()
		return len(failed) > 0
	}, log, func(context.Context, json.RawMessage) error { return errors.New("handler failed") })
	_, completed, failed := q.snapshot()
	require.Empty(t, completed)
	require.Equal(t, []model.Job{owned}, failed)
	require.Contains(t, buf.String(), "save job outcome")
}
