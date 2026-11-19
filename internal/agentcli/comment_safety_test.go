package agentcli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/commentsafety"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type fakeCommentSafety struct {
	listCalls int
	filter    model.CommentSafetyFilter
	dismissID string
	removeID  string
	reopenID  string
	limit     int
	offset    int
	forbid    bool
	missing   bool
}

func (f *fakeCommentSafety) List(_ context.Context, _ model.User, filter model.CommentSafetyFilter, limit, offset int) (model.CommentSafetyPage, error) {
	f.listCalls++
	f.filter, f.limit, f.offset = filter, limit, offset
	if f.forbid {
		return model.CommentSafetyPage{}, commentsafety.ErrForbidden
	}
	return model.CommentSafetyPage{
		Total: 1,
		Events: []model.CommentSafetyEvent{{
			ID: "e1", CommentID: "c1", PostID: "p1", AuthorUsername: "ada",
			Action: "flagged", Categories: nil, Body: "dm me", Status: "open",
		}},
	}, nil
}

func (f *fakeCommentSafety) Resolvers(context.Context, model.User) ([]model.CommentSafetyResolver, error) {
	if f.forbid {
		return nil, commentsafety.ErrForbidden
	}
	return []model.CommentSafetyResolver{{
		ID: "22222222-2222-4222-8222-222222222222", Username: "kiri", Decisions: 2,
		LastStatus: "dismissed", LastAction: "blocked", LastCategories: nil,
		LastBody: "https://t.me/example_user",
	}}, nil
}

func (f *fakeCommentSafety) Dismiss(_ context.Context, _ model.User, id string) (model.CommentSafetyEvent, error) {
	f.dismissID = id
	if f.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	return model.CommentSafetyEvent{
		ID: id, CommentID: "c1", PostID: "p1", AuthorUsername: "ada",
		Action: "flagged", Categories: []string{"solicitation"}, Body: "dm me", Status: "dismissed",
	}, nil
}

func (f *fakeCommentSafety) Reopen(_ context.Context, _ model.User, id string) (model.CommentSafetyEvent, error) {
	f.reopenID = id
	if f.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	return model.CommentSafetyEvent{
		ID: id, CommentID: "c9", PostID: "p1", AuthorUsername: "ada",
		Action: "flagged", Categories: []string{"email"}, Body: "reach me", Status: "open",
	}, nil
}

func (f *fakeCommentSafety) Remove(_ context.Context, _ model.User, id string) (model.CommentSafetyEvent, error) {
	f.removeID = id
	return model.CommentSafetyEvent{
		ID: id, PostID: "p1", AuthorUsername: "ada", Action: "blocked",
		Categories: []string{"phone"}, Body: "555-123-4567", Status: "removed",
	}, nil
}

func TestCommentSafetyListModerator(t *testing.T) {
	session, accounts := postsSession(t, model.RoleMember)
	store := &fakeCommentSafety{}
	code, env, _ := postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "list"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.Equal(t, "comment-safety.list", env.Command)
	require.Zero(t, store.listCalls)

	session, accounts = postsSession(t, model.RoleModerator)
	code, env, _ = postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "list", "--limit", "10", "--offset", "5"}, session)
	require.Equal(t, 0, code)
	require.Equal(t, "comment-safety.list", env.Command)
	require.Equal(t, 1, store.listCalls)
	require.Equal(t, 10, store.limit)
	require.Equal(t, 5, store.offset)
	require.Equal(t, "open", store.filter.Status)
	data := postsData(t, env)
	require.Equal(t, float64(1), data["total"])
	events, ok := data["events"].([]any)
	require.True(t, ok)
	require.Len(t, events, 1)
	row, ok := events[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "e1", row["id"])
	require.Equal(t, "c1", row["comment_id"])
	require.Equal(t, "p1", row["post_id"])
	require.Equal(t, "ada", row["author_username"])
	require.Equal(t, "flagged", row["action"])
	require.Equal(t, "dm me", row["body"])
	require.Equal(t, "open", row["status"])
	cats, ok := row["categories"].([]any)
	require.True(t, ok)
	require.Empty(t, cats)
	_, hasCreated := row["created_at"]
	require.False(t, hasCreated)

	code, env, _ = postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "list", "extra"}, session)
	require.Equal(t, 2, code)
	require.Equal(t, "usage", env.Error.Code)
}

