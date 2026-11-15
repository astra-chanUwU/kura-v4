package board

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestSearchAliasExcludeWins(t *testing.T) {
	repo := &fakeRepo{aliases: map[string]string{"grey_hair": "gray_hair"}}
	_, err := NewService(repo, nil).Search(context.Background(), model.PostQuery{
		Tags:    []string{"grey_hair"},
		Exclude: []string{"gray_hair"},
	})
	require.NoError(t, err)
	require.Empty(t, repo.lastQuery.Tags)
	require.Equal(t, []string{"gray_hair"}, repo.lastQuery.Exclude)
}

func TestSetMetaImplicationFollowsAlias(t *testing.T) {
	repo := &fakeRepo{
		post:    model.Post{ID: "p1", Rating: "sfw"},
		aliases: map[string]string{"grey_hair": "gray_hair"},
		edges: []model.TagImplication{
			{Antecedent: "cat", Consequent: "grey_hair"},
			{Antecedent: "grey_hair", Consequent: "long_hair"},
		},
	}
	got, err := NewService(repo, nil).SetMeta(context.Background(), "p1", "", []string{"cat"}, "sfw", "")
	require.NoError(t, err)
	require.Equal(t, []string{"cat", "gray_hair", "long_hair"}, got.Tags)
	require.NotContains(t, got.Tags, "grey_hair")
}

func TestSetMetaArtistSurvivesAliasImplication(t *testing.T) {
	repo := &fakeRepo{
		post:    model.Post{ID: "p1", Rating: "sfw"},
		aliases: map[string]string{"grey_hair": "gray_hair"},
		edges: []model.TagImplication{
			{Antecedent: "grey_hair", Consequent: "long_hair"},
		},
	}
	got, err := NewService(repo, nil).SetMeta(context.Background(), "p1", "Grey Hair", []string{"cat"}, "sfw", "")
	require.NoError(t, err)
	require.Equal(t, []string{"cat", "gray_hair", "long_hair"}, got.Tags)
	require.Equal(t, []string{"gray_hair"}, repo.artistPages)
	require.NotContains(t, got.Tags, "grey_hair")
}

func TestSearchDropsPostsTheViewerCannotSee(t *testing.T) {
	repo := &fakeRepo{
		found: []model.Post{
			{ID: "sfw", Rating: "sfw"},
			{ID: "nsfw", Rating: "nsfw"},
			{ID: "hidden", Hidden: true, Rating: "sfw"},
			{ID: "deleted", Deleted: true, Rating: "sfw"},
		},
		foundTotal: 4,
	}
	svc := NewService(repo, nil)
	page, err := svc.Search(context.Background(), model.PostQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, []string{"sfw"}, postIDs(page.Posts))

	page, err = svc.Search(context.Background(), model.PostQuery{IncludeNSFW: true})
	require.NoError(t, err)
	require.Equal(t, int64(2), page.Total)
	require.Equal(t, []string{"sfw", "nsfw"}, postIDs(page.Posts))
}

func TestRelatedAndProfileDropPostsTheViewerCannotSee(t *testing.T) {
	rows := []model.Post{
		{ID: "sfw", Rating: "sfw"},
		{ID: "nsfw", Rating: "nsfw"},
		{ID: "hidden", Hidden: true, Rating: "sfw"},
		{ID: "deleted", Deleted: true, Rating: "sfw"},
	}
	repo := &fakeRepo{related: rows, publicPosts: rows, publicTotal: 4}
	svc := NewService(repo, nil)

	related, err := svc.Related(context.Background(), "anchor", false, nil, 12)
	require.NoError(t, err)
	require.Equal(t, []string{"sfw"}, postIDs(related))
	related, err = svc.Related(context.Background(), "anchor", true, nil, 12)
	require.NoError(t, err)
	require.Equal(t, []string{"sfw", "nsfw"}, postIDs(related))

	posts, total, err := svc.PublicByOwner(context.Background(), "u", false, 12, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, []string{"sfw"}, postIDs(posts))
	require.Equal(t, int32(12), repo.publicLimit)
	require.Equal(t, int32(0), repo.publicOffset)
	posts, total, err = svc.PublicByOwner(context.Background(), "u", true, 12, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, []string{"sfw", "nsfw"}, postIDs(posts))
	require.Equal(t, int32(0), repo.publicOffset)
}

func TestRandomSkipsPostsTheViewerCannotSee(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	for _, post := range []model.Post{
		{ID: "nsfw", Rating: "nsfw"},
		{ID: "hidden", Hidden: true, Rating: "sfw"},
		{ID: "deleted", Deleted: true, Rating: "sfw"},
	} {
		repo.random = post
		_, err := svc.Random(context.Background(), model.PostQuery{})
		require.ErrorIs(t, err, model.ErrNotFound)
	}
	repo.random = model.Post{ID: "ok", Rating: "sfw"}
	got, err := svc.Random(context.Background(), model.PostQuery{})
	require.NoError(t, err)
	require.Equal(t, "ok", got.ID)

	repo.random = model.Post{ID: "nsfw", Rating: "nsfw"}
	got, err = svc.Random(context.Background(), model.PostQuery{IncludeNSFW: true})
	require.NoError(t, err)
	require.Equal(t, "nsfw", got.ID)
}

