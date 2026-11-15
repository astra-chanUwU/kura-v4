package dao

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type execDB struct {
	query string
	args  []any
	n     int64
	err   error
}

func (e *execDB) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.query = query
	e.args = append([]any(nil), args...)
	if e.err != nil {
		return nil, e.err
	}
	return rowsResult(e.n), nil
}

func (e *execDB) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("not used")
}
func (e *execDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("not used")
}
func (e *execDB) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }

type rowsResult int64

func (r rowsResult) LastInsertId() (int64, error) { return 0, errors.New("no id") }
func (r rowsResult) RowsAffected() (int64, error) { return int64(r), nil }

func TestCompleteAndFailRejectStaleLease(t *testing.T) {
	job := model.Job{ID: 4, Token: "lease-a"}
	for _, tc := range []struct {
		name string
		call func(*JobDAO) error
	}{
		{"complete", func(d *JobDAO) error { return d.Complete(context.Background(), job) }},
		{"fail", func(d *JobDAO) error { return d.Fail(context.Background(), job, "job execution failed") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &execDB{}
			err := tc.call(&JobDAO{queries: sqlcgen.New(db)})
			require.ErrorIs(t, err, sql.ErrNoRows)
			require.Contains(t, db.query, "lease_token")
			require.Contains(t, db.query, "state = 'running'")
			var token sql.NullString
			switch arg := db.args[len(db.args)-1].(type) {
			case sql.NullString:
				token = arg
			default:
				t.Fatalf("lease token arg = %T", db.args[len(db.args)-1])
			}
			require.True(t, token.Valid)
			require.Equal(t, "lease-a", token.String)
		})
	}
}

func TestCompleteKeepsOwnerLease(t *testing.T) {
	db := &execDB{n: 1}
	err := (&JobDAO{queries: sqlcgen.New(db)}).Complete(context.Background(), model.Job{ID: 4, Token: "lease-a"})
	require.NoError(t, err)
	require.Equal(t, int64(4), db.args[0])
}

func TestCompleteReturnsDatabaseError(t *testing.T) {
	outage := errors.New("database unavailable")
	db := &execDB{err: outage}
	err := (&JobDAO{queries: sqlcgen.New(db)}).Complete(context.Background(), model.Job{ID: 4, Token: "lease-a"})
	require.ErrorIs(t, err, outage)
	require.NotErrorIs(t, err, sql.ErrNoRows)
	require.NotErrorIs(t, err, model.ErrEmpty)
}

func TestRetryFailedOnlyUpdatesFailedRows(t *testing.T) {
	db := &execDB{}
	n, err := (&JobDAO{queries: sqlcgen.New(db)}).RetryFailed(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
	require.Contains(t, db.query, "state = 'failed'")
	require.Equal(t, int64(9), db.args[0])

	db.n = 1
	n, err = (&JobDAO{queries: sqlcgen.New(db)}).RetryFailed(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}
