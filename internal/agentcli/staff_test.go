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

	rollbackCalled bool
	rollbackActor  model.User
	rollbackID     string

	listCalled bool

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

func (s *stubModeration) HistoryByActor(context.Context, model.User, string, int32) ([]model.ChangeEntry, error) {
	panic("HistoryByActor")
}

func (s *stubModeration) Rollback(_ context.Context, actor model.User, historyID string) (model.ChangeEntry, error) {
	s.rollbackCalled = true
	s.rollbackActor = actor
	s.rollbackID = historyID
	return model.ChangeEntry{ID: historyID, EntityType: "post", EntityID: "post-1", Action: "rollback"}, nil
}

func (s *stubModeration) ListHiddenPosts(context.Context, model.User, string, int, int) (model.HiddenPage, error) {
	panic("ListHiddenPosts")
}
