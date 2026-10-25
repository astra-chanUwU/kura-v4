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
