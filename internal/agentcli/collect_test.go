package agentcli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

const testSessionToken = "session-token"

func execAgent(t *testing.T, deps Deps, args []string, opt Options) (int, Envelope) {
	t.Helper()
	var buf bytes.Buffer
	code := Execute(context.Background(), deps, args, &buf, opt)
	var env Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	return code, env
}

func signedIn(t *testing.T, user model.User) (*stubAccounts, Options) {
	t.Helper()
	opt := Options{SessionPath: filepath.Join(t.TempDir(), "session")}
	require.NoError(t, SaveSession(opt.SessionPath, testSessionToken))
	return &stubAccounts{token: testSessionToken, user: user}, opt
}

func envelopePostIDs(t *testing.T, data any) []string {
	t.Helper()
	raw, ok := data.([]any)
	require.True(t, ok)
	ids := make([]string, 0, len(raw))
	for _, item := range raw {
		post, ok := item.(map[string]any)
		require.True(t, ok)
		id, ok := post["id"].(string)
		require.True(t, ok)
		require.Equal(t, true, post["is_favorite"])
		ids = append(ids, id)
	}
	return ids
}

func TestFavoritesAdd(t *testing.T) {
	user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	cols := &stubCollections{}
	code, env := execAgent(t, Deps{Accounts: accounts, Collections: cols}, []string{"favorites", "add", "post-1"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "favorites.add", env.Command)
	require.True(t, cols.favCalled)
	require.Equal(t, "user-1", cols.favUser)
	require.Equal(t, "post-1", cols.favPost)
}

func TestFavoritesListNSFW(t *testing.T) {
	posts := []model.Post{
		{ID: "sfw", Rating: "sfw"},
		{ID: "nsfw", Rating: "nsfw"},
	}
	for _, tc := range []struct {
		name string
		show bool
		want []string
	}{
		{name: "opted out", show: false, want: []string{"sfw"}},
		{name: "opted in", show: true, want: []string{"sfw", "nsfw"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember, ShowNSFW: tc.show}
			accounts, opt := signedIn(t, user)
			cols := &stubCollections{posts: posts}
			code, env := execAgent(t, Deps{Accounts: accounts, Collections: cols}, []string{"favorites", "list"}, opt)
			require.Equal(t, 0, code)
			require.Equal(t, tc.want, envelopePostIDs(t, env.Data))
		})
	}
}

func TestListsCreateRequiresTitle(t *testing.T) {
	user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	cols := &stubCollections{}
	code, env := execAgent(t, Deps{Accounts: accounts, Collections: cols}, []string{"lists", "create"}, opt)
	require.Equal(t, 2, code)
	require.Equal(t, "usage", env.Error.Code)
	require.False(t, cols.createCalled)
}

func TestListsCreateDefaultsVisibility(t *testing.T) {
	user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	cols := &stubCollections{}
	code, env := execAgent(t, Deps{Accounts: accounts, Collections: cols}, []string{"lists", "create", "--title", "Mine"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "lists.create", env.Command)
	require.True(t, cols.createCalled)
	require.Equal(t, "user-1", cols.createOwner)
	require.Equal(t, "Mine", cols.createTitle)
	require.Equal(t, "", cols.createDesc)
	require.Equal(t, "", cols.createVis)
}

func TestListsGetVisitor(t *testing.T) {
	cols := &stubCollections{}
	opt := Options{SessionPath: filepath.Join(t.TempDir(), "session")}
	code, env := execAgent(t, Deps{Collections: cols}, []string{"lists", "get", "list-1"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "lists.get", env.Command)
	require.True(t, cols.getCalled)
	require.Nil(t, cols.viewerID)
	require.Equal(t, "list-1", cols.getID)
}

func TestListsAdd(t *testing.T) {
	user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	cols := &stubCollections{}
	code, env := execAgent(t, Deps{Accounts: accounts, Collections: cols}, []string{"lists", "add", "list-1", "post-9"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "lists.add", env.Command)
	require.True(t, cols.addCalled)
	require.Equal(t, "user-1", cols.addOwner)
	require.Equal(t, "list-1", cols.addList)
	require.Equal(t, "post-9", cols.addPost)
}

type stubAccounts struct {
	token  string
	user   model.User
	byName map[string]model.User
}

func (s *stubAccounts) LoginWithPassword(context.Context, string, string) (model.User, string, error) {
	panic("LoginWithPassword")
}

func (s *stubAccounts) UserForSession(_ context.Context, token string) (model.User, error) {
	if token == s.token {
		return s.user, nil
	}
	return model.User{}, accounts.ErrNotFound
}

func (s *stubAccounts) Logout(context.Context, string) error {
	panic("Logout")
}

func (s *stubAccounts) GetByUsername(_ context.Context, username string) (model.User, error) {
	if s.byName == nil {
		panic("GetByUsername")
	}
	user, ok := s.byName[username]
	if !ok {
		return model.User{}, accounts.ErrNotFound
	}
	return user, nil
}

type stubCollections struct {
	favCalled bool
	favUser   string
	favPost   string

	posts []model.Post

	createCalled bool
	createOwner  string
	createTitle  string
	createDesc   string
	createVis    string

	getCalled bool
	viewerID  *string
	getID     string

	addCalled bool
	addOwner  string
	addList   string
	addPost   string
}

func (s *stubCollections) Favorite(_ context.Context, userID, postID string) error {
	s.favCalled = true
	s.favUser = userID
	s.favPost = postID
	return nil
}

func (s *stubCollections) Unfavorite(context.Context, string, string) error {
	panic("Unfavorite")
}

func (s *stubCollections) Favorites(context.Context, string) ([]model.Post, error) {
	return s.posts, nil
}

func (s *stubCollections) FavoriteIDs(context.Context, string) (map[string]bool, error) {
	panic("FavoriteIDs")
}

func (s *stubCollections) CreateList(_ context.Context, ownerID, title, desc, vis string) (model.List, error) {
	s.createCalled = true
	s.createOwner = ownerID
	s.createTitle = title
	s.createDesc = desc
	s.createVis = vis
	return model.List{ID: "list-1", OwnerID: ownerID, Title: title, Description: desc, Visibility: vis}, nil
}

func (s *stubCollections) MyLists(context.Context, string) ([]model.List, error) {
	panic("MyLists")
}

func (s *stubCollections) GetList(_ context.Context, viewerID *string, id string) (model.ListDetail, error) {
	s.getCalled = true
	s.viewerID = viewerID
	s.getID = id
	return model.ListDetail{List: model.List{ID: id}, Items: []model.Post{}}, nil
}

func (s *stubCollections) AddItem(_ context.Context, ownerID, listID, postID string) error {
	s.addCalled = true
	s.addOwner = ownerID
	s.addList = listID
	s.addPost = postID
	return nil
}

func (s *stubCollections) RemoveItem(context.Context, string, string, string) error {
	panic("RemoveItem")
}
