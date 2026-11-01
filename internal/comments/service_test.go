package comments

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	access      model.CommentPostAccess
	accessErr   error
	comments    []model.Comment
	updateCalls int
	deleted     []string
	lastLimit   int32
}

func (f *fakeRepo) CommentPostAccess(context.Context, string) (model.CommentPostAccess, error) {
	if f.accessErr != nil {
		return model.CommentPostAccess{}, f.accessErr
	}
	return f.access, nil
}

func (f *fakeRepo) ListCommentsByPost(_ context.Context, postID string, limit, _ int32) ([]model.Comment, error) {
	f.lastLimit = limit
	var out []model.Comment
	for _, c := range f.comments {
		if c.PostID == postID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) CountCommentsByPost(_ context.Context, postID string) (int64, error) {
	var n int64
	for _, c := range f.comments {
		if c.PostID == postID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) CreateComment(_ context.Context, postID, authorID, body string) (model.Comment, error) {
	c := model.Comment{ID: "c-new", PostID: postID, AuthorID: authorID, AuthorUsername: "member", Body: body}
	f.comments = append(f.comments, c)
	return c, nil
}

func (f *fakeRepo) GetComment(_ context.Context, id string) (model.Comment, error) {
	for _, c := range f.comments {
		if c.ID == id {
			return c, nil
		}
	}
	return model.Comment{}, sql.ErrNoRows
}

func (f *fakeRepo) UpdateCommentBody(_ context.Context, id, body string) (model.Comment, error) {
	f.updateCalls++
	for i, c := range f.comments {
		if c.ID == id {
			c.Body = body
			f.comments[i] = c
			return c, nil
		}
	}
	return model.Comment{}, sql.ErrNoRows
}

func (f *fakeRepo) DeleteComment(_ context.Context, id string) (int64, error) {
	for i, c := range f.comments {
		if c.ID == id {
			f.comments = append(f.comments[:i], f.comments[i+1:]...)
			f.deleted = append(f.deleted, id)
			return 1, nil
		}
	}
	return 0, nil
}

func publicPost() model.CommentPostAccess {
	return model.CommentPostAccess{OwnerID: "owner", Rating: "sfw"}
}

func member(id string) model.User {
	return model.User{ID: id, Role: model.RoleMember}
}

func seed(access model.CommentPostAccess) *fakeRepo {
	return &fakeRepo{
		access: access,
		comments: []model.Comment{{
			ID: "c1", PostID: "p1", AuthorID: "author", AuthorUsername: "author", Body: "hello",
		}},
	}
}

func TestAnonymousCannotCreate(t *testing.T) {
	svc := NewService(&fakeRepo{access: publicPost()})
	_, err := svc.Create(context.Background(), model.User{}, "p1", "hello")
	require.ErrorIs(t, err, ErrForbidden)
}

func TestMemberCanCreate(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	svc := NewService(repo)
	got, err := svc.Create(context.Background(), member("mem"), "p1", "  hello  ")
	require.NoError(t, err)
	require.Equal(t, "hello", got.Body)
	require.Equal(t, "mem", got.AuthorID)
	require.Equal(t, "hello", repo.comments[0].Body)
}

func TestRejectsEmptyComment(t *testing.T) {
	svc := NewService(&fakeRepo{access: publicPost()})
	_, err := svc.Create(context.Background(), member("mem"), "p1", "   ")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
}

func TestRejectsTooLongComment(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	svc := NewService(repo)
	_, err := svc.Create(context.Background(), member("mem"), "p1", strings.Repeat("a", 2001))
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Empty(t, repo.comments)

	body := strings.Repeat("あ", 2000)
	got, err := svc.Create(context.Background(), member("mem"), "p1", body)
	require.NoError(t, err)
	require.Equal(t, body, got.Body)
}

func TestNSFWHiddenWithoutShowNSFW(t *testing.T) {
	repo := seed(model.CommentPostAccess{OwnerID: "owner", Rating: "nsfw"})
	svc := NewService(repo)
	viewer := member("mem")
	_, err := svc.List(context.Background(), &viewer, "p1", 0, 0)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = svc.Create(context.Background(), viewer, "p1", "nope")
	require.ErrorIs(t, err, ErrNotFound)

	viewer.ShowNSFW = true
	page, err := svc.List(context.Background(), &viewer, "p1", 0, 0)
	require.NoError(t, err)
	require.Len(t, page.Comments, 1)
}

func TestHiddenPostVisibility(t *testing.T) {
	hidden := model.CommentPostAccess{OwnerID: "owner", Rating: "sfw", Hidden: true}
	deleted := model.CommentPostAccess{OwnerID: "owner", Rating: "sfw", Deleted: true}
	for _, access := range []model.CommentPostAccess{hidden, deleted} {
		repo := seed(access)
		svc := NewService(repo)
		memberViewer := member("mem")
		_, err := svc.List(context.Background(), &memberViewer, "p1", 0, 0)
		require.ErrorIs(t, err, ErrNotFound)

		owner := member("owner")
		page, err := svc.List(context.Background(), &owner, "p1", 0, 0)
		require.NoError(t, err)
		require.Len(t, page.Comments, 1)

		mod := model.User{ID: "mod", Role: model.RoleModerator}
		page, err = svc.List(context.Background(), &mod, "p1", 0, 0)
		require.NoError(t, err)
		require.Len(t, page.Comments, 1)
		require.Equal(t, int32(40), repo.lastLimit)
	}
}

func TestAuthorCanEdit(t *testing.T) {
	repo := seed(publicPost())
	svc := NewService(repo)
	got, err := svc.Update(context.Background(), member("author"), "c1", "  revised  ")
	require.NoError(t, err)
	require.Equal(t, "revised", got.Body)
	require.Equal(t, "revised", repo.comments[0].Body)
}

func TestOtherMemberCannotEdit(t *testing.T) {
	repo := seed(publicPost())
	svc := NewService(repo)
	_, err := svc.Update(context.Background(), member("other"), "c1", "rewritten")
	require.ErrorIs(t, err, ErrForbidden)
	require.Zero(t, repo.updateCalls)
	require.Equal(t, "hello", repo.comments[0].Body)
}

func TestModeratorCannotRewrite(t *testing.T) {
	repo := seed(publicPost())
	svc := NewService(repo)
	mod := model.User{ID: "mod", Role: model.RoleModerator}
	_, err := svc.Update(context.Background(), mod, "c1", "staff rewrite")
	require.ErrorIs(t, err, ErrForbidden)
	require.Zero(t, repo.updateCalls)
}

func TestOtherMemberCannotDelete(t *testing.T) {
	repo := seed(publicPost())
	svc := NewService(repo)
	err := svc.Delete(context.Background(), member("other"), "c1")
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, repo.deleted)
}

func TestModeratorCanDelete(t *testing.T) {
	for _, role := range []model.Role{model.RoleModerator, model.RoleAdmin, model.RoleSuperAdmin} {
		repo := seed(publicPost())
		svc := NewService(repo)
		err := svc.Delete(context.Background(), model.User{ID: "staff", Role: role}, "c1")
		require.NoError(t, err)
		require.Equal(t, []string{"c1"}, repo.deleted)
	}
	repo := seed(publicPost())
	err := NewService(repo).Delete(context.Background(), member("author"), "c1")
	require.NoError(t, err)
	require.Equal(t, []string{"c1"}, repo.deleted)
}

func TestListInvisiblePostNotFound(t *testing.T) {
	repo := seed(model.CommentPostAccess{OwnerID: "owner", Rating: "sfw", Hidden: true})
	viewer := member("mem")
	_, err := NewService(repo).List(context.Background(), &viewer, "p1", 0, 0)
	require.ErrorIs(t, err, ErrNotFound)

	missing := &fakeRepo{accessErr: sql.ErrNoRows}
	_, err = NewService(missing).List(context.Background(), nil, "missing", 0, 0)
	require.ErrorIs(t, err, ErrNotFound)

	hidden := seed(model.CommentPostAccess{OwnerID: "owner", Rating: "nsfw", Hidden: true})
	err = NewService(hidden).Delete(context.Background(), model.User{ID: "mod", Role: model.RoleModerator}, "c1")
	require.ErrorIs(t, err, ErrNotFound)
	require.Empty(t, hidden.deleted)
}
