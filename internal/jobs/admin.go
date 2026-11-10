package jobs

import (
	"context"
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("you don't have permission to do that")
)

// ValidationError is a rejected jobs request, safe to show to staff.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return &ValidationError{Message: msg} }

// Repository is the failed-job view. The worker queue stays on the DAO.
type Repository interface {
	ListFailed(ctx context.Context, limit, offset int32) ([]model.FailedJob, error)
	CountFailed(ctx context.Context) (int64, error)
	RetryFailed(ctx context.Context, id int64) (int64, error)
}

// Service lists and requeues exhausted jobs for moderators.
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// ListFailed returns one page of failed jobs. Moderators and above only.
// state is empty or "failed"; any other state is rejected.
func (s *Service) ListFailed(ctx context.Context, actor model.User, state string, limit, offset int) (model.FailedJobPage, error) {
	if state != "" && state != "failed" {
		return model.FailedJobPage{}, invalid("Only failed jobs can be listed.")
	}
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.FailedJobPage{}, ErrForbidden
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountFailed(ctx)
	if err != nil {
		return model.FailedJobPage{}, err
	}
	jobs, err := s.repo.ListFailed(ctx, lim, off)
	if err != nil {
		return model.FailedJobPage{}, err
	}
	if jobs == nil {
		jobs = []model.FailedJob{}
	}
	return model.FailedJobPage{Total: total, Jobs: jobs}, nil
}

// Retry puts a failed job back on the queue. A job in any other state is
// not found, so a queued or running job cannot be reset.
func (s *Service) Retry(ctx context.Context, actor model.User, id int64) error {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return ErrForbidden
	}
	n, err := s.repo.RetryFailed(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
