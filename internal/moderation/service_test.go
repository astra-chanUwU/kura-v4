package moderation

import (
	"context"
	"strings"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	superAdmin = model.User{ID: "u-super", Username: "root", Role: model.RoleSuperAdmin}
	admin      = model.User{ID: "u-admin", Username: "adm", Role: model.RoleAdmin}
	moderator  = model.User{ID: "u-mod", Username: "mod", Role: model.RoleModerator}
	member     = model.User{ID: "u-mem", Username: "mem", Role: model.RoleMember}
)

func TestAdminCannotGrantAdminOrTouchAdmins(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUser(mock.Anything, "u-mem").Return(member, nil)
	svc := NewService(repo)
	_, err := svc.SetRole(context.Background(), admin, "u-mem", model.RoleAdmin)
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	targetAdmin := model.User{ID: "u-a2", Username: "a2", Role: model.RoleAdmin}
	repo2.EXPECT().GetUser(mock.Anything, "u-a2").Return(targetAdmin, nil)
	_, err = NewService(repo2).SetRole(context.Background(), admin, "u-a2", model.RoleMember)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestAdminPromotesMemberToModeratorWithHistory(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUser(mock.Anything, "u-mem").Return(member, nil)
	repo.EXPECT().ApplyUserRole(mock.Anything, "u-mem", model.RoleModerator, mock.MatchedBy(func(e model.ChangeEntry) bool {
		before, _ := e.RoleBefore()
		after, _ := e.RoleAfter()
		return e.Action == "user.role_changed" && before.Role == model.RoleMember && after.Role == model.RoleModerator
	})).Return(model.User{ID: "u-mem", Username: "mem", Role: model.RoleModerator}, nil)
	got, err := NewService(repo).SetRole(context.Background(), admin, "u-mem", model.RoleModerator)
	require.NoError(t, err)
	require.Equal(t, model.RoleModerator, got.Role)
}

func TestSuperAdminPromotesToAdminAndLastSuperAdminGuard(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUser(mock.Anything, "u-mem").Return(member, nil)
	repo.EXPECT().ApplyUserRole(mock.Anything, "u-mem", model.RoleAdmin, mock.Anything).
		Return(model.User{ID: "u-mem", Role: model.RoleAdmin}, nil)
	_, err := NewService(repo).SetRole(context.Background(), superAdmin, "u-mem", model.RoleAdmin)
	require.NoError(t, err)

	repo2 := NewMockRepository(t)
	repo2.EXPECT().GetUser(mock.Anything, "u-super").Return(superAdmin, nil)
	repo2.EXPECT().CountSuperAdmins(mock.Anything).Return(int64(1), nil)
	_, err = NewService(repo2).SetRole(context.Background(), superAdmin, "u-super", model.RoleMember)
	require.ErrorContains(t, err, "at least one super admin")
}

func TestModeratorCannotManageRoles(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUser(mock.Anything, "u-mem").Return(member, nil)
	_, err := NewService(repo).SetRole(context.Background(), moderator, "u-mem", model.RoleModerator)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestModeratorHidesButCannotDelete(t *testing.T) {
	post := model.Post{ID: "p1", Artist: "a", Tags: []string{"x"}, Rating: "sfw"}
	repo := NewMockRepository(t)
	repo.EXPECT().GetPost(mock.Anything, "p1").Return(post, nil)
	repo.EXPECT().SetPostHidden(mock.Anything, "p1", true, false, "u-mod", mock.MatchedBy(func(e model.ChangeEntry) bool {
		return e.Action == "post.hidden"
	})).Return(nil)
	got, err := NewService(repo).SetPostHidden(context.Background(), moderator, "p1", true, false)
	require.NoError(t, err)
	require.True(t, got.Hidden)

	repo2 := NewMockRepository(t)
	_, err = NewService(repo2).SetPostHidden(context.Background(), moderator, "p1", true, true)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestRollbackPostRestoresBeforeAndNeedsModerator(t *testing.T) {
	entry := model.ChangeEntry{ID: "h1", EntityType: "post", EntityID: "p1", Action: "post.meta_updated",
		Before: model.MarshalPostSnapshot(model.PostSnapshot{Artist: "good", Tags: []string{"keep"}, Rating: "sfw"}),
		After:  model.MarshalPostSnapshot(model.PostSnapshot{Artist: "rogue", Tags: []string{"spam"}, Rating: "nsfw"})}
	current := model.Post{ID: "p1", Artist: "rogue", Tags: []string{"spam"}, Rating: "nsfw"}

	repo := NewMockRepository(t)
	repo.EXPECT().GetHistory(mock.Anything, "h1").Return(entry, nil)
	_, err := NewService(repo).Rollback(context.Background(), member, "h1")
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	repo2.EXPECT().GetHistory(mock.Anything, "h1").Return(entry, nil)
	repo2.EXPECT().GetPost(mock.Anything, "p1").Return(current, nil)
	repo2.EXPECT().EnsureArtistPage(mock.Anything, "good").Return(nil)
	repo2.EXPECT().ApplyPostSnapshot(mock.Anything, mock.MatchedBy(func(p model.Post) bool {
		return p.Artist == "good" && len(p.Tags) == 2 && p.Tags[0] == "good" && p.Tags[1] == "keep" && p.Rating == "sfw"
	}), mock.MatchedBy(func(e model.ChangeEntry) bool {
		after, err := e.PostAfter()
		return err == nil && e.Action == "rollback" && e.RollbackOf != nil && *e.RollbackOf == "h1" &&
			len(after.Tags) == 2 && after.Tags[0] == "good" && after.Tags[1] == "keep"
	})).Return(nil)
	rb, err := NewService(repo2).Rollback(context.Background(), moderator, "h1")
	require.NoError(t, err)
	require.Equal(t, "rollback", rb.Action)
}

func TestRollbackRoleReChecksHierarchy(t *testing.T) {
	// An admin rolling back a promotion-to-admin is out of admin scope.
	entry := model.ChangeEntry{ID: "h2", EntityType: "user", EntityID: "u-mem", Action: "rollback",
		Before: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: model.RoleMember}),
		After:  model.MarshalRoleSnapshot(model.RoleSnapshot{Role: model.RoleAdmin})}
	repo := NewMockRepository(t)
	repo.EXPECT().GetHistory(mock.Anything, "h2").Return(entry, nil)
	repo.EXPECT().GetUser(mock.Anything, "u-mem").Return(model.User{ID: "u-mem", Role: model.RoleAdmin}, nil)
	_, err := NewService(repo).Rollback(context.Background(), admin, "h2")
	require.ErrorIs(t, err, ErrForbidden)
}

func TestListHiddenPostsNeedsModerator(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).ListHiddenPosts(context.Background(), member, "all", "", "", "", 0, 0)
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	want := []model.Post{{ID: "p1", Hidden: true, Deleted: true}, {ID: "p2", Hidden: true}}
	repo2.EXPECT().CountRemovedPosts(mock.Anything).Return(int64(1), int64(1), nil)
	repo2.EXPECT().CountHiddenPosts(mock.Anything, "all", "", "").Return(int64(2), nil)
	repo2.EXPECT().ListHiddenPosts(mock.Anything, "all", "", "", "", int32(40), int32(0)).Return(want, nil)
	got, err := NewService(repo2).ListHiddenPosts(context.Background(), moderator, "", "", "", "", 0, 0)
	require.NoError(t, err)
	require.Equal(t, want, got.Posts)
	require.Equal(t, int64(2), got.Total)
	require.Equal(t, int64(1), got.Hidden)
	require.Equal(t, int64(1), got.Deleted)
}

