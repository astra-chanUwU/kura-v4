package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/commentsafety"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

type safetyQueue struct {
	limit    int32
	offset   int32
	filter   model.CommentSafetyFilter
	dismiss  string
	remove   string
	reopen   string
	resolver string
	missing  bool
}

func (q *safetyQueue) List(_ context.Context, filter model.CommentSafetyFilter, limit, offset int32) ([]model.CommentSafetyEvent, error) {
	q.filter, q.limit, q.offset = filter, limit, offset
	return []model.CommentSafetyEvent{{
		ID: "e1", CommentID: "", PostID: "p1", AuthorID: "secret-author", AuthorUsername: "ada",
		Action: "blocked", Categories: nil, Body: "555-123-4567", Status: "open",
		CreatedAt: time.Date(2026, 11, 19, 1, 2, 3, 0, time.FixedZone("X", 3600)),
	}}, nil
}

func (q *safetyQueue) Count(context.Context, model.CommentSafetyFilter) (int64, error) { return 1, nil }

func (q *safetyQueue) Resolvers(context.Context) ([]model.CommentSafetyResolver, error) {
	return []model.CommentSafetyResolver{{
		ID: "22222222-2222-4222-8222-222222222222", Username: "kiri", Decisions: 2,
		LastStatus: "dismissed", LastAction: "blocked", LastCategories: nil,
		LastBody: "https://t.me/example_user",
		LastAt:   time.Date(2026, 11, 19, 2, 3, 4, 0, time.UTC),
	}}, nil
}

func (q *safetyQueue) Dismiss(_ context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	if q.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	q.dismiss, q.resolver = id, resolverID
	return model.CommentSafetyEvent{
		ID: id, CommentID: "c1", PostID: "p1", AuthorUsername: "ada", Action: "flagged",
		Categories: []string{"solicitation"}, Body: "dm me", Status: "dismissed",
		CreatedAt:  time.Date(2026, 11, 19, 1, 2, 3, 0, time.UTC),
		ResolvedAt: time.Date(2026, 11, 19, 4, 5, 6, 0, time.UTC), ResolverID: resolverID,
	}, nil
}

func (q *safetyQueue) Reopen(_ context.Context, id, _ string) (model.CommentSafetyEvent, error) {
	if q.missing {
		return model.CommentSafetyEvent{}, model.ErrNotFound
	}
	q.reopen = id
	return model.CommentSafetyEvent{
		ID: id, CommentID: "c9", PostID: "p1", AuthorUsername: "ada", Action: "flagged",
		Categories: []string{"email"}, Body: "reach me", Status: "open",
		CreatedAt: time.Date(2026, 11, 19, 1, 2, 3, 0, time.UTC),
	}, nil
}

func (q *safetyQueue) Remove(_ context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	q.remove, q.resolver = id, resolverID
	return model.CommentSafetyEvent{
		ID: id, PostID: "p1", AuthorUsername: "ada", Action: "blocked",
		Categories: []string{"phone"}, Body: "555-123-4567", Status: "removed",
		CreatedAt: time.Date(2026, 11, 19, 1, 2, 3, 0, time.UTC), ResolverID: resolverID,
	}, nil
}

