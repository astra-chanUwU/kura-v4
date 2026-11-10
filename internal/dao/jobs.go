package dao

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type JobDAO struct{ queries *sqlcgen.Queries }

func NewJobs(conn *sql.DB) *JobDAO { return &JobDAO{queries: sqlcgen.New(conn)} }
func (d *JobDAO) Enqueue(ctx context.Context, kind string, payload json.RawMessage) (int64, error) {
	return d.queries.EnqueueJob(ctx, sqlcgen.EnqueueJobParams{Kind: kind, Payload: payload})
}
func (d *JobDAO) Claim(ctx context.Context, kinds []string) (model.Job, error) {
	if err := d.queries.ReapExhaustedJobs(ctx); err != nil {
		return model.Job{}, err
	}
	token := uuid.NewString()
	row, err := d.queries.ClaimJob(ctx, sqlcgen.ClaimJobParams{Kinds: kinds, Token: sql.NullString{String: token, Valid: true}})
	if err != nil {
		return model.Job{}, err
	}
	return model.Job{ID: row.ID, Kind: row.Kind, Payload: row.Payload, Token: token}, nil
}
func (d *JobDAO) Complete(ctx context.Context, job model.Job) error {
	rows, err := d.queries.CompleteJob(ctx, sqlcgen.CompleteJobParams{ID: job.ID, LeaseToken: sql.NullString{String: job.Token, Valid: true}})
	if err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return err
}
func (d *JobDAO) Fail(ctx context.Context, job model.Job, message string) error {
	rows, err := d.queries.FailJob(ctx, sqlcgen.FailJobParams{ID: job.ID, Token: sql.NullString{String: job.Token, Valid: true}, ErrorMessage: sql.NullString{String: message, Valid: true}})
	if err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (d *JobDAO) ListFailed(ctx context.Context, limit, offset int32) ([]model.FailedJob, error) {
	rows, err := d.queries.ListFailedJobs(ctx, sqlcgen.ListFailedJobsParams{RowLimit: limit, RowOffset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]model.FailedJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, model.FailedJob{
			ID: row.ID, Kind: row.Kind, Attempts: row.Attempts, MaxAttempts: row.MaxAttempts,
			LastError: row.LastError.String, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (d *JobDAO) CountFailed(ctx context.Context) (int64, error) {
	return d.queries.CountFailedJobs(ctx)
}

func (d *JobDAO) RetryFailed(ctx context.Context, id int64) (int64, error) {
	return d.queries.RetryFailedJob(ctx, id)
}
