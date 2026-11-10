package agentcli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/comments"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCommentsVisitorListsPublicPostAndCannotCreate(t *testing.T) {
	repo := newCommentRepo(publicCommentPost(), model.Comment{
		ID: "c1", PostID: "p1", AuthorID: "author", AuthorUsername: "author", Body: "hello",
	})
	svc := comments.NewService(repo)
	opt := Options{SessionPath: filepath.Join(t.TempDir(), "session")}
	deps := Deps{Comments: svc}

	code, env, raw := runComments(t, deps, []string{"comments", "list", "p1"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "comments.list", env.Command)
	data := commentData(t, env)
	require.Equal(t, float64(1), data["total"])
	rows, ok := data["comments"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "c1", row["id"])
	require.Equal(t, "hello", row["body"])
	require.Equal(t, false, row["can_edit"])
	require.Equal(t, false, row["can_delete"])
	require.Equal(t, int32(40), repo.lastLimit)
	require.Equal(t, int32(0), repo.lastOffset)
	require.NotContains(t, raw, testSessionToken)

	code, env, _ = runComments(t, deps, []string{"comments", "list", "p1", "--limit", "10", "--offset", "5"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, int32(10), repo.lastLimit)
	require.Equal(t, int32(5), repo.lastOffset)

	code, env, _ = runComments(t, deps, []string{"comments", "list", "p1", "--limit", "101"}, opt)
	require.Equal(t, 2, code)
	require.False(t, env.OK)
	require.Equal(t, "usage", env.Error.Code)
	require.Equal(t, int32(10), repo.lastLimit)

	code, env, raw = runComments(t, deps, []string{"comments", "create", "p1", "--body", "nope"}, opt)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.Equal(t, "comments.create", env.Command)
	require.NotNil(t, env.Error)
	require.Equal(t, "unauthenticated", env.Error.Code)
	require.Empty(t, repo.created)
	require.NotContains(t, raw, "nope")
}

func TestCommentsMemberCreatesAndUpdatesOwn(t *testing.T) {
	user := model.User{ID: "mem", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	repo := newCommentRepo(publicCommentPost())
	deps := Deps{Accounts: accounts, Comments: comments.NewService(repo)}

	code, env, raw := runComments(t, deps, []string{"comments", "create", "p1", "--body", "  hello  "}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "comments.create", env.Command)
	created := commentData(t, env)
	require.Equal(t, "c-new", created["id"])
	require.Equal(t, "p1", created["post_id"])
	require.Equal(t, "hello", created["body"])
	require.Equal(t, true, created["can_edit"])
	require.Equal(t, true, created["can_delete"])
	require.Equal(t, "mem", repo.comments[0].AuthorID)
	require.Equal(t, "hello", repo.comments[0].Body)
	require.NotContains(t, raw, testSessionToken)

	code, env, raw = runComments(t, deps, []string{"comments", "update", "c-new", "--body", "  revised  "}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "comments.update", env.Command)
	updated := commentData(t, env)
	require.Equal(t, "c-new", updated["id"])
	require.Equal(t, "revised", updated["body"])
	require.Equal(t, true, updated["can_edit"])
	require.Equal(t, "revised", repo.comments[0].Body)
	require.NotContains(t, raw, testSessionToken)
}

func TestCommentsDeleteAuthorOrModerator(t *testing.T) {
	for _, tc := range []struct {
		name     string
		userID   string
		role     model.Role
		wantCode int
		wantErr  string
		deleted  bool
	}{
		{name: "other member", userID: "other", role: model.RoleMember, wantCode: 1, wantErr: "forbidden"},
		{name: "moderator", userID: "mod", role: model.RoleModerator, wantCode: 0, deleted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{ID: tc.userID, Username: tc.userID, Role: tc.role}
			accounts, opt := signedIn(t, user)
			repo := newCommentRepo(publicCommentPost(), model.Comment{
				ID: "c1", PostID: "p1", AuthorID: "author", AuthorUsername: "author", Body: "hello",
			})
			deps := Deps{Accounts: accounts, Comments: comments.NewService(repo)}
			code, env, raw := runComments(t, deps, []string{"comments", "delete", "c1"}, opt)
			require.Equal(t, tc.wantCode, code)
			require.Equal(t, "comments.delete", env.Command)
			require.NotContains(t, raw, testSessionToken)
			if tc.deleted {
				require.True(t, env.OK)
				require.Equal(t, "c1", commentData(t, env)["comment_id"])
				require.Equal(t, []string{"c1"}, repo.deleted)
				return
			}
			require.False(t, env.OK)
			require.NotNil(t, env.Error)
			require.Equal(t, tc.wantErr, env.Error.Code)
			require.Empty(t, repo.deleted)
			require.Equal(t, "hello", repo.comments[0].Body)
		})
	}
}

func TestCommentsHiddenPostNotFoundForUnrelatedMember(t *testing.T) {
	user := model.User{ID: "mem", Username: "ada", Role: model.RoleMember}
	accounts, opt := signedIn(t, user)
	repo := newCommentRepo(model.CommentPostAccess{
		OwnerID: "owner", Rating: "sfw", Hidden: true,
	}, model.Comment{
		ID: "c1", PostID: "p1", AuthorID: "author", AuthorUsername: "author", Body: "secret",
	})
	svc := comments.NewService(repo)
	deps := Deps{Accounts: accounts, Comments: svc}

	code, env, raw := runComments(t, deps, []string{"comments", "list", "p1"}, opt)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.Equal(t, "comments.list", env.Command)
	require.NotNil(t, env.Error)
	require.Equal(t, "not_found", env.Error.Code)
	require.Nil(t, env.Data)
	require.Zero(t, repo.lastLimit)
	require.NotContains(t, raw, "secret")
	require.NotContains(t, raw, testSessionToken)

	owner := model.User{ID: "owner", Username: "owner", Role: model.RoleMember}
	accounts, opt = signedIn(t, owner)
	code, env, raw = runComments(t, Deps{Accounts: accounts, Comments: svc}, []string{"comments", "list", "p1"}, opt)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	rows, ok := commentData(t, env)["comments"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "secret", row["body"])
	require.Equal(t, false, row["can_edit"])
	require.Equal(t, false, row["can_delete"])
	require.NotContains(t, raw, testSessionToken)
}

func runComments(t *testing.T, deps Deps, args []string, opt Options) (int, Envelope, string) {
	t.Helper()
	var buf bytes.Buffer
	code := Execute(context.Background(), deps, args, &buf, opt)
	var env Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	return code, env, buf.String()
}

func commentData(t *testing.T, env Envelope) map[string]any {
	t.Helper()
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	return data
}

type commentRepo struct {
	access     model.CommentPostAccess
	comments   []model.Comment
	created    []model.Comment
	deleted    []string
	lastLimit  int32
	lastOffset int32
}

func newCommentRepo(access model.CommentPostAccess, comments ...model.Comment) *commentRepo {
	return &commentRepo{access: access, comments: append([]model.Comment{}, comments...)}
}

func publicCommentPost() model.CommentPostAccess {
	return model.CommentPostAccess{OwnerID: "owner", Rating: "sfw"}
}

func (f *commentRepo) CommentPostAccess(context.Context, string) (model.CommentPostAccess, error) {
	return f.access, nil
}

func (f *commentRepo) ListCommentsByPost(_ context.Context, postID string, limit, offset int32) ([]model.Comment, error) {
	f.lastLimit = limit
	f.lastOffset = offset
	var out []model.Comment
	for _, comment := range f.comments {
		if comment.PostID == postID {
			out = append(out, comment)
		}
	}
	return out, nil
}

func (f *commentRepo) CountCommentsByPost(_ context.Context, postID string) (int64, error) {
	var n int64
	for _, comment := range f.comments {
		if comment.PostID == postID {
			n++
		}
	}
	return n, nil
}

func (f *commentRepo) CreateComment(_ context.Context, postID, authorID, body string) (model.Comment, error) {
	comment := model.Comment{ID: "c-new", PostID: postID, AuthorID: authorID, AuthorUsername: "ada", Body: body}
	f.comments = append(f.comments, comment)
	f.created = append(f.created, comment)
	return comment, nil
}

func (f *commentRepo) GetComment(_ context.Context, id string) (model.Comment, error) {
	for _, comment := range f.comments {
		if comment.ID == id {
			return comment, nil
		}
	}
	return model.Comment{}, sql.ErrNoRows
}

func (f *commentRepo) UpdateCommentBody(_ context.Context, id, body string) (model.Comment, error) {
	for i, comment := range f.comments {
		if comment.ID == id {
			comment.Body = body
			f.comments[i] = comment
			return comment, nil
		}
	}
	return model.Comment{}, sql.ErrNoRows
}

func (f *commentRepo) DeleteComment(_ context.Context, id string) (int64, error) {
	for i, comment := range f.comments {
		if comment.ID == id {
			f.comments = append(f.comments[:i], f.comments[i+1:]...)
			f.deleted = append(f.deleted, id)
			return 1, nil
		}
	}
	return 0, nil
}
