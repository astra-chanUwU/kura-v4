package comments

import (
	"context"
	"errors"
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
	return model.Comment{}, model.ErrNotFound
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
	return model.Comment{}, model.ErrNotFound
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
	require.ErrorIs(t, err, model.ErrNotFound)
	_, err = svc.Create(context.Background(), viewer, "p1", "nope")
	require.ErrorIs(t, err, model.ErrNotFound)

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
		require.ErrorIs(t, err, model.ErrNotFound)

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
	require.ErrorIs(t, err, model.ErrNotFound)

	missing := &fakeRepo{accessErr: model.ErrNotFound}
	_, err = NewService(missing).List(context.Background(), nil, "missing", 0, 0)
	require.ErrorIs(t, err, model.ErrNotFound)

	hidden := seed(model.CommentPostAccess{OwnerID: "owner", Rating: "nsfw", Hidden: true})
	err = NewService(hidden).Delete(context.Background(), model.User{ID: "mod", Role: model.RoleModerator}, "c1")
	require.ErrorIs(t, err, model.ErrNotFound)
	require.Empty(t, hidden.deleted)
}

var errSafetyQueue = errors.New("safety queue unavailable")

type fakeSafety struct {
	drafts []model.CommentSafetyDraft
	err    error
}

func (f *fakeSafety) Record(_ context.Context, draft model.CommentSafetyDraft) error {
	f.drafts = append(f.drafts, draft)
	return f.err
}

func TestPlainCommentStillCreates(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	log := &fakeSafety{}
	svc := NewService(repo)
	svc.SetSafety(log)
	got, err := svc.Create(context.Background(), member("mem"), "p1", "  hello  ")
	require.NoError(t, err)
	require.Equal(t, "hello", got.Body)
	require.Equal(t, "hello", repo.comments[0].Body)
	require.Empty(t, log.drafts)
}

func TestPhoneOrTelegramDoesNotPublish(t *testing.T) {
	const message = "Comments can't include phone numbers or off-site contact details."
	bodies := []string{"555-123-4567", "https://t.me/example_user"}
	roles := []model.Role{model.RoleMember, model.RoleModerator, model.RoleAdmin, model.RoleSuperAdmin}
	for _, role := range roles {
		for _, body := range bodies {
			repo := &fakeRepo{access: publicPost()}
			log := &fakeSafety{}
			svc := NewService(repo)
			svc.SetSafety(log)
			_, err := svc.Create(context.Background(), model.User{ID: "mem", Role: role}, "p1", "  "+body+"  ")
			var vErr *ValidationError
			require.ErrorAs(t, err, &vErr)
			require.Equal(t, message, vErr.Message)
			require.NotContains(t, err.Error(), "555")
			require.NotContains(t, err.Error(), "t.me")
			require.Empty(t, repo.comments)
			require.Len(t, log.drafts, 1)
			require.Equal(t, "blocked", log.drafts[0].Action)
			require.Empty(t, log.drafts[0].CommentID)
			require.Equal(t, "p1", log.drafts[0].PostID)
			require.Equal(t, "mem", log.drafts[0].AuthorID)
			require.Equal(t, body, log.drafts[0].Body)
			if strings.Contains(body, "t.me") {
				require.Contains(t, log.drafts[0].Categories, "telegram")
			} else {
				require.Contains(t, log.drafts[0].Categories, "phone")
			}
		}
	}
}

func TestBlockWithoutSafetyLog(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	_, err := NewService(repo).Create(context.Background(), member("mem"), "p1", "555-123-4567")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Equal(t, "Comments can't include phone numbers or off-site contact details.", vErr.Message)
	require.Empty(t, repo.comments)
}

func TestFailingLogOnBlockStillRejects(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	log := &fakeSafety{err: errSafetyQueue}
	svc := NewService(repo)
	svc.SetSafety(log)
	_, err := svc.Create(context.Background(), member("mem"), "p1", "555-123-4567")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Equal(t, "Comments can't include phone numbers or off-site contact details.", vErr.Message)
	require.NotErrorIs(t, err, errSafetyQueue)
	require.Empty(t, repo.comments)
	require.Equal(t, "blocked", log.drafts[0].Action)
}

