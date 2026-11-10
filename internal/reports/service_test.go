package reports

import (
	"context"
	"database/sql"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type memRepo struct {
	post    model.ReportPostAccess
	missing bool
	open    map[string]bool
	stored  []model.PostReport
}

func (m *memRepo) ReportPostAccess(context.Context, string) (model.ReportPostAccess, error) {
	if m.missing {
		return model.ReportPostAccess{}, sql.ErrNoRows
	}
	return m.post, nil
}
func (m *memRepo) GetOpenPostReport(_ context.Context, postID, reporterID string) (bool, error) {
	return m.open[postID+"/"+reporterID], nil
}
func (m *memRepo) CreatePostReport(_ context.Context, postID, reporterID, reason, note string) (model.PostReport, error) {
	report := model.PostReport{ID: "r1", PostID: postID, ReporterID: reporterID, Reason: reason, Note: note, Status: "open"}
	m.stored = append(m.stored, report)
	m.open[postID+"/"+reporterID] = true
	return report, nil
}
func (m *memRepo) ListOpenPostReports(context.Context, int32, int32) ([]model.PostReport, error) {
	return m.stored, nil
}
func (m *memRepo) CountOpenPostReports(context.Context) (int64, error) {
	return int64(len(m.stored)), nil
}
func (m *memRepo) DismissPostReport(_ context.Context, id, resolverID string) (model.PostReport, error) {
	for i, report := range m.stored {
		if report.ID == id && report.Status == "open" {
			report.Status = "dismissed"
			report.ResolverID = resolverID
			m.stored[i] = report
			return report, nil
		}
	}
	return model.PostReport{}, sql.ErrNoRows
}

func TestCreateRejectsVisitorAndDuplicateAndHidden(t *testing.T) {
	repo := &memRepo{post: model.ReportPostAccess{Rating: "sfw"}, open: map[string]bool{}}
	svc := NewService(repo)
	_, err := svc.Create(context.Background(), model.User{}, "p1", "spam", "")
	require.ErrorIs(t, err, ErrForbidden)

	member := model.User{ID: "m1", Role: model.RoleMember}
	_, err = svc.Create(context.Background(), member, "p1", "spam", "")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), member, "p1", "spam", "")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)

	hidden := NewService(&memRepo{post: model.ReportPostAccess{Hidden: true, OwnerID: "someone-else", Rating: "sfw"}, open: map[string]bool{}})
	_, err = hidden.Create(context.Background(), member, "p1", "other", "")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDismissRequiresModerator(t *testing.T) {
	repo := &memRepo{stored: []model.PostReport{{ID: "r1", Status: "open"}}}
	svc := NewService(repo)
	_, err := svc.Dismiss(context.Background(), model.User{ID: "m1", Role: model.RoleMember}, "r1")
	require.ErrorIs(t, err, ErrForbidden)
	got, err := svc.Dismiss(context.Background(), model.User{ID: "mod", Role: model.RoleModerator}, "r1")
	require.NoError(t, err)
	require.Equal(t, "dismissed", got.Status)
	require.Equal(t, "mod", got.ResolverID)
}
