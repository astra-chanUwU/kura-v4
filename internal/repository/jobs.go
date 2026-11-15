package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// Jobs forwards the job DAO. Claim maps an empty queue to model.ErrEmpty.
// Complete and Fail are not mapped, so a stale lease stays an error.
type Jobs struct {
	dao *dao.JobDAO
}

func NewJobs(d *dao.JobDAO) *Jobs { return &Jobs{dao: d} }

func (r *Jobs) Enqueue(ctx context.Context, kind string, payload json.RawMessage) (int64, error) {
	return r.dao.Enqueue(ctx, kind, payload)
}

func (r *Jobs) Claim(ctx context.Context, kinds []string) (model.Job, error) {
	job, err := r.dao.Claim(ctx, kinds)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Job{}, model.ErrEmpty
	}
	return job, err
}

func (r *Jobs) Complete(ctx context.Context, job model.Job) error {
	return r.dao.Complete(ctx, job)
}

func (r *Jobs) Fail(ctx context.Context, job model.Job, message string) error {
	return r.dao.Fail(ctx, job, message)
}

func (r *Jobs) ListFailed(ctx context.Context, limit, offset int32) ([]model.FailedJob, error) {
	return r.dao.ListFailed(ctx, limit, offset)
}

func (r *Jobs) CountFailed(ctx context.Context) (int64, error) {
	return r.dao.CountFailed(ctx)
}

func (r *Jobs) RetryFailed(ctx context.Context, id int64) (int64, error) {
	return r.dao.RetryFailed(ctx, id)
}