func TestListHiddenPostsRejectsRating(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).ListHiddenPosts(context.Background(), moderator, "hidden", "ada", "questionable", "newest", 10, 0)
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.EqualError(t, err, "rating must be sfw or nsfw")
}

func TestListHistoryRejectsAction(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).ListHistory(context.Background(), moderator, "", "ada", "nope", "post", "newest", 10, 0)
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.EqualError(t, err, "action is not a history action")
}

func TestListHistoryNeedsModerator(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).ListHistory(context.Background(), member, "", "", "", "", "", 0, 0)
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	repo2.EXPECT().CountHistory(mock.Anything, "", "", "", "").Return(int64(0), nil)
	repo2.EXPECT().ListHistory(mock.Anything, "", "", "", "", "", int32(40), int32(0)).Return(nil, nil)
	got, err := NewService(repo2).ListHistory(context.Background(), moderator, "", "  ", "", "", "", 0, 0)
	require.NoError(t, err)
	require.NotNil(t, got.Entries)
	require.Empty(t, got.Entries)
	require.Equal(t, int64(0), got.Total)
}

func TestListHistoryForwardsFilters(t *testing.T) {
	repo := NewMockRepository(t)
	want := []model.ChangeEntry{{ID: "h1", Action: "post.hidden", EntityType: "post"}}
	repo.EXPECT().CountHistory(mock.Anything, "", "ada", "post.hidden", "post").Return(int64(4), nil)
	repo.EXPECT().ListHistory(mock.Anything, "", "ada", "post.hidden", "post", "oldest", int32(20), int32(5)).Return(want, nil)
	got, err := NewService(repo).ListHistory(context.Background(), moderator, "", "  ada  ", "post.hidden", "post", "oldest", 20, 5)
	require.NoError(t, err)
	require.Equal(t, want, got.Entries)
	require.Equal(t, int64(4), got.Total)
}