func TestCommentSafetyQueueJSON(t *testing.T) {
	queue := &safetyQueue{}
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "11111111-1111-4111-8111-111111111111", Username: "kiri", Role: model.RoleModerator})
		return c.Next()
	})
	NewCommentSafety(commentsafety.NewService(queue)).RegisterRoutes(app)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/comment-safety?limit=5&offset=2", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(5), queue.limit)
	require.Equal(t, int32(2), queue.offset)
	require.Equal(t, "open", queue.filter.Status)
	require.Equal(t, "newest", queue.filter.Sort)
	require.Contains(t, string(payload), `"categories":[]`)
	require.Contains(t, string(payload), `"decisions":[]`)
	require.Contains(t, string(payload), `"resolved_at":""`)
	require.Contains(t, string(payload), `"resolver_id":""`)
	require.NotContains(t, string(payload), `"categories":null`)
	require.NotContains(t, string(payload), `"author_id"`)
	var page dto.CommentSafetyPage
	require.NoError(t, json.Unmarshal(payload, &page))
	require.Equal(t, int64(1), page.Total)
	require.Len(t, page.Events, 1)
	require.NotNil(t, page.Events[0].Categories)
	require.Empty(t, page.Events[0].Categories)
	require.Equal(t, "", page.Events[0].CommentID)
	require.Equal(t, "ada", page.Events[0].AuthorUsername)
	require.Equal(t, "blocked", page.Events[0].Action)
	require.Equal(t, "2026-11-19T00:02:03Z", page.Events[0].CreatedAt)
	require.Equal(t, "", page.Events[0].ResolvedAt)

	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/admin/comment-safety/e9/dismiss", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "e9", queue.dismiss)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", queue.resolver)
	var dismissed dto.CommentSafetyEvent
	require.NoError(t, json.Unmarshal(payload, &dismissed))
	require.Equal(t, "dismissed", dismissed.Status)
	require.Equal(t, "c1", dismissed.CommentID)
	require.Equal(t, []string{"solicitation"}, dismissed.Categories)
	require.Equal(t, "2026-11-19T04:05:06Z", dismissed.ResolvedAt)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", dismissed.ResolverID)

	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/admin/comment-safety/e4/remove", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "e4", queue.remove)

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/admin/comment-safety?status=dismissed&action=flagged&category=email&q=painter&resolver=me&sort=oldest", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "dismissed", queue.filter.Status)
	require.Equal(t, "flagged", queue.filter.Action)
	require.Equal(t, "email", queue.filter.Category)
	require.Equal(t, "painter", queue.filter.Query)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", queue.filter.ResolverID)
	require.Equal(t, "oldest", queue.filter.Sort)

	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/admin/comment-safety/e3/reopen", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "e3", queue.reopen)
	var reopened dto.CommentSafetyEvent
	require.NoError(t, json.Unmarshal(payload, &reopened))
	require.Equal(t, "open", reopened.Status)
	require.Equal(t, "c9", reopened.CommentID)

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/admin/comment-safety/resolvers", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, string(payload), `"last_categories":[]`)
	require.NotContains(t, string(payload), `"last_categories":null`)
	var resolvers dto.CommentSafetyResolvers
	require.NoError(t, json.Unmarshal(payload, &resolvers))
	require.Len(t, resolvers.Resolvers, 1)
	require.Equal(t, "kiri", resolvers.Resolvers[0].Username)
	require.Equal(t, int64(2), resolvers.Resolvers[0].Decisions)
	require.Equal(t, "dismissed", resolvers.Resolvers[0].LastStatus)
	require.Equal(t, "2026-11-19T02:03:04Z", resolvers.Resolvers[0].LastAt)
}

func TestCommentSafetyForbiddenAndMissing(t *testing.T) {
	queue := &safetyQueue{}
	memberApp := fiber.New()
	memberApp.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "m1", Role: model.RoleMember})
		return c.Next()
	})
	NewCommentSafety(commentsafety.NewService(queue)).RegisterRoutes(memberApp)
	resp, err := memberApp.Test(httptest.NewRequest(http.MethodGet, "/admin/comment-safety", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	require.Equal(t, "You don't have permission to do that.", string(body))
	require.Zero(t, queue.limit)

	anon := fiber.New()
	NewCommentSafety(commentsafety.NewService(queue)).RegisterRoutes(anon)
	resp, err = anon.Test(httptest.NewRequest(http.MethodPost, "/admin/comment-safety/e1/dismiss", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, "Sign in first.", string(body))

	queue.missing = true
	mod := fiber.New()
	mod.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "mod", Role: model.RoleModerator})
		return c.Next()
	})
	NewCommentSafety(commentsafety.NewService(queue)).RegisterRoutes(mod)
	resp, err = mod.Test(httptest.NewRequest(http.MethodPost, "/admin/comment-safety/missing/dismiss", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, "Not found.", string(body))
}