func TestSolicitationOrEmailIsFlagged(t *testing.T) {
	for _, body := range []string{"dm me", "ada@example.com"} {
		repo := &fakeRepo{access: publicPost()}
		log := &fakeSafety{}
		svc := NewService(repo)
		svc.SetSafety(log)
		got, err := svc.Create(context.Background(), member("mem"), "p1", "  "+body+"  ")
		require.NoError(t, err)
		require.Equal(t, body, got.Body)
		require.Equal(t, body, repo.comments[0].Body)
		require.Len(t, log.drafts, 1)
		require.Equal(t, "flagged", log.drafts[0].Action)
		require.Equal(t, got.ID, log.drafts[0].CommentID)
		require.Equal(t, "p1", log.drafts[0].PostID)
		require.Equal(t, "mem", log.drafts[0].AuthorID)
		require.Equal(t, body, log.drafts[0].Body)
		if strings.Contains(body, "@") {
			require.Contains(t, log.drafts[0].Categories, "email")
		} else {
			require.Contains(t, log.drafts[0].Categories, "solicitation")
		}
	}
}

func TestFlagWithoutSafetyLogStillPublishes(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	got, err := NewService(repo).Create(context.Background(), member("mem"), "p1", "dm me")
	require.NoError(t, err)
	require.Equal(t, "dm me", got.Body)
	require.Len(t, repo.comments, 1)
	require.Empty(t, repo.deleted)
}

func TestFlaggedCreateRollsBackWhenLogFails(t *testing.T) {
	repo := &fakeRepo{access: publicPost()}
	log := &fakeSafety{err: errSafetyQueue}
	svc := NewService(repo)
	svc.SetSafety(log)
	_, err := svc.Create(context.Background(), member("mem"), "p1", "dm me")
	require.ErrorIs(t, err, errSafetyQueue)
	require.Empty(t, repo.comments)
	require.Equal(t, []string{"c-new"}, repo.deleted)
	require.Equal(t, "flagged", log.drafts[0].Action)
	require.Equal(t, "c-new", log.drafts[0].CommentID)
}

func TestUpdateIntoPhoneDoesNotChangeBody(t *testing.T) {
	repo := seed(publicPost())
	log := &fakeSafety{}
	svc := NewService(repo)
	svc.SetSafety(log)
	_, err := svc.Update(context.Background(), member("author"), "c1", "  555-123-4567  ")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Equal(t, "Comments can't include phone numbers or off-site contact details.", vErr.Message)
	require.NotContains(t, err.Error(), "555")
	require.Zero(t, repo.updateCalls)
	require.Equal(t, "hello", repo.comments[0].Body)
	require.Len(t, log.drafts, 1)
	require.Equal(t, "blocked", log.drafts[0].Action)
	require.Equal(t, "c1", log.drafts[0].CommentID)
	require.Equal(t, "p1", log.drafts[0].PostID)
	require.Equal(t, "author", log.drafts[0].AuthorID)
	require.Equal(t, "555-123-4567", log.drafts[0].Body)
	require.Contains(t, log.drafts[0].Categories, "phone")
}

func TestUpdateIntoSolicitationFlags(t *testing.T) {
	repo := seed(publicPost())
	log := &fakeSafety{}
	svc := NewService(repo)
	svc.SetSafety(log)
	got, err := svc.Update(context.Background(), member("author"), "c1", "  dm me  ")
	require.NoError(t, err)
	require.Equal(t, "dm me", got.Body)
	require.Equal(t, "dm me", repo.comments[0].Body)
	require.Equal(t, 1, repo.updateCalls)
	require.Len(t, log.drafts, 1)
	require.Equal(t, "flagged", log.drafts[0].Action)
	require.Equal(t, "c1", log.drafts[0].CommentID)
	require.Equal(t, "p1", log.drafts[0].PostID)
	require.Equal(t, "author", log.drafts[0].AuthorID)
	require.Equal(t, "dm me", log.drafts[0].Body)
	require.Contains(t, log.drafts[0].Categories, "solicitation")
}

func TestFlaggedUpdateRestoresBodyWhenLogFails(t *testing.T) {
	repo := seed(publicPost())
	log := &fakeSafety{err: errSafetyQueue}
	svc := NewService(repo)
	svc.SetSafety(log)
	_, err := svc.Update(context.Background(), member("author"), "c1", "dm me")
	require.ErrorIs(t, err, errSafetyQueue)
	require.Equal(t, "hello", repo.comments[0].Body)
	require.Equal(t, 2, repo.updateCalls)
	require.Equal(t, "flagged", log.drafts[0].Action)
	require.Equal(t, "c1", log.drafts[0].CommentID)
}
