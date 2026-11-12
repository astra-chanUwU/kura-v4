package agentcli

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/stretchr/testify/require"
)

func TestHistoryRecentForbiddenForMember(t *testing.T) {
	user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "recent"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.False(t, mod.recentCalled)
}

func TestHistoryRecentModerator(t *testing.T) {
	user := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "recent"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "history.recent", env.Command)
	require.True(t, mod.recentCalled)
	require.Equal(t, user.ID, mod.recentActor.ID)
	require.Equal(t, int32(50), mod.recentLimit)
}

func TestHistoryRecentFiltered(t *testing.T) {
	user := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{historyPage: model.HistoryPage{Total: 3, Entries: []model.ChangeEntry{{ID: "h1", EntityType: "post", EntityID: "p1", Action: "post.hidden"}}}}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{
		"history", "recent",
		"--q", "  ada  ",
		"--action", "post.hidden",
		"--entity", "post",
		"--sort", "oldest",
		"--limit", "20",
		"--offset", "5",
	}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "history.recent", env.Command)
	require.False(t, mod.recentCalled)
	require.True(t, mod.historyCalled)
	require.Equal(t, user.ID, mod.historyUser.ID)
	require.Equal(t, "", mod.historyActorID)
	require.Equal(t, "ada", mod.historyQuery)
	require.Equal(t, "post.hidden", mod.historyAction)
	require.Equal(t, "post", mod.historyEntity)
	require.Equal(t, "oldest", mod.historySort)
	require.Equal(t, 20, mod.historyLimit)
	require.Equal(t, 5, mod.historyOffset)
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), data["total"])
	entries, ok := data["entries"].([]any)
	require.True(t, ok)
	require.Len(t, entries, 1)
}

func TestHistoryRecentInvalidAction(t *testing.T) {
	user := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{historyErr: &moderation.ValidationError{Message: "action is not a history action"}}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "recent", "--action", "nope"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "validation", env.Error.Code)
	require.Equal(t, "action is not a history action", env.Error.Message)
	require.False(t, mod.recentCalled)
	require.True(t, mod.historyCalled)
}

func TestHistoryActorOffsetUsesPage(t *testing.T) {
	user := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	target := model.User{ID: "user-9", Username: "bea", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	accounts.byName = map[string]model.User{target.Username: target}
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "actor", "bea", "--limit", "12"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.True(t, mod.byActorCalled)
	require.False(t, mod.historyCalled)
	require.Equal(t, target.ID, mod.byActorID)
	require.Equal(t, int32(12), mod.byActorLimit)

	mod.byActorCalled = false
	code, env = execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "actor", "bea", "--offset", "5"}, opt)
	require.Equal(t, 0, code)
	require.False(t, mod.byActorCalled)
	require.True(t, mod.historyCalled)
	require.Equal(t, target.ID, mod.historyActorID)
	require.Equal(t, "", mod.historyQuery)
	require.Equal(t, 40, mod.historyLimit)
	require.Equal(t, 5, mod.historyOffset)
}

func TestHistoryActorsModeratorOnly(t *testing.T) {
	member := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, member)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "actors", "--q", "be"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.False(t, mod.actorsCalled)

	moderator := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	accounts, opt = signedIn(t, moderator)
	mod = &stubModeration{actors: []model.HistoryActor{{ID: "user-9", Username: "bea"}}}
	code, env = execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "actors", "--q", "  be  "}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "history.actors", env.Command)
	require.True(t, mod.actorsCalled)
	require.Equal(t, moderator.ID, mod.actorsActor.ID)
	require.Equal(t, "be", mod.actorsQuery)
	rows, ok := env.Data.([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user-9", row["id"])
	require.Equal(t, "bea", row["username"])
	_, hasEmail := row["email"]
	require.False(t, hasEmail)

	mod.actorsCalled = false
	mod.actors = nil
	code, env = execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "actors", "--q", "   "}, opt)
	require.Equal(t, 0, code)
	require.Equal(t, "", mod.actorsQuery)
	rows, ok = env.Data.([]any)
	require.True(t, ok)
	require.Empty(t, rows)
}

func TestHistoryRollback(t *testing.T) {
	user := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"history", "rollback", "hist-7"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "history.rollback", env.Command)
	require.True(t, mod.rollbackCalled)
	require.Equal(t, user.ID, mod.rollbackActor.ID)
	require.Equal(t, "hist-7", mod.rollbackID)
}

