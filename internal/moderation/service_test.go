package moderation

import (
	"context"
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
	repo2.EXPECT().ApplyPostSnapshot(mock.Anything, mock.MatchedBy(func(p model.Post) bool {
		return p.Artist == "good" && len(p.Tags) == 1 && p.Tags[0] == "keep" && p.Rating == "sfw"
	}), mock.MatchedBy(func(e model.ChangeEntry) bool {
		return e.Action == "rollback" && e.RollbackOf != nil && *e.RollbackOf == "h1"
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
	_, err := NewService(repo).ListHiddenPosts(context.Background(), member, "all", 0, 0)
	require.ErrorIs(t, err, ErrForbidden)

	repo2 := NewMockRepository(t)
	want := []model.Post{{ID: "p1", Hidden: true, Deleted: true}, {ID: "p2", Hidden: true}}
	repo2.EXPECT().CountRemovedPosts(mock.Anything).Return(int64(1), int64(1), nil)
	repo2.EXPECT().ListHiddenPosts(mock.Anything, "all", int32(40), int32(0)).Return(want, nil)
	got, err := NewService(repo2).ListHiddenPosts(context.Background(), moderator, "", 0, 0)
	require.NoError(t, err)
	require.Equal(t, want, got.Posts)
	require.Equal(t, int64(2), got.Total)
	require.Equal(t, int64(1), got.Hidden)
	require.Equal(t, int64(1), got.Deleted)
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
	repo2.EXPECT().ApplyPostSnapshot(mock.Anything, mock.MatchedBy(func(p model.Post) bool {
		return !p.Hidden && !p.Deleted
	}), mock.Anything).Return(nil)
	_, err = NewService(repo2).Rollback(context.Background(), moderator, "h-del")
	require.NoError(t, err)
}