func TestListHistoryRejectsFilters(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	cases := []struct {
		name, actorID, query, action, entity, sort, msg string
	}{
		{name: "entity", entity: "comment", msg: "entity must be post or user"},
		{name: "sort", sort: "popular", msg: "sort must be newest or oldest"},
		{name: "actor", actorID: "ada", msg: "actor_id must be a user id"},
		{name: "query", query: strings.Repeat("あ", 81), msg: "q must be 80 characters or fewer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMockRepository(t)
			_, err := NewService(repo).ListHistory(context.Background(), moderator, tc.actorID, tc.query, tc.action, tc.entity, tc.sort, 10, 0)
			require.EqualError(t, err, tc.msg)
		})
	}

	repo := NewMockRepository(t)
	repo.EXPECT().CountHistory(mock.Anything, id, "", "", "").Return(int64(1), nil)
	repo.EXPECT().ListHistory(mock.Anything, id, "", "", "", "newest", int32(40), int32(0)).Return([]model.ChangeEntry{}, nil)
	_, err := NewService(repo).ListHistory(context.Background(), moderator, strings.ToUpper(id), "", "", "", "newest", 0, 0)
	require.NoError(t, err)
}

func TestSearchHistoryActorsNeedsModerator(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).SearchHistoryActors(context.Background(), member, "ada")
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	want := []model.HistoryActor{{ID: "u1", Username: "Ada"}}
	repo2.EXPECT().SearchUsernames(mock.Anything, "ada").Return(want, nil)
	got, err := NewService(repo2).SearchHistoryActors(context.Background(), moderator, "  ada  ")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestSearchHistoryActorsEmptySkipsQuery(t *testing.T) {
	repo := NewMockRepository(t)
	got, err := NewService(repo).SearchHistoryActors(context.Background(), moderator, "   ")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)

	repo2 := NewMockRepository(t)
	_, err = NewService(repo2).SearchHistoryActors(context.Background(), moderator, strings.Repeat("あ", 81))
	require.EqualError(t, err, "q must be 80 characters or fewer")
}

func TestListHiddenPostsForwardsFilters(t *testing.T) {
	repo := NewMockRepository(t)
	want := []model.Post{{ID: "p9", Hidden: true, Artist: "catgirl"}}
	repo.EXPECT().CountRemovedPosts(mock.Anything).Return(int64(8), int64(3), nil)
	repo.EXPECT().CountHiddenPosts(mock.Anything, "all", "catgirl", "sfw").Return(int64(1), nil)
	repo.EXPECT().ListHiddenPosts(mock.Anything, "all", "catgirl", "sfw", "oldest", int32(20), int32(5)).Return(want, nil)
	got, err := NewService(repo).ListHiddenPosts(context.Background(), moderator, "all", "  catgirl  ", "sfw", "oldest", 20, 5)
	require.NoError(t, err)
	require.Equal(t, want, got.Posts)
	require.Equal(t, int64(1), got.Total)
	require.Equal(t, int64(8), got.Hidden)
	require.Equal(t, int64(3), got.Deleted)
}

func TestAdminDeleteMarksDeletedAndRollbackRestoresIt(t *testing.T) {
	post := model.Post{ID: "p1", Artist: "a", Tags: []string{"x"}, Rating: "sfw"}
	repo := NewMockRepository(t)
	repo.EXPECT().GetPost(mock.Anything, "p1").Return(post, nil)
	repo.EXPECT().SetPostHidden(mock.Anything, "p1", true, true, "u-admin", mock.MatchedBy(func(e model.ChangeEntry) bool {
		after, _ := e.PostAfter()
		return e.Action == "post.deleted" && after.Hidden && after.Deleted
	})).Return(nil)
	got, err := NewService(repo).SetPostHidden(context.Background(), admin, "p1", true, true)
	require.NoError(t, err)
	require.True(t, got.Hidden && got.Deleted)

	// Rolling the delete back restores the visible, not-deleted snapshot.
	entry := model.ChangeEntry{ID: "h-del", EntityType: "post", EntityID: "p1", Action: "post.deleted",
		Before: model.MarshalPostSnapshot(model.PostSnapshot{Artist: "a", Tags: []string{"x"}, Rating: "sfw", Hidden: false, Deleted: false}),
		After:  model.MarshalPostSnapshot(model.PostSnapshot{Artist: "a", Tags: []string{"x"}, Rating: "sfw", Hidden: true, Deleted: true})}
	repo2 := NewMockRepository(t)
	repo2.EXPECT().GetHistory(mock.Anything, "h-del").Return(entry, nil)
	repo2.EXPECT().GetPost(mock.Anything, "p1").Return(model.Post{ID: "p1", Artist: "a", Tags: []string{"x"}, Rating: "sfw", Hidden: true, Deleted: true}, nil)
	repo2.EXPECT().EnsureArtistPage(mock.Anything, "a").Return(nil)
	repo2.EXPECT().ApplyPostSnapshot(mock.Anything, mock.MatchedBy(func(p model.Post) bool {
		return !p.Hidden && !p.Deleted
	}), mock.Anything).Return(nil)
	_, err = NewService(repo2).Rollback(context.Background(), moderator, "h-del")
	require.NoError(t, err)
}

