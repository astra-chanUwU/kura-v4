package jobs

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type fakeJobs struct {
	state string
}

func (f *fakeJobs) ListFailed(context.Context, int32, int32) ([]model.FailedJob, error) {
	return nil, nil
}
func (f *fakeJobs) CountFailed(context.Context) (int64, error) { return 0, nil }
func (f *fakeJobs) RetryFailed(context.Context, int64) (int64, error) {
	if f.state != "failed" {
		return 0, nil
	}
	return 1, nil
}

func TestRetryRejectsJobThatIsNotFailed(t *testing.T) {
	svc := NewService(&fakeJobs{state: "queued"})
	actor := model.User{Role: model.RoleModerator}
	err := svc.Retry(context.Background(), actor, 7)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRetryRequeuesFailedJob(t *testing.T) {
	svc := NewService(&fakeJobs{state: "failed"})
	err := svc.Retry(context.Background(), model.User{Role: model.RoleModerator}, 7)
	require.NoError(t, err)
}

func TestRetryRequiresModerator(t *testing.T) {
	svc := NewService(&fakeJobs{state: "failed"})
	err := svc.Retry(context.Background(), model.User{Role: model.RoleMember}, 7)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestListFailedRejectsStateOtherThanFailed(t *testing.T) {
	svc := NewService(&fakeJobs{})
	for _, role := range []model.Role{model.RoleModerator, model.RoleMember} {
		_, err := svc.ListFailed(context.Background(), model.User{Role: role}, "queued", 40, 0)
		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		require.Equal(t, "Only failed jobs can be listed.", vErr.Message)
	}
}

func TestListFailedRequiresModerator(t *testing.T) {
	svc := NewService(&fakeJobs{})
	_, err := svc.ListFailed(context.Background(), model.User{Role: model.RoleMember}, "failed", 40, 0)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestListFailedAcceptsEmptyOrFailedState(t *testing.T) {
	svc := NewService(&fakeJobs{})
	actor := model.User{Role: model.RoleModerator}
	page, err := svc.ListFailed(context.Background(), actor, "", 40, 0)
	require.NoError(t, err)
	require.Empty(t, page.Jobs)
	_, err = svc.ListFailed(context.Background(), actor, "failed", 40, 0)
	require.NoError(t, err)
}
