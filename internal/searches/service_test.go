package searches

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	items map[string][]model.SavedSearch
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string][]model.SavedSearch{}}
}

func (f *fakeRepo) List(_ context.Context, userID string) ([]model.SavedSearch, error) {
	src := f.items[userID]
	out := make([]model.SavedSearch, len(src))
	copy(out, src)
	return out, nil
}

func (f *fakeRepo) Count(_ context.Context, userID string) (int64, error) {
	return int64(len(f.items[userID])), nil
}

func (f *fakeRepo) Create(_ context.Context, userID string, in model.SavedSearch) (model.SavedSearch, error) {
	in.UserID = userID
	if in.ID == "" {
		in.ID = "saved-" + in.Name
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Unix(0, 0).UTC()
	}
	in.IncludeTags = append([]string(nil), in.IncludeTags...)
	in.ExcludeTags = append([]string(nil), in.ExcludeTags...)
	f.items[userID] = append(f.items[userID], in)
	return in, nil
}

func (f *fakeRepo) Delete(_ context.Context, userID, id string) (int64, error) {
	items := f.items[userID]
	for i, item := range items {
		if item.ID == id {
			f.items[userID] = append(items[:i], items[i+1:]...)
			return 1, nil
		}
	}
	return 0, nil
}

func validationMessage(t *testing.T, err error) string {
	t.Helper()
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	return vErr.Message
}

func TestCreateRejectsEmptyName(t *testing.T) {
	repo := newFakeRepo()
	_, err := NewService(repo).Create(context.Background(), "u", model.SavedSearch{Name: "   "})
	require.Equal(t, "Name your search first.", validationMessage(t, err))
	require.Empty(t, repo.items["u"])
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	repo := newFakeRepo()
	repo.items["u"] = []model.SavedSearch{{
		ID: "s1", Name: "Daily", IncludeTags: []string{}, ExcludeTags: []string{},
	}}
	_, err := NewService(repo).Create(context.Background(), "u", model.SavedSearch{Name: "  Daily  "})
	require.Equal(t, "You already saved a search with that name.", validationMessage(t, err))
	require.Len(t, repo.items["u"], 1)
}

func TestCreateCapsAtThirty(t *testing.T) {
	repo := newFakeRepo()
	repo.items["u"] = make([]model.SavedSearch, 30)
	for i := range repo.items["u"] {
		repo.items["u"][i] = model.SavedSearch{
			ID:          strings.Repeat("i", i+1),
			Name:        "search-" + strings.Repeat("n", i+1),
			IncludeTags: []string{},
			ExcludeTags: []string{},
		}
	}
	_, err := NewService(repo).Create(context.Background(), "u", model.SavedSearch{Name: "one more"})
	require.Equal(t, "You can save up to 30 searches.", validationMessage(t, err))
	require.Len(t, repo.items["u"], 30)
}

func TestCreateNormalizesTags(t *testing.T) {
	got, err := NewService(newFakeRepo()).Create(context.Background(), "u", model.SavedSearch{
		Name:        "  Hair  ",
		IncludeTags: []string{"White Hair", "white-hair", strings.Repeat("a", 41), "!!!!"},
		ExcludeTags: []string{"smile, solo", "smile"},
		Query:       "  yuri  ",
	})
	require.NoError(t, err)
	require.Equal(t, "Hair", got.Name)
	require.Equal(t, []string{"white_hair"}, got.IncludeTags)
	require.Equal(t, []string{"smile", "solo"}, got.ExcludeTags)
	require.Equal(t, "yuri", got.Query)
	require.Equal(t, "newest", got.Sort)
	require.NotNil(t, got.IncludeTags)
	require.NotNil(t, got.ExcludeTags)
}

func TestCreateRejectsBadSort(t *testing.T) {
	repo := newFakeRepo()
	_, err := NewService(repo).Create(context.Background(), "u", model.SavedSearch{Name: "Ok", Sort: "popular"})
	require.Equal(t, "Sort must be newest, oldest, or favorites.", validationMessage(t, err))
	require.Empty(t, repo.items["u"])
}

func TestCreateStoresOldest(t *testing.T) {
	got, err := NewService(newFakeRepo()).Create(context.Background(), "u", model.SavedSearch{Name: "Old", Sort: "oldest"})
	require.NoError(t, err)
	require.Equal(t, "oldest", got.Sort)

	got, err = NewService(newFakeRepo()).Create(context.Background(), "u", model.SavedSearch{Name: "Hot", Sort: "favorites"})
	require.NoError(t, err)
	require.Equal(t, "favorites", got.Sort)
	require.Equal(t, []string{}, got.IncludeTags)
	require.Equal(t, []string{}, got.ExcludeTags)
}

func TestCreateRejectsLongNameAndQuery(t *testing.T) {
	svc := NewService(newFakeRepo())
	_, err := svc.Create(context.Background(), "u", model.SavedSearch{Name: strings.Repeat("a", 61)})
	require.Equal(t, "Search names are capped at 60 characters.", validationMessage(t, err))
	_, err = svc.Create(context.Background(), "u", model.SavedSearch{Name: "Ok", Query: strings.Repeat("b", 81)})
	require.Equal(t, "Search text is capped at 80 characters.", validationMessage(t, err))
}

func TestDeleteIsScopedToOwner(t *testing.T) {
	repo := newFakeRepo()
	repo.items["owner"] = []model.SavedSearch{{
		ID: "s1", UserID: "owner", Name: "Mine", IncludeTags: []string{}, ExcludeTags: []string{},
	}}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), "intruder", "s1")
	require.ErrorIs(t, err, ErrNotFound)
	require.Len(t, repo.items["owner"], 1)
	require.NoError(t, svc.Delete(context.Background(), "owner", "s1"))
	require.Empty(t, repo.items["owner"])
	require.ErrorIs(t, svc.Delete(context.Background(), "owner", "missing"), ErrNotFound)
}

func TestListEmptyIsNotNil(t *testing.T) {
	items, err := NewService(newFakeRepo()).List(context.Background(), "u")
	require.NoError(t, err)
	require.NotNil(t, items)
	require.Empty(t, items)
}
