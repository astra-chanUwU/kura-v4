//go:build integration

package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresContracts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := postgres.Run(ctx, "postgres:18-alpine", postgres.WithDatabase("kura_test"), postgres.WithUsername("test"), postgres.WithPassword("test"), postgres.BasicWaitStrategies())
	require.NoError(t, err, "Docker must be running for disposable database tests")
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	url, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	conn, err := db.Open(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Migrate(ctx, conn))
	require.NoError(t, db.Migrate(ctx, conn), "migrations are repeatable")
	info, err := dao.NewSite(conn).Get(ctx)
	require.NoError(t, err)
	require.Equal(t, "Kura", info.Name)

	queue := dao.NewJobs(conn)
	id, err := queue.Enqueue(ctx, "test", json.RawMessage(`{"key":"original"}`))
	require.NoError(t, err)
	var wg sync.WaitGroup
	claimed := make(chan model.Job, 2)
	failures := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			job, err := queue.Claim(ctx, []string{"test"})
			if err == nil {
				claimed <- job
			} else {
				failures <- err
			}
		})
	}
	wg.Wait()
	require.Len(t, claimed, 1, "two workers must not lease the same job")
	require.Len(t, failures, 1)
	require.ErrorIs(t, <-failures, sql.ErrNoRows)
	first := <-claimed
	require.Equal(t, id, first.ID)
	_, err = conn.ExecContext(ctx, "UPDATE jobs SET lease_until = now() - interval '1 second' WHERE id = $1", id)
	require.NoError(t, err)
	restartedWorker := dao.NewJobs(conn)
	second, err := restartedWorker.Claim(ctx, []string{"test"})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.NotEqual(t, first.Token, second.Token)
	require.ErrorIs(t, queue.Complete(ctx, first), sql.ErrNoRows, "stale worker must not acknowledge a new lease")
	require.NoError(t, restartedWorker.Fail(ctx, second, "retry"))
	_, err = conn.ExecContext(ctx, "UPDATE jobs SET available_at = now() WHERE id = $1", id)
	require.NoError(t, err)
	third, err := queue.Claim(ctx, []string{"test"})
	require.NoError(t, err)
	require.NoError(t, queue.Fail(ctx, third, "final failure"))
	_, err = queue.Claim(ctx, []string{"test"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	var state string
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT state FROM jobs WHERE id = $1", id).Scan(&state))
	require.Equal(t, "failed", state)

	require.NoError(t, goose.DownToContext(ctx, conn, "migrations", 0))
	require.NoError(t, db.Migrate(ctx, conn), "fresh schema can be rebuilt after rollback")
}
