package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

// profileAccounts serves one public profile. Other account methods are unused.
type profileAccounts struct{ accounts.Repository }

func (profileAccounts) GetUserByUsername(context.Context, string) (model.User, error) {
	return model.User{
		ID: "user-ada", Username: "ada", DisplayName: "Ada", Role: model.RoleMember,
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}, nil
}

// profileLists has no public lists. Other collection methods are unused.
type profileLists struct{ collections.Repository }

func (profileLists) ListsByOwner(context.Context, string, bool) ([]model.List, error) {
	return []model.List{}, nil
}

// profileBoardRepo records the public-upload page the profile handler asks for.
type profileBoardRepo struct {
	quotaRepo
	ownerID     string
	includeNSFW bool
	limit       int32
	offset      int32
}

func (r *profileBoardRepo) CountPublicPostsByOwner(context.Context, string, bool) (int64, error) {
	return 48, nil
}

func (r *profileBoardRepo) ListPublicPostsByOwner(_ context.Context, ownerID string, includeNSFW bool, limit, offset int32) ([]model.Post, error) {
	r.ownerID = ownerID
	r.includeNSFW = includeNSFW
	r.limit = limit
	r.offset = offset
	return []model.Post{{ID: "from-offset", Rating: "sfw"}}, nil
}

func TestPublicProfilePageTwoAsksForOffset24(t *testing.T) {
	repo := &profileBoardRepo{}
	accountsSvc, err := accounts.NewService(profileAccounts{}, "http://localhost:5174", nil)
	require.NoError(t, err)
	app := fiber.New()
	NewAuth(accountsSvc, collections.NewService(profileLists{}), board.NewService(repo, nil), false).RegisterRoutes(app)

	ask := func(rawURL string) dto.Profile {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, rawURL, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body dto.Profile
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		return body
	}

	body := ask("/users/ada?page=2")
	require.Equal(t, "user-ada", repo.ownerID)
	require.False(t, repo.includeNSFW)
	require.Equal(t, int32(24), repo.limit)
	require.Equal(t, int32(24), repo.offset)
	require.Equal(t, int64(48), body.UploadTotal)
	require.Len(t, body.Uploads, 1)
	require.Equal(t, "from-offset", body.Uploads[0].ID)

	repo.offset = -1
	body = ask("/users/ada")
	require.Equal(t, int32(0), repo.offset)
	require.Equal(t, int32(24), repo.limit)
	require.Equal(t, int64(48), body.UploadTotal)

	for _, rawURL := range []string{"/users/ada?page=0", "/users/ada?page=nope", "/users/ada?page=1.5", "/users/ada?page="} {
		repo.offset = -1
		_ = ask(rawURL)
		require.Equal(t, int32(0), repo.offset, rawURL)
	}
}