func TestUsersListForbiddenForMember(t *testing.T) {
	user := model.User{ID: "user-1", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "list"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.False(t, mod.listCalled)
	require.False(t, mod.pageCalled)

	moderator := model.User{ID: "mod-1", Username: "cy", Role: model.RoleModerator}
	accounts, opt = signedIn(t, moderator)
	code, env = execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "list", "--role", "admin"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.False(t, mod.listCalled)
	require.False(t, mod.pageCalled)
}

func TestUsersListNoFlags(t *testing.T) {
	user := model.User{ID: "admin-1", Username: "ada", Role: model.RoleAdmin}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "list"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "users.list", env.Command)
	require.True(t, mod.listCalled)
	require.False(t, mod.pageCalled)
	rows, ok := env.Data.([]any)
	require.True(t, ok)
	require.Empty(t, rows)
}

func TestUsersListForwardsFilters(t *testing.T) {
	user := model.User{ID: "admin-1", Username: "ada", Role: model.RoleAdmin}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{page: model.UserPage{
		Total: 3, Member: 10, Moderator: 2, Admin: 1, SuperAdmin: 1,
		Users: []model.User{{ID: "user-9", Username: "bea", DisplayName: "Bea", Role: model.RoleModerator}},
	}}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{
		"users", "list",
		"--q", "  bea  ",
		"--role", "moderator",
		"--sort", "oldest",
		"--limit", "20",
		"--offset", "5",
	}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "users.list", env.Command)
	require.False(t, mod.listCalled)
	require.True(t, mod.pageCalled)
	require.Equal(t, user.ID, mod.pageActor.ID)
	require.Equal(t, "bea", mod.pageQuery)
	require.Equal(t, "moderator", mod.pageRole)
	require.Equal(t, "oldest", mod.pageSort)
	require.Equal(t, 20, mod.pageLimit)
	require.Equal(t, 5, mod.pageOffset)
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), data["total"])
	require.Equal(t, float64(10), data["member"])
	require.Equal(t, float64(2), data["moderator"])
	require.Equal(t, float64(1), data["admin"])
	require.Equal(t, float64(1), data["super_admin"])
	users, ok := data["users"].([]any)
	require.True(t, ok)
	require.Len(t, users, 1)
	row, ok := users[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "bea", row["username"])
	require.Equal(t, "moderator", row["role"])
}

func TestUsersListLimitOrOffsetUsesPage(t *testing.T) {
	user := model.User{ID: "admin-1", Username: "ada", Role: model.RoleAdmin}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "list", "--limit", "12"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.False(t, mod.listCalled)
	require.True(t, mod.pageCalled)
	require.Equal(t, 12, mod.pageLimit)
	require.Equal(t, 0, mod.pageOffset)

	mod.pageCalled = false
	code, env = execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "list", "--offset", "5"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.False(t, mod.listCalled)
	require.True(t, mod.pageCalled)
	require.Equal(t, "", mod.pageQuery)
	require.Equal(t, "", mod.pageRole)
	require.Equal(t, "", mod.pageSort)
	require.Equal(t, 40, mod.pageLimit)
	require.Equal(t, 5, mod.pageOffset)
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	users, ok := data["users"].([]any)
	require.True(t, ok)
	require.Empty(t, users)
}

func TestUsersListInvalidRole(t *testing.T) {
	user := model.User{ID: "admin-1", Username: "ada", Role: model.RoleAdmin}
	accounts, opt := signedIn(t, user)
	mod := &stubModeration{pageErr: &moderation.ValidationError{Message: "role must be member, moderator, admin, or super_admin"}}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "list", "--role", "owner"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "validation", env.Error.Code)
	require.Equal(t, "role must be member, moderator, admin, or super_admin", env.Error.Message)
	require.False(t, mod.listCalled)
	require.True(t, mod.pageCalled)
	require.Equal(t, "owner", mod.pageRole)
}

