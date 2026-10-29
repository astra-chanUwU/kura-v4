package board

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListByOwnerReturnsOwnPosts(t *testing.T) {
	repo := NewMockRepository(t)
	want := []model.Post{{ID: "p2", OriginalName: "b.png"}, {ID: "p1", OriginalName: "a.png"}}
	repo.EXPECT().CountVisiblePostsByOwner(mock.Anything, "u1", false).Return(int64(2), nil)
	repo.EXPECT().ListPostsByOwner(mock.Anything, "u1", false, int32(40), int32(0)).Return(want, nil)
	svc := NewService(repo, nil)
	got, err := svc.ListByOwner(context.Background(), "u1", false, 0, 0)
	require.NoError(t, err)
	require.Equal(t, want, got.Posts)
	require.Equal(t, int64(2), got.Total)
}

func TestListByOwnerEmptyIsNonNil(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().CountVisiblePostsByOwner(mock.Anything, "u1", false).Return(int64(0), nil)
	repo.EXPECT().ListPostsByOwner(mock.Anything, "u1", false, int32(40), int32(0)).Return(nil, nil)
	svc := NewService(repo, nil)
	got, err := svc.ListByOwner(context.Background(), "u1", false, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, got.Posts)
	require.Empty(t, got.Posts)
}

func TestSearchCursorIgnoresOffset(t *testing.T) {
	repo := NewMockRepository(t)
	id := "11111111-1111-1111-1111-111111111111"
	when := model.Post{ID: id}.CreatedAt
	repo.EXPECT().GetPost(mock.Anything, id).Return(model.Post{ID: id, CreatedAt: when}, nil)
	repo.EXPECT().CountBoardMatches(mock.Anything, mock.Anything).Return(int64(1), nil)
	repo.EXPECT().SearchPosts(mock.Anything, mock.MatchedBy(func(q model.PostQuery) bool {
		return q.AfterID == id && q.Offset == 0 && q.AfterAt.Equal(when)
	})).Return([]model.Post{{ID: "next"}}, nil)
	got, err := NewService(repo, nil).Search(context.Background(), model.PostQuery{AfterID: id, Offset: 50})
	require.NoError(t, err)
	require.Equal(t, "next", got.Posts[0].ID)
}

func TestSearchRejectsBadCursor(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo, nil).Search(context.Background(), model.PostQuery{AfterID: "nope"})
	require.ErrorIs(t, err, ErrBadCursor)
}

func TestSetMetaAsWritesHistoryOnlyWhenChanged(t *testing.T) {
	actor := model.User{ID: "u1", Username: "tagger", Role: model.RoleMember}
	existing := model.Post{ID: "p1", Artist: "old", Tags: []string{"keep"}, Rating: "sfw"}
	repo := NewMockRepository(t)
	repo.EXPECT().GetPost(mock.Anything, "p1").Return(existing, nil)
	repo.EXPECT().ReplacePostMetaAudited(mock.Anything, mock.MatchedBy(func(p model.Post) bool {
		return p.Artist == "new"
	}), mock.MatchedBy(func(e model.ChangeEntry) bool {
		b, _ := e.PostBefore()
		a, _ := e.PostAfter()
		return e.Action == "post.meta_updated" && b.Artist == "old" && a.Artist == "new" && e.ActorUsername == "tagger"
	})).Return(nil)
	svc := NewService(repo, nil)
	// Get() may probe dimensions via media (nil media skips it).
	got, err := svc.SetMetaAs(context.Background(), &actor, "p1", "new", []string{"keep"}, "sfw")
	require.NoError(t, err)
	require.Equal(t, "new", got.Artist)
}
