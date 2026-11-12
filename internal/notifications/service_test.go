package notifications

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type notice struct {
	recipientID string
	actorID     string
	kind        string
	postID      string
	commentID   string
}

type row struct {
	id          string
	recipientID string
	read        bool
}

type memRepo struct {
	created []notice
	rows    []row
}

func (m *memRepo) Create(_ context.Context, recipientID, actorID, kind, postID, commentID string) error {
	m.created = append(m.created, notice{recipientID, actorID, kind, postID, commentID})
	return nil
}

func (m *memRepo) List(context.Context, string, int32, int32) ([]model.Notification, error) {
	return nil, nil
}

func (m *memRepo) Count(context.Context, string) (int64, error) { return 0, nil }

func (m *memRepo) CountUnread(context.Context, string) (int64, error) { return 0, nil }

func (m *memRepo) MarkRead(_ context.Context, recipientID string, ids []string) error {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	for i := range m.rows {
		if m.rows[i].recipientID == recipientID && want[m.rows[i].id] {
			m.rows[i].read = true
		}
	}
	return nil
}

func (m *memRepo) MarkAllRead(_ context.Context, recipientID string) error {
	for i := range m.rows {
		if m.rows[i].recipientID == recipientID {
			m.rows[i].read = true
		}
	}
	return nil
}

func (m *memRepo) PostOwner(context.Context, string) (string, error) { return "", nil }

func TestNotifySkipsSelfAndEmptyRecipient(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo)
	ctx := context.Background()

	require.NoError(t, svc.Notify(ctx, "owner", "owner", model.NotificationFavorite, "post-1", ""))
	require.NoError(t, svc.Notify(ctx, "", "fan", model.NotificationFavorite, "post-1", ""))
	require.NoError(t, svc.Notify(ctx, "   ", "fan", model.NotificationComment, "post-1", "c1"))
	require.Empty(t, repo.created)
}

func TestNotifyWritesFavoriteAndComment(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo)
	ctx := context.Background()

	require.NoError(t, svc.Notify(ctx, "owner", "fan", model.NotificationFavorite, "post-1", "ignored"))
	require.NoError(t, svc.Notify(ctx, "owner", "fan", model.NotificationComment, "post-1", "comment-1"))
	require.Len(t, repo.created, 2)
	require.Equal(t, model.NotificationFavorite, repo.created[0].kind)
	require.Equal(t, "owner", repo.created[0].recipientID)
	require.Equal(t, "fan", repo.created[0].actorID)
	require.Equal(t, "post-1", repo.created[0].postID)
	require.Empty(t, repo.created[0].commentID)
	require.Equal(t, model.NotificationComment, repo.created[1].kind)
	require.Equal(t, "comment-1", repo.created[1].commentID)
	require.Equal(t, "post-1", repo.created[1].postID)

	err := svc.Notify(ctx, "owner", "fan", model.NotificationComment, "post-1", "")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Len(t, repo.created, 2)
}

func TestMarkRead(t *testing.T) {
	repo := &memRepo{rows: []row{
		{id: "n1", recipientID: "owner"},
		{id: "n2", recipientID: "owner"},
		{id: "n3", recipientID: "other"},
	}}
	svc := NewService(repo)
	ctx := context.Background()

	require.NoError(t, svc.MarkRead(ctx, "owner", []string{"n1"}))
	require.True(t, repo.rows[0].read)
	require.False(t, repo.rows[1].read)
	require.False(t, repo.rows[2].read)

	require.NoError(t, svc.MarkRead(ctx, "owner", []string{"n3"}))
	require.False(t, repo.rows[2].read)

	require.NoError(t, svc.MarkRead(ctx, "owner", nil))
	require.True(t, repo.rows[0].read)
	require.True(t, repo.rows[1].read)
	require.False(t, repo.rows[2].read)

	require.ErrorIs(t, svc.MarkRead(ctx, "", []string{"n3"}), ErrForbidden)
	require.False(t, repo.rows[2].read)
}
