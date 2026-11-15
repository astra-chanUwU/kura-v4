package agentcli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/reports"
	"github.com/stretchr/testify/require"
)

type fakeReports struct {
	created bool
}

func (f *fakeReports) Create(_ context.Context, viewer model.User, _, _, _ string) (model.PostReport, error) {
	if viewer.ID == "" {
		return model.PostReport{}, reports.ErrForbidden
	}
	f.created = true
	return model.PostReport{ID: "r1", PostID: "p1", Reason: "spam", Status: "open"}, nil
}
func (f *fakeReports) ListOpen(context.Context, model.User, int, int) (model.PostReportPage, error) {
	return model.PostReportPage{Total: 0, Reports: []model.PostReport{}}, nil
}
func (f *fakeReports) Dismiss(context.Context, model.User, string) (model.PostReport, error) {
	return model.PostReport{}, reports.ErrForbidden
}

func TestReportsCreateRequiresMember(t *testing.T) {
	code, env, _ := postsRun(t, Deps{Reports: &fakeReports{}}, []string{"reports", "create", "p1", "--reason", "spam"}, filepath.Join(t.TempDir(), "none"))
	require.Equal(t, 1, code)
	require.Equal(t, "unauthenticated", env.Error.Code)

	session, accounts := postsSession(t, model.RoleMember)
	store := &fakeReports{}
	code, env, _ = postsRun(t, Deps{Accounts: accounts, Reports: store}, []string{"reports", "create", "p1", "--reason", "spam"}, session)
	require.Equal(t, 0, code)
	require.True(t, store.created)
	require.Equal(t, "reports.create", env.Command)
}

func TestReportsListUsageAndRole(t *testing.T) {
	session, accounts := postsSession(t, model.RoleMember)
	code, env, _ := postsRun(t, Deps{Accounts: accounts, Reports: &fakeReports{}}, []string{"reports", "list"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)

	session, accounts = postsSession(t, model.RoleModerator)
	code, env, _ = postsRun(t, Deps{Accounts: accounts, Reports: &fakeReports{}}, []string{"reports", "list", "extra"}, session)
	require.Equal(t, 2, code)
	require.Equal(t, "usage", env.Error.Code)

	code, env, _ = postsRun(t, Deps{Accounts: accounts, Reports: &fakeReports{}}, []string{"reports", "list", "--limit", "0"}, session)
	require.Equal(t, 2, code)
	require.Equal(t, "usage", env.Error.Code)
	require.Equal(t, "reports.list", env.Command)
}

func TestReportsDismissRequiresModerator(t *testing.T) {
	session, accounts := postsSession(t, model.RoleMember)
	code, env, _ := postsRun(t, Deps{Accounts: accounts, Reports: &fakeReports{}}, []string{"reports", "dismiss", "r1"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
}
