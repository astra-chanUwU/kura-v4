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
	_, err := svc.GetList(context.Background(), &other, "l1", false, []string{}, 0, 0)
	require.ErrorIs(t, err, model.ErrNotFound)
	_, err = svc.GetList(context.Background(), nil, "l1", false, []string{}, 0, 0)
	require.ErrorIs(t, err, model.ErrNotFound)
}

func TestPublicListVisibleAnonymously(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner", Visibility: "public", ItemCount: 4}, nil)
	repo.EXPECT().CountListItems(mock.Anything, "l1", false, false, []string{}).Return(int64(1), nil)
	repo.EXPECT().ListItems(mock.Anything, "l1", false, false, []string{}, int32(40), int32(0)).Return([]model.Post{{ID: "p1"}}, nil)
	svc := NewService(repo)
	detail, err := svc.GetList(context.Background(), nil, "l1", false, nil, 0, 0)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	require.Equal(t, int64(1), detail.List.ItemCount)
}

func TestOwnerListKeepsHiddenItems(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner", Visibility: "public", ItemCount: 2}, nil)
	repo.EXPECT().CountListItems(mock.Anything, "l1", false, true, []string{}).Return(int64(2), nil)
	repo.EXPECT().ListItems(mock.Anything, "l1", false, true, []string{}, int32(40), int32(0)).Return([]model.Post{{ID: "p1"}, {ID: "hidden"}}, nil)
	owner := "owner"
	detail, err := NewService(repo).GetList(context.Background(), &owner, "l1", false, []string{}, 0, 0)
	require.NoError(t, err)
	require.Len(t, detail.Items, 2)
	require.Equal(t, int64(2), detail.List.ItemCount)
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
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner", Visibility: "public"}, nil)
	svc := NewService(repo)
	_, err := svc.UpdateList(context.Background(), "intruder", "l1", "X", "", "private")
	require.ErrorIs(t, err, ErrForbidden)
}

func TestPrivateListMutationsDoNotDisclose(t *testing.T) {
	secret := model.List{ID: "l1", OwnerID: "owner", Title: "Secret stash", Visibility: "private"}
	ctx := context.Background()

	updateRepo := NewMockRepository(t)
	updateRepo.EXPECT().GetList(mock.Anything, "l1").Return(secret, nil)
	_, err := NewService(updateRepo).UpdateList(ctx, "intruder", "l1", "Stolen", "", "public")
	require.ErrorIs(t, err, model.ErrNotFound)
	require.NotContains(t, err.Error(), "Secret")
	require.NotContains(t, err.Error(), "not yours")

	addRepo := NewMockRepository(t)
	addRepo.EXPECT().GetList(mock.Anything, "l1").Return(secret, nil)
	err = NewService(addRepo).AddItem(ctx, "intruder", "l1", "p1")
	require.ErrorIs(t, err, model.ErrNotFound)

	removeRepo := NewMockRepository(t)
	removeRepo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: "owner", Title: "Secret stash"}, nil)
	err = NewService(removeRepo).RemoveItem(ctx, "intruder", "l1", "p1")
	require.ErrorIs(t, err, model.ErrNotFound)

	reorderRepo := NewMockRepository(t)
	reorderRepo.EXPECT().GetList(mock.Anything, "l1").Return(secret, nil)
	err = NewService(reorderRepo).Reorder(ctx, "intruder", "l1", []string{"p1"})
	require.ErrorIs(t, err, model.ErrNotFound)

	moveRepo := NewMockRepository(t)
	moveRepo.EXPECT().GetList(mock.Anything, "l1").Return(secret, nil)
	err = NewService(moveRepo).MoveItem(ctx, "intruder", "l1", "p1", "")
	require.ErrorIs(t, err, model.ErrNotFound)

	deleteRepo := NewMockRepository(t)
	deleteRepo.EXPECT().DeleteList(mock.Anything, "l1", "intruder").Return(int64(0), nil)
	err = NewService(deleteRepo).DeleteList(ctx, "intruder", "l1")
	require.ErrorIs(t, err, model.ErrNotFound)
	require.NotContains(t, err.Error(), "Secret")
}

func TestFavoritesHideGatedPosts(t *testing.T) {
	viewer := "fan"
	other := "someone"
	posts := []model.Post{
		{ID: "ok", Rating: "sfw"},
		{ID: "nsfw", Rating: "nsfw"},
		{ID: "hidden", Rating: "sfw", Hidden: true, OwnerID: &other},
		{ID: "mine", Rating: "sfw", Hidden: true, OwnerID: &viewer},
		{ID: "gone", Rating: "sfw", Deleted: true, OwnerID: &other},
	}
	repo := NewMockRepository(t)
	repo.EXPECT().CountFavorites(mock.Anything, viewer, false).Return(int64(5), nil)
	repo.EXPECT().ListFavorites(mock.Anything, viewer, false, int32(40), int32(0)).Return(posts, nil)
	page, err := NewService(repo).Favorites(context.Background(), viewer, false, 0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"ok", "mine"}, collectionPostIDs(page.Posts))
	require.Equal(t, int64(2), page.Total)
}

func TestListHidesGatedPostsFromStrangers(t *testing.T) {
	owner := "owner"
	stranger := "stranger"
	posts := []model.Post{
		{ID: "ok", Rating: "sfw"},
		{ID: "nsfw", Rating: "nsfw"},
		{ID: "hidden", Rating: "sfw", Hidden: true},
		{ID: "deleted", Rating: "sfw", Deleted: true},
	}
	repo := NewMockRepository(t)
	repo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: owner, Visibility: "public", ItemCount: 4}, nil)
	repo.EXPECT().CountListItems(mock.Anything, "l1", false, false, []string{}).Return(int64(4), nil)
	repo.EXPECT().ListItems(mock.Anything, "l1", false, false, []string{}, int32(40), int32(0)).Return(posts, nil)
	detail, err := NewService(repo).GetList(context.Background(), &stranger, "l1", false, nil, 0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"ok"}, collectionPostIDs(detail.Items))
	require.Equal(t, int64(1), detail.Total)
	require.Equal(t, int64(1), detail.List.ItemCount)

	ownerRepo := NewMockRepository(t)
	ownerRepo.EXPECT().GetList(mock.Anything, "l1").Return(model.List{ID: "l1", OwnerID: owner, Visibility: "public", ItemCount: 4}, nil)
	ownerRepo.EXPECT().CountListItems(mock.Anything, "l1", false, true, []string{}).Return(int64(4), nil)
	ownerRepo.EXPECT().ListItems(mock.Anything, "l1", false, true, []string{}, int32(40), int32(0)).Return(posts, nil)
	detail, err = NewService(ownerRepo).GetList(context.Background(), &owner, "l1", false, nil, 0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"ok", "hidden", "deleted"}, collectionPostIDs(detail.Items))
	require.Equal(t, int64(3), detail.Total)
	require.Equal(t, int64(4), detail.List.ItemCount)
}

func collectionPostIDs(posts []model.Post) []string {
	out := make([]string, 0, len(posts))
	for _, post := range posts {
		out = append(out, post.ID)
	}
	return out
}
