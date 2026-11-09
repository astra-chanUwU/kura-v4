package board

import (
	"context"
	"database/sql"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestRelatedLimitClamp(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)
	for _, tc := range []struct {
		in   int
		want int32
	}{
		{in: 0, want: 12},
		{in: -3, want: 12},
		{in: 5, want: 5},
		{in: 24, want: 24},
		{in: 100, want: 24},
	} {
		posts, err := svc.Related(context.Background(), "p1", true, []string{}, tc.in)
		require.NoError(t, err)
		require.Empty(t, posts)
		require.Equal(t, tc.want, repo.relatedLimit)
		require.True(t, repo.relatedNSFW)
		require.NotNil(t, repo.relatedExclude)
		require.Empty(t, repo.relatedExclude)
	}
}

func TestRelatedPassesExclude(t *testing.T) {
	repo := &fakeRepo{}
	exclude := []string{"spoilers", "gore"}
	posts, err := NewService(repo, nil).Related(context.Background(), "p1", false, exclude, 5)
	require.NoError(t, err)
	require.Empty(t, posts)
	require.Equal(t, exclude, repo.relatedExclude)
	require.False(t, repo.relatedNSFW)
	require.Equal(t, int32(5), repo.relatedLimit)
}

func TestSearchRewritesAliasToCanonicalTag(t *testing.T) {
	repo := &fakeRepo{aliases: map[string]string{"grey_hair": "gray_hair"}}
	_, err := NewService(repo, nil).Search(context.Background(), model.PostQuery{Tags: []string{"grey_hair", "gray_hair"}})
	require.NoError(t, err)
	require.Equal(t, []string{"gray_hair"}, repo.lastQuery.Tags)
}

func TestSetMetaRewritesAliasTag(t *testing.T) {
	repo := &fakeRepo{aliases: map[string]string{"grey_hair": "gray_hair"}, post: model.Post{ID: "p1", Rating: "sfw"}}
	_, err := NewService(repo, nil).SetMeta(context.Background(), "p1", "", []string{"grey_hair"}, "sfw", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gray_hair"}, repo.meta[0].Tags)
}

func TestRandomNotFound(t *testing.T) {
	repo := &fakeRepo{randomErr: sql.ErrNoRows}
	_, err := NewService(repo, nil).Random(context.Background(), model.PostQuery{Tags: []string{"missing"}})
	require.ErrorIs(t, err, ErrNotFound)

	repo.randomErr = ErrNotFound
	_, err = NewService(repo, nil).Random(context.Background(), model.PostQuery{})
	require.ErrorIs(t, err, ErrNotFound)
}
