package collections

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPrivateListHiddenFromOthers(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner", Visibility: "private"}, nil)
	svc := NewService(repo)
	other := "someone-else"
	_, err := svc.GetList(context.Background(), &other, "l1", false, 0, 0)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = svc.GetList(context.Background(), nil, "l1", false, 0, 0)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPublicListVisibleAnonymously(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner", Visibility: "public"}, nil)
	repo.EXPECT().CountListItems(mock.Anything, "l1", false).Return(int64(1), nil)
	repo.EXPECT().ListItems(mock.Anything, "l1", false, int32(40), int32(0)).Return([]model.Post{{ID: "p1"}}, nil)
	svc := NewService(repo)
	detail, err := svc.GetList(context.Background(), nil, "l1", false, 0, 0)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
}

func TestCreateListDefaultsToPrivate(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().CreateList(mock.Anything, "owner", "Faves", "", "private").Return(model.List{ID: "l1", Visibility: "private"}, nil)
	svc := NewService(repo)
	list, err := svc.CreateList(context.Background(), "owner", "  Faves ", "", "surprise")
	require.NoError(t, err)
	require.Equal(t, "private", list.Visibility)
}

func TestReorderRejectsMissingItems(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner"}, nil)
	repo.EXPECT().ListItemIDs(mock.Anything, "l1").Return([]string{"p1", "p2"}, nil)
	svc := NewService(repo)
	err := svc.Reorder(context.Background(), "owner", "l1", []string{"p1"})
	require.Error(t, err)
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
}

func TestMoveItemPlacesAfterPredecessor(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner"}, nil)
	repo.EXPECT().ListItemIDs(mock.Anything, "l1").Return([]string{"p1", "p2", "p3"}, nil)
	repo.EXPECT().Reorder(mock.Anything, "l1", []string{"p1", "p3", "p2"}).Return(nil)
	err := NewService(repo).MoveItem(context.Background(), "owner", "l1", "p3", "p1")
	require.NoError(t, err)
}

func TestUpdateForeignListForbidden(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner"}, nil)
	svc := NewService(repo)
	_, err := svc.UpdateList(context.Background(), "intruder", "l1", "X", "", "private")
	require.ErrorIs(t, err, ErrForbidden)
}