func TestCommentSafetyDismissAndRemove(t *testing.T) {
	session, accounts := postsSession(t, model.RoleModerator)
	store := &fakeCommentSafety{}
	deps := Deps{Accounts: accounts, CommentSafety: store}
	code, env, _ := postsRun(t, deps, []string{"comment-safety", "dismiss", "e1"}, session)
	require.Equal(t, 0, code)
	require.Equal(t, "comment-safety.dismiss", env.Command)
	require.Equal(t, "e1", store.dismissID)
	row := postsData(t, env)
	require.Equal(t, "dismissed", row["status"])
	require.Equal(t, "c1", row["comment_id"])
	require.Equal(t, []any{"solicitation"}, row["categories"])

	code, env, _ = postsRun(t, deps, []string{"comment-safety", "remove", "e2"}, session)
	require.Equal(t, 0, code)
	require.Equal(t, "comment-safety.remove", env.Command)
	require.Equal(t, "e2", store.removeID)
	row = postsData(t, env)
	require.Equal(t, "removed", row["status"])
	require.Equal(t, "", row["comment_id"])
	require.Equal(t, []any{"phone"}, row["categories"])

	code, env, _ = postsRun(t, deps, []string{"comment-safety", "reopen", "e3"}, session)
	require.Equal(t, 0, code)
	require.Equal(t, "comment-safety.reopen", env.Command)
	require.Equal(t, "e3", store.reopenID)
	row = postsData(t, env)
	require.Equal(t, "open", row["status"])
	require.Equal(t, "c9", row["comment_id"])

	session, accounts = postsSession(t, model.RoleMember)
	code, env, _ = postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "dismiss", "e-member"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.Equal(t, "e1", store.dismissID)

	store.missing = true
	session, accounts = postsSession(t, model.RoleModerator)
	code, env, _ = postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "dismiss", "gone"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "not_found", env.Error.Code)

	store.forbid = true
	code, env, _ = postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "list"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)
	require.Equal(t, "You don't have permission to do that.", env.Error.Message)

	store.forbid = false
	code, env, _ = postsRun(t, Deps{Accounts: accounts, CommentSafety: store}, []string{"comment-safety", "resolvers"}, session)
	require.Equal(t, 0, code)
	require.Equal(t, "comment-safety.resolvers", env.Command)
	row = postsData(t, env)
	resolvers, ok := row["resolvers"].([]any)
	require.True(t, ok)
	require.Len(t, resolvers, 1)
	person, ok := resolvers[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "kiri", person["username"])
	require.Equal(t, "dismissed", person["last_status"])
	require.Equal(t, []any{}, person["last_categories"])
	require.Equal(t, float64(2), person["decisions"])
}

func TestCommentSafetyVisitorRejected(t *testing.T) {
	none := filepath.Join(t.TempDir(), "none")
	store := &fakeCommentSafety{}
	for _, args := range [][]string{
		{"comment-safety", "list"},
		{"comment-safety", "resolvers"},
		{"comment-safety", "dismiss", "e1"},
		{"comment-safety", "remove", "e1"},
		{"comment-safety", "reopen", "e1"},
	} {
		code, env, _ := postsRun(t, Deps{CommentSafety: store}, args, none)
		require.Equal(t, 1, code, args)
		require.Equal(t, "unauthenticated", env.Error.Code, args)
	}
	require.Zero(t, store.listCalls)
	require.Empty(t, store.dismissID)
	require.Empty(t, store.removeID)
	require.Empty(t, store.reopenID)
}
