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
	repo.EXPECT().ListPostsByOwner(mock.Anything, "u1").Return(want, nil)
	svc := NewService(repo, nil)
	got, err := svc.ListByOwner(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestListByOwnerEmptyIsNonNil(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().ListPostsByOwner(mock.Anything, "u1").Return(nil, nil)
	svc := NewService(repo, nil)
	got, err := svc.ListByOwner(context.Background(), "u1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
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