func TestUsersRole(t *testing.T) {
	admin := model.User{ID: "admin-1", Username: "ada", Role: model.RoleAdmin}
	target := model.User{ID: "user-9", Username: "bea", Role: model.RoleMember}
	accounts, opt := signedIn(t, admin)
	accounts.byName = map[string]model.User{target.Username: target}
	mod := &stubModeration{}
	deps := Deps{Accounts: accounts, Moderation: mod}

	code, env := execAgent(t, deps, []string{"users", "role", "bea", "moderator"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "users.role", env.Command)
	require.True(t, mod.setCalled)
	require.Equal(t, admin.ID, mod.setActor.ID)
	require.Equal(t, target.ID, mod.setTarget)
	require.Equal(t, model.RoleModerator, mod.setRole)

	mod.setCalled = false
	code, env = execAgent(t, deps, []string{"users", "role", "bea", "owner"}, opt)
	require.Equal(t, 2, code)
	require.Equal(t, "usage", env.Error.Code)
	require.Equal(t, "role must be member, moderator, admin or super_admin", env.Error.Message)
	require.False(t, mod.setCalled)
}

func TestUsersRoleForbidden(t *testing.T) {
	admin := model.User{ID: "admin-1", Username: "ada", Role: model.RoleAdmin}
	target := model.User{ID: "user-9", Username: "bea", Role: model.RoleMember}
	accounts, opt := signedIn(t, admin)
	accounts.byName = map[string]model.User{target.Username: target}
	mod := &stubModeration{setErr: moderation.ErrForbidden}
	code, env := execAgent(t, Deps{Accounts: accounts, Moderation: mod}, []string{"users", "role", "bea", "moderator"}, opt)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.True(t, mod.setCalled)
	require.Equal(t, target.ID, mod.setTarget)
	require.Equal(t, model.RoleModerator, mod.setRole)
}

type stubModeration struct {
	recentCalled bool
	recentActor  model.User
	recentLimit  int32

	byActorCalled bool
	byActorID     string
	byActorLimit  int32

	historyCalled  bool
	historyUser    model.User
	historyActorID string
	historyQuery   string
	historyAction  string
	historyEntity  string
	historySort    string
	historyLimit   int
	historyOffset  int
	historyPage    model.HistoryPage
	historyErr     error

	actorsCalled bool
	actorsActor  model.User
	actorsQuery  string
	actors       []model.HistoryActor

	rollbackCalled bool
	rollbackActor  model.User
	rollbackID     string

	listCalled bool

	pageCalled bool
	pageActor  model.User
	pageQuery  string
	pageRole   string
	pageSort   string
	pageLimit  int
	pageOffset int
	page       model.UserPage
	pageErr    error

	setCalled bool
	setActor  model.User
	setTarget string
	setRole   model.Role
	setErr    error
}

func (s *stubModeration) ListUsers(context.Context, model.User) ([]model.User, error) {
	s.listCalled = true
	return []model.User{}, nil
}

func (s *stubModeration) ListUserPage(_ context.Context, actor model.User, query, role, sort string, limit, offset int) (model.UserPage, error) {
	s.pageCalled = true
	s.pageActor = actor
	s.pageQuery = query
	s.pageRole = role
	s.pageSort = sort
	s.pageLimit = limit
	s.pageOffset = offset
	if s.pageErr != nil {
		return model.UserPage{}, s.pageErr
	}
	if s.page.Users == nil {
		s.page.Users = []model.User{}
	}
	return s.page, nil
}

func (s *stubModeration) SetRole(_ context.Context, actor model.User, targetID string, newRole model.Role) (model.User, error) {
	s.setCalled = true
	s.setActor = actor
	s.setTarget = targetID
	s.setRole = newRole
	if s.setErr != nil {
		return model.User{}, s.setErr
	}
	updated := actor
	updated.ID = targetID
	updated.Role = newRole
	return updated, nil
}

func (s *stubModeration) SetPostHidden(context.Context, model.User, string, bool, bool) (model.Post, error) {
	panic("SetPostHidden")
}

func (s *stubModeration) HistoryForPost(context.Context, string, int32) ([]model.ChangeEntry, error) {
	panic("HistoryForPost")
}

func (s *stubModeration) RecentHistory(_ context.Context, actor model.User, limit int32) ([]model.ChangeEntry, error) {
	s.recentCalled = true
	s.recentActor = actor
	s.recentLimit = limit
	return []model.ChangeEntry{}, nil
}

func (s *stubModeration) HistoryByActor(_ context.Context, _ model.User, actorID string, limit int32) ([]model.ChangeEntry, error) {
	s.byActorCalled = true
	s.byActorID = actorID
	s.byActorLimit = limit
	return []model.ChangeEntry{}, nil
}

func (s *stubModeration) ListHistory(_ context.Context, actor model.User, actorID, query, action, entity, sort string, limit, offset int) (model.HistoryPage, error) {
	s.historyCalled = true
	s.historyUser = actor
	s.historyActorID = actorID
	s.historyQuery = query
	s.historyAction = action
	s.historyEntity = entity
	s.historySort = sort
	s.historyLimit = limit
	s.historyOffset = offset
	if s.historyErr != nil {
		return model.HistoryPage{}, s.historyErr
	}
	return s.historyPage, nil
}

func (s *stubModeration) SearchHistoryActors(_ context.Context, actor model.User, query string) ([]model.HistoryActor, error) {
	s.actorsCalled = true
	s.actorsActor = actor
	s.actorsQuery = query
	if s.actors == nil {
		return []model.HistoryActor{}, nil
	}
	return s.actors, nil
}

func (s *stubModeration) Rollback(_ context.Context, actor model.User, historyID string) (model.ChangeEntry, error) {
	s.rollbackCalled = true
	s.rollbackActor = actor
	s.rollbackID = historyID
	return model.ChangeEntry{ID: historyID, EntityType: "post", EntityID: "post-1", Action: "rollback"}, nil
}

func (s *stubModeration) ListHiddenPosts(context.Context, model.User, string, string, string, string, int, int) (model.HiddenPage, error) {
	panic("ListHiddenPosts")
}