func TestNeighborsClearPostsTheViewerCannotSee(t *testing.T) {
	const (
		visible = "22222222-2222-4222-8222-222222222222"
		hidden  = "33333333-3333-4333-8333-333333333333"
		nsfw    = "44444444-4444-4444-8444-444444444444"
	)
	repo := &fakeRepo{
		postsByID: map[string]model.Post{
			visible: {ID: visible, Rating: "sfw"},
			hidden:  {ID: hidden, Hidden: true, Rating: "sfw"},
			nsfw:    {ID: nsfw, Rating: "nsfw"},
		},
		neighbors: model.Neighbors{PrevID: visible, NextID: hidden, Index: 4, Total: 9},
	}
	svc := NewService(repo, nil)
	got, err := svc.Neighbors(context.Background(), "11111111-1111-4111-8111-111111111111", model.PostQuery{Sort: "favorites"})
	require.NoError(t, err)
	require.Equal(t, visible, got.PrevID)
	require.Equal(t, "", got.NextID)
	require.Equal(t, 4, got.Index)
	require.Equal(t, int64(9), got.Total)
	require.Equal(t, "favorites", repo.lastQuery.Sort)

	repo.neighbors.NextID = nsfw
	got, err = svc.Neighbors(context.Background(), "11111111-1111-4111-8111-111111111111", model.PostQuery{Sort: "oldest"})
	require.NoError(t, err)
	require.Equal(t, "", got.NextID)
	require.Equal(t, "oldest", repo.lastQuery.Sort)

	got, err = svc.Neighbors(context.Background(), "11111111-1111-4111-8111-111111111111", model.PostQuery{Sort: "oldest", IncludeNSFW: true})
	require.NoError(t, err)
	require.Equal(t, nsfw, got.NextID)
}

func TestListByOwnerOffsetDoesNotWrap(t *testing.T) {
	repo := &fakeRepo{
		ownedPosts: []model.Post{
			{ID: "hidden", Hidden: true, Rating: "sfw"},
			{ID: "deleted", Deleted: true, Rating: "sfw"},
			{ID: "nsfw", Rating: "nsfw"},
		},
		ownedTotal: 3,
	}
	page, err := NewService(repo, nil).ListByOwner(context.Background(), "u", false, 40, math.MaxInt32+10)
	require.NoError(t, err)
	require.Equal(t, int32(math.MaxInt32), repo.ownerOffset)
	require.Equal(t, int32(40), repo.ownerLimit)
	require.Equal(t, int64(2), page.Total)
	require.Equal(t, []string{"hidden", "deleted"}, postIDs(page.Posts))
}

func TestListKeepsRowsForAudit(t *testing.T) {
	repo := &fakeRepo{listed: []model.Post{{ID: "hidden", Hidden: true, Rating: "nsfw"}}}
	got, err := NewService(repo, nil).List(context.Background(), 500, -3, true)
	require.NoError(t, err)
	require.Equal(t, int32(0), repo.listOffset)
	require.Equal(t, repo.listed, got)
}

func TestGetBadIDIsNotFound(t *testing.T) {
	repo := &fakeRepo{getErr: errors.New("invalid UUID length: 4")}
	svc := NewService(repo, nil)
	_, err := svc.Get(context.Background(), "nope")
	require.ErrorIs(t, err, model.ErrNotFound)

	_, err = svc.Get(context.Background(), "11111111-1111-4111-8111-111111111111")
	require.Error(t, err)
	require.NotErrorIs(t, err, model.ErrNotFound)
}

func TestRecordCreationLogsHistoryFailure(t *testing.T) {
	repo := &fakeRepo{historyErr: errors.New("history down")}
	var logs bytes.Buffer
	svc := NewService(repo, nil)
	svc.SetJobQueue(nil, slog.New(slog.NewTextHandler(&logs, nil)))
	svc.RecordCreation(context.Background(), model.User{ID: "u1", Username: "ada"}, model.Post{ID: "p9"})
	require.Contains(t, logs.String(), "record post.created")
	require.Contains(t, logs.String(), "history down")
	require.Contains(t, logs.String(), "p9")
}

func TestCreateQueuesDeriveWhenArtistPageFails(t *testing.T) {
	repo := &fakeRepo{artistErr: errors.New("wiki down")}
	q := &fakeQueue{}
	svc := NewService(repo, nil)
	svc.SetJobQueue(q, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := svc.Create(context.Background(), model.Post{Artist: "Ada"}, "sha")
	require.EqualError(t, err, "wiki down")
	require.Len(t, q.kinds, 1)
	require.Equal(t, KindPostDerive, q.kinds[0])
	require.NotEmpty(t, repo.created)
}

func postIDs(posts []model.Post) []string {
	ids := make([]string, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID)
	}
	return ids
}