func TestListUserPageNeedsAdmin(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).ListUserPage(context.Background(), member, "", "owner", "newest", 10, 0)
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	_, err = NewService(repo2).ListUserPage(context.Background(), moderator, "ada", "member", "name", 10, 0)
	require.ErrorIs(t, err, ErrForbidden)

	repo3 := NewMockRepository(t)
	want := []model.User{{ID: "u1", Username: "ada", Role: model.RoleMember}}
	repo3.EXPECT().CountUsersByRoles(mock.Anything).Return(int64(4), int64(2), int64(1), int64(1), nil)
	repo3.EXPECT().CountFilteredUsers(mock.Anything, "", "").Return(int64(8), nil)
	repo3.EXPECT().ListFilteredUsers(mock.Anything, "", "", "name", int32(40), int32(0)).Return(want, nil)
	got, err := NewService(repo3).ListUserPage(context.Background(), admin, "  ", "", "", 0, 0)
	require.NoError(t, err)
	require.Equal(t, want, got.Users)
	require.Equal(t, int64(8), got.Total)
	require.Equal(t, int64(4), got.Member)
	require.Equal(t, int64(2), got.Moderator)
	require.Equal(t, int64(1), got.Admin)
	require.Equal(t, int64(1), got.SuperAdmin)
}

func TestListUserPageRejectsRoleBeforeQuery(t *testing.T) {
	repo := NewMockRepository(t)
	_, err := NewService(repo).ListUserPage(context.Background(), admin, "ada", "owner", "name", 10, 0)
	require.EqualError(t, err, "role must be member, moderator, admin, or super_admin")
}

func TestListUserPageForwardsFilters(t *testing.T) {
	repo := NewMockRepository(t)
	want := []model.User{{ID: "u9", Username: "bea", DisplayName: "Bea", Role: model.RoleModerator}}
	repo.EXPECT().CountUsersByRoles(mock.Anything).Return(int64(10), int64(3), int64(2), int64(1), nil)
	repo.EXPECT().CountFilteredUsers(mock.Anything, "bea", "moderator").Return(int64(1), nil)
	repo.EXPECT().ListFilteredUsers(mock.Anything, "bea", "moderator", "oldest", int32(20), int32(5)).Return(want, nil)
	got, err := NewService(repo).ListUserPage(context.Background(), superAdmin, "  bea  ", "moderator", "oldest", 20, 5)
	require.NoError(t, err)
	require.Equal(t, want, got.Users)
	require.Equal(t, int64(1), got.Total)
	require.Equal(t, int64(10), got.Member)
	require.Equal(t, int64(3), got.Moderator)
	require.Equal(t, int64(2), got.Admin)
	require.Equal(t, int64(1), got.SuperAdmin)
}

func TestListUserPageRejectsFilters(t *testing.T) {
	cases := []struct {
		name, query, role, sort, msg string
	}{
		{name: "query", query: strings.Repeat("あ", 81), msg: "q must be 80 characters or fewer"},
		{name: "role", role: "visitor", msg: "role must be member, moderator, admin, or super_admin"},
		{name: "sort", sort: "popular", msg: "sort must be name, newest, or oldest"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMockRepository(t)
			_, err := NewService(repo).ListUserPage(context.Background(), admin, tc.query, tc.role, tc.sort, 10, 0)
			require.EqualError(t, err, tc.msg)
		})
	}
}

func TestListUserPageNilUsers(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().CountUsersByRoles(mock.Anything).Return(int64(0), int64(0), int64(0), int64(0), nil)
	repo.EXPECT().CountFilteredUsers(mock.Anything, "", "super_admin").Return(int64(0), nil)
	repo.EXPECT().ListFilteredUsers(mock.Anything, "", "super_admin", "newest", int32(100), int32(0)).Return(nil, nil)
	got, err := NewService(repo).ListUserPage(context.Background(), admin, "", "super_admin", "newest", 500, -4)
	require.NoError(t, err)
	require.NotNil(t, got.Users)
	require.Empty(t, got.Users)
	require.Equal(t, int64(0), got.Total)
}
