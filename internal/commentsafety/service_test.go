package commentsafety

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type memRepo struct {
	events    []model.CommentSafetyEvent
	nilList   bool
	filter    model.CommentSafetyFilter
	limit     int32
	offset    int32
	dismissed string
	removed   string
	reopened  string
	missing   bool
}

func (m *memRepo) List(_ context.Context, filter model.CommentSafetyFilter, limit, offset int32) ([]model.CommentSafetyEvent, error) {
	m.filter, m.limit, m.offset = filter, limit, offset
	if m.nilList {
		return nil, nil
	}
	return m.events, nil
}

func (m *memRepo) Count(context.Context, model.CommentSafetyFilter) (int64, error) {
	return int64(len(m.events)), nil
}

func (m *memRepo) Resolvers(context.Context) ([]model.CommentSafetyResolver, error) {
	if m.nilList {
		return nil, nil
	}
	return []model.CommentSafetyResolver{{
		ID: "11111111-1111-4111-8111-111111111111", Username: "kiri", Decisions: 2,
		LastStatus: "dismissed", LastAction: "blocked", LastCategories: []string{"telegram"},
		LastBody: "https://t.me/example_user",
	}}, nil
}

func (m *memRepo) Dismiss(_ context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	if m.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	m.dismissed = id
	return model.CommentSafetyEvent{ID: id, CommentID: "c1", Status: "dismissed", ResolverID: resolverID}, nil
}

func (m *memRepo) Remove(_ context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	if m.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	m.removed = id
	return model.CommentSafetyEvent{ID: id, CommentID: "c1", Status: "removed", ResolverID: resolverID}, nil
}

func (m *memRepo) Reopen(_ context.Context, id, actorID string) (model.CommentSafetyEvent, error) {
	if m.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	m.reopened = id
	return model.CommentSafetyEvent{ID: id, CommentID: "c1", Status: "open", ResolverID: actorID}, nil
}

func TestListRequiresModeratorAndEmptiesNil(t *testing.T) {
	repo := &memRepo{nilList: true}
	svc := NewService(repo)
	_, err := svc.List(context.Background(), model.User{}, model.CommentSafetyFilter{}, 0, 0)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = svc.List(context.Background(), model.User{Role: model.RoleModerator}, model.CommentSafetyFilter{}, 0, 0)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = svc.List(context.Background(), model.User{ID: "m1", Role: model.RoleMember}, model.CommentSafetyFilter{}, 0, 0)
	require.ErrorIs(t, err, ErrForbidden)
	require.Zero(t, repo.limit)

	for _, role := range []model.Role{model.RoleModerator, model.RoleAdmin, model.RoleSuperAdmin} {
		page, err := svc.List(context.Background(), model.User{ID: "staff", Role: role}, model.CommentSafetyFilter{Status: "open"}, 0, -5)
		require.NoError(t, err)
		require.NotNil(t, page.Events)
		require.Empty(t, page.Events)
		require.Equal(t, int64(0), page.Total)
		require.Equal(t, int32(40), repo.limit)
		require.Equal(t, int32(0), repo.offset)
		require.Equal(t, "open", repo.filter.Status)
		require.Equal(t, "newest", repo.filter.Sort)
	}

	_, err = svc.List(context.Background(), model.User{ID: "staff", Role: model.RoleModerator}, model.CommentSafetyFilter{
		Status: "any", Action: "blocked", Category: "phone", Query: "  discord  ", Sort: "oldest",
		ResolverID: "11111111-1111-1111-1111-111111111111",
	}, 500, 3)
	require.NoError(t, err)
	require.Equal(t, int32(100), repo.limit)
	require.Equal(t, int32(3), repo.offset)
	require.Equal(t, "", repo.filter.Status)
	require.Equal(t, "blocked", repo.filter.Action)
	require.Equal(t, "phone", repo.filter.Category)
	require.Equal(t, "discord", repo.filter.Query)
	require.Equal(t, "oldest", repo.filter.Sort)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", repo.filter.ResolverID)

	_, err = svc.List(context.Background(), model.User{ID: "staff", Role: model.RoleModerator}, model.CommentSafetyFilter{Sort: "decided"}, 1, 0)
	require.NoError(t, err)
	require.Equal(t, "decided", repo.filter.Sort)

	_, err = svc.List(context.Background(), model.User{ID: "staff", Role: model.RoleModerator}, model.CommentSafetyFilter{Status: "closed"}, 0, 0)
	var invalid *ValidationError
	require.ErrorAs(t, err, &invalid)

	people, err := NewService(&memRepo{}).Resolvers(context.Background(), model.User{ID: "staff", Role: model.RoleModerator})
	require.NoError(t, err)
	require.Len(t, people, 1)
	require.Equal(t, "kiri", people[0].Username)
	_, err = svc.Resolvers(context.Background(), model.User{ID: "m1", Role: model.RoleMember})
	require.ErrorIs(t, err, ErrForbidden)
	empty, err := NewService(&memRepo{nilList: true}).Resolvers(context.Background(), model.User{ID: "staff", Role: model.RoleModerator})
	require.NoError(t, err)
	require.NotNil(t, empty)
	require.Empty(t, empty)
}

func TestDismissLeavesCommentAndRemoveDelegates(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo)
	_, err := svc.Dismiss(context.Background(), model.User{ID: "m1", Role: model.RoleMember}, "e1")
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, repo.dismissed)
	_, err = svc.Remove(context.Background(), model.User{}, "e1")
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, repo.removed)

	got, err := svc.Dismiss(context.Background(), model.User{ID: "mod", Role: model.RoleModerator}, "e1")
	require.NoError(t, err)
	require.Equal(t, "dismissed", got.Status)
	require.Equal(t, "mod", got.ResolverID)
	require.Equal(t, "c1", got.CommentID)
	require.Equal(t, "e1", repo.dismissed)
	require.Empty(t, repo.removed)

	got, err = svc.Remove(context.Background(), model.User{ID: "admin", Role: model.RoleAdmin}, "e2")
	require.NoError(t, err)
	require.Equal(t, "removed", got.Status)
	require.Equal(t, "admin", got.ResolverID)
	require.Equal(t, "e2", repo.removed)
	require.Equal(t, "e1", repo.dismissed)

	missing := NewService(&memRepo{missing: true})
	actor := model.User{ID: "mod", Role: model.RoleModerator}
	_, err = missing.Dismiss(context.Background(), actor, "gone")
	require.ErrorIs(t, err, model.ErrNotFound)
	_, err = missing.Remove(context.Background(), model.User{ID: "root", Role: model.RoleSuperAdmin}, "gone")
	require.ErrorIs(t, err, model.ErrNotFound)

	got, err = svc.Reopen(context.Background(), model.User{ID: "mod", Role: model.RoleModerator}, "e3")
	require.NoError(t, err)
	require.Equal(t, "open", got.Status)
	require.Equal(t, "e3", repo.reopened)
	_, err = svc.Reopen(context.Background(), model.User{ID: "m1", Role: model.RoleMember}, "e3")
	require.ErrorIs(t, err, ErrForbidden)
	_, err = missing.Reopen(context.Background(), actor, "gone")
	require.ErrorIs(t, err, model.ErrNotFound)
}
