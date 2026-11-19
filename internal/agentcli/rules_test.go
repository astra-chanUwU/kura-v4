package agentcli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/comments"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

// TestVisitorReadsStayPublicAndWritesSignIn checks the catalog against the
// website: public reads run with no session, and every other mutation or
// private read stops before a service write.
func TestVisitorReadsStayPublicAndWritesSignIn(t *testing.T) {
	none := filepath.Join(t.TempDir(), "none")
	board := &postsBoard{
		get: func(_ context.Context, id string) (model.Post, error) {
			return model.Post{ID: id, Rating: "sfw"}, nil
		},
		search: func(context.Context, model.PostQuery) (model.SearchPage, error) {
			return model.SearchPage{Posts: []model.Post{}}, nil
		},
	}
	cols := &stubCollections{}
	tags := &auditTags{}
	reports := &auditReports{}
	safety := &auditCommentSafety{}
	repo := newCommentRepo(publicCommentPost())
	deps := Deps{
		Board:         board,
		Collections:   cols,
		Comments:      comments.NewService(repo),
		Tags:          tags,
		Reports:       reports,
		CommentSafety: safety,
		Site:          &authSite{get: func(context.Context) (model.Site, error) { return model.Site{Name: "Kura"}, nil }},
		Moderation:    &stubModeration{},
	}

	public := [][]string{
		{"schema"},
		{"whoami"},
		{"capabilities"},
		{"site"},
		{"posts", "list"},
		{"posts", "get", "p1"},
		{"posts", "search", "--q", "cat"},
		{"comments", "list", "p1"},
		{"lists", "get", "list-1"},
		{"tags", "show", "cat"},
	}
	for _, args := range public {
		code, env, _ := postsRun(t, deps, args, none)
		require.Equal(t, 0, code, args)
		require.True(t, env.OK, args)
	}

	writes := [][]string{
		{"posts", "upload", "x.jpg"},
		{"posts", "edit", "p1", "--rating", "sfw"},
		{"posts", "hide", "p1"},
		{"posts", "unhide", "p1"},
		{"posts", "restore", "p1"},
		{"posts", "delete", "p1"},
		{"posts", "mine"},
		{"posts", "hidden"},
		{"comments", "create", "p1", "--body", "hi"},
		{"comments", "update", "c1", "--body", "hi"},
		{"comments", "delete", "c1"},
		{"favorites", "list"},
		{"favorites", "add", "p1"},
		{"favorites", "remove", "p1"},
		{"lists", "list"},
		{"lists", "create", "--title", "Mine"},
		{"lists", "add", "list-1", "p1"},
		{"lists", "remove", "list-1", "p1"},
		{"history", "post", "p1"},
		{"history", "recent"},
		{"history", "actor", "ada"},
		{"history", "actors"},
		{"history", "rollback", "h1"},
		{"tags", "edit", "cat", "--category", "general", "--description", "d"},
		{"tags", "alias", "cat", "kitty"},
		{"tags", "unalias", "cat", "kitty"},
		{"tags", "imply", "cat", "animal"},
		{"tags", "unimply", "cat", "animal"},
		{"reports", "create", "p1", "--reason", "spam"},
		{"reports", "list"},
		{"reports", "dismiss", "r1"},
		{"comment-safety", "list"},
		{"comment-safety", "resolvers"},
		{"comment-safety", "dismiss", "e1"},
		{"comment-safety", "remove", "e1"},
		{"comment-safety", "reopen", "e1"},
		{"users", "list"},
		{"users", "role", "ada", "member"},
	}
	for _, args := range writes {
		code, env, raw := postsRun(t, deps, args, none)
		require.Equal(t, 1, code, args)
		require.False(t, env.OK, args)
		require.NotNil(t, env.Error, args)
		require.Equal(t, "unauthenticated", env.Error.Code, args)
		require.NotContains(t, raw, "token")
	}
	require.False(t, tags.mutated)
	require.False(t, reports.mutated)
	require.False(t, safety.mutated)
	require.False(t, cols.favCalled)
	require.False(t, cols.addCalled)
	require.Empty(t, repo.created)
	require.False(t, deps.Moderation.(*stubModeration).listCalled)
	require.False(t, deps.Moderation.(*stubModeration).rollbackCalled)
}

func TestExpiredSessionDoesNotBecomeVisitor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	require.NoError(t, SaveSession(path, "dead-token"))
	searched := false
	acc := &postsAccounts{session: func(context.Context, string) (model.User, error) {
		return model.User{}, model.ErrNotFound
	}}
	board := &postsBoard{search: func(context.Context, model.PostQuery) (model.SearchPage, error) {
		searched = true
		return model.SearchPage{}, nil
	}}
	code, env, raw := postsRun(t, Deps{Accounts: acc, Board: board}, []string{"posts", "list"}, path)
	require.Equal(t, 1, code)
	require.Equal(t, "unauthenticated", env.Error.Code)
	require.NotContains(t, raw, "dead-token")
	require.False(t, searched)
}

type auditTags struct{ mutated bool }

func (a *auditTags) Get(context.Context, *model.User, string) (model.TagPage, error) {
	return model.TagPage{Tag: "cat"}, nil
}
func (a *auditTags) Update(context.Context, model.User, string, string, string) (model.TagPage, error) {
	a.mutated = true
	return model.TagPage{}, nil
}
func (a *auditTags) AddAlias(context.Context, model.User, string, string) (model.TagPage, error) {
	a.mutated = true
	return model.TagPage{}, nil
}
func (a *auditTags) RemoveAlias(context.Context, model.User, string, string) (model.TagPage, error) {
	a.mutated = true
	return model.TagPage{}, nil
}
func (a *auditTags) AddImplication(context.Context, model.User, string, string) (model.TagPage, error) {
	a.mutated = true
	return model.TagPage{}, nil
}
func (a *auditTags) RemoveImplication(context.Context, model.User, string, string) (model.TagPage, error) {
	a.mutated = true
	return model.TagPage{}, nil
}

type auditReports struct{ mutated bool }

func (a *auditReports) Create(context.Context, model.User, string, string, string) (model.PostReport, error) {
	a.mutated = true
	return model.PostReport{}, nil
}
func (a *auditReports) ListOpen(context.Context, model.User, int, int) (model.PostReportPage, error) {
	a.mutated = true
	return model.PostReportPage{}, nil
}
func (a *auditReports) Dismiss(context.Context, model.User, string) (model.PostReport, error) {
	a.mutated = true
	return model.PostReport{}, nil
}

type auditCommentSafety struct{ mutated bool }

func (a *auditCommentSafety) List(context.Context, model.User, model.CommentSafetyFilter, int, int) (model.CommentSafetyPage, error) {
	a.mutated = true
	return model.CommentSafetyPage{}, nil
}
func (a *auditCommentSafety) Resolvers(context.Context, model.User) ([]model.CommentSafetyResolver, error) {
	a.mutated = true
	return nil, nil
}
func (a *auditCommentSafety) Dismiss(context.Context, model.User, string) (model.CommentSafetyEvent, error) {
	a.mutated = true
	return model.CommentSafetyEvent{}, nil
}
func (a *auditCommentSafety) Remove(context.Context, model.User, string) (model.CommentSafetyEvent, error) {
	a.mutated = true
	return model.CommentSafetyEvent{}, nil
}
func (a *auditCommentSafety) Reopen(context.Context, model.User, string) (model.CommentSafetyEvent, error) {
	a.mutated = true
	return model.CommentSafetyEvent{}, nil
}
