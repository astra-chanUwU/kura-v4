package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestSearchOffsetDoesNotWrap(t *testing.T) {
	repo := &pageRepo{total: 3}
	app := fiber.New()
	NewPosts(board.NewService(repo, nil), nil).RegisterRoutes(app)

	resp, payload := getJSON(t, app, "/search?offset=3000000000&limit=6442450944")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(math.MaxInt32), repo.query.Offset)
	require.Equal(t, int32(100), repo.query.Limit)
	require.False(t, repo.query.IncludeNSFW)
	require.JSONEq(t, `{"total":3,"posts":[]}`, payload)
}

func TestListOffsetAndVisibility(t *testing.T) {
	repo := &pageRepo{listed: []model.Post{
		{ID: "sfw", Rating: "sfw"},
		{ID: "nsfw", Rating: "nsfw"},
		{ID: "hidden", Hidden: true, Rating: "sfw"},
		{ID: "deleted", Deleted: true, Rating: "sfw"},
	}}
	app := fiber.New()
	NewPosts(board.NewService(repo, nil), collections.NewService(emptyFavs{})).RegisterRoutes(app)

	resp, payload := getJSON(t, app, "/posts?offset=3000000000&limit=6442450944")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, int32(math.MaxInt32), repo.offset)
	require.Equal(t, int32(100), repo.limit)
	require.False(t, repo.nsfw)
	require.Contains(t, payload, `"id":"sfw"`)
	require.NotContains(t, payload, `"id":"nsfw"`)
	require.NotContains(t, payload, `"id":"hidden"`)
	require.NotContains(t, payload, `"id":"deleted"`)

	opted := fiber.New()
	opted.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "user-1", Username: "ada", ShowNSFW: true})
		return c.Next()
	})
	NewPosts(board.NewService(repo, nil), collections.NewService(emptyFavs{})).RegisterRoutes(opted)
	resp, payload = getJSON(t, opted, "/posts?limit=40")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, repo.nsfw)
	require.Contains(t, payload, `"id":"sfw"`)
	require.Contains(t, payload, `"id":"nsfw"`)
	require.NotContains(t, payload, `"id":"hidden"`)
	require.NotContains(t, payload, `"id":"deleted"`)
}

func TestCreateLookupErrorDoesNotStoreAnotherPost(t *testing.T) {
	repo := &shaRepo{first: errors.New("lookup failed")}
	status, payload := uploadJPEG(t, repo, model.User{ID: "user-1", Username: "ada"})
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, 0, repo.creates)
	require.Equal(t, 1, repo.gets)
	require.NotContains(t, payload, "created")
}

func TestCreateDuplicateReturnsExistingPost(t *testing.T) {
	repo := &shaRepo{later: model.Post{ID: "old-post", Rating: "sfw"}}
	status, payload := uploadJPEG(t, repo, model.User{ID: "user-1", Username: "ada"})
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, 0, repo.creates)
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload), &body))
	post, ok := body["post"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "old-post", post["id"])
}

func TestCreateDuplicateOmitsPostTheViewerCannotRead(t *testing.T) {
	repo := &shaRepo{later: model.Post{ID: "nsfw-post", Rating: "nsfw", Hidden: true}}
	status, payload := uploadJPEG(t, repo, model.User{ID: "user-1", Username: "ada"})
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, 0, repo.creates)
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload), &body))
	_, ok := body["post"]
	require.False(t, ok)
	require.NotContains(t, payload, "nsfw-post")
}

func TestCreateUniqueRaceReturnsExistingPost(t *testing.T) {
	repo := &shaRepo{
		first:     model.ErrNotFound,
		later:     model.Post{ID: "raced-post", Rating: "sfw"},
		createErr: errors.New(`duplicate key value violates unique constraint "posts_sha256_idx" (SQLSTATE 23505)`),
	}
	status, payload := uploadJPEG(t, repo, model.User{ID: "user-1", Username: "ada"})
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, 1, repo.creates)
	require.Equal(t, 2, repo.gets)
	require.Contains(t, payload, "raced-post")
}

func TestCreateFailureAfterInsertIsNotADuplicate(t *testing.T) {
	repo := &shaRepo{
		first:     model.ErrNotFound,
		later:     model.Post{ID: "just-stored", Rating: "sfw"},
		createErr: errors.New("wiki down"),
	}
	status, payload := uploadJPEG(t, repo, model.User{ID: "user-1", Username: "ada"})
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, 1, repo.creates)
	require.NotContains(t, payload, "just-stored")
	require.NotContains(t, payload, "duplicate_post")
}

type pageRepo struct {
	quotaRepo
	query  model.PostQuery
	limit  int32
	offset int32
	nsfw   bool
	total  int64
	listed []model.Post
}

func (r *pageRepo) SearchPosts(_ context.Context, q model.PostQuery) ([]model.Post, error) {
	r.query = q
	return []model.Post{}, nil
}

func (r *pageRepo) CountBoardMatches(context.Context, model.PostQuery) (int64, error) {
	return r.total, nil
}

func (r *pageRepo) ListPosts(_ context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	r.limit = limit
	r.offset = offset
	r.nsfw = includeNSFW
	if r.listed == nil {
		return []model.Post{}, nil
	}
	return r.listed, nil
}

type shaRepo struct {
	quotaRepo
	gets      int
	creates   int
	first     error
	later     model.Post
	createErr error
}

func (r *shaRepo) CountPostsByOwnerSince(context.Context, string, time.Duration) (int64, error) {
	return 0, nil
}

func (r *shaRepo) GetPostBySHA(context.Context, string) (model.Post, error) {
	r.gets++
	if r.gets == 1 {
		if r.first != nil {
			return model.Post{}, r.first
		}
		return r.later, nil
	}
	return r.later, nil
}

func (r *shaRepo) CreatePost(context.Context, model.Post, string) (model.Post, error) {
	r.creates++
	if r.createErr != nil {
		return model.Post{}, r.createErr
	}
	return model.Post{ID: "created"}, nil
}

func uploadJPEG(t *testing.T, repo board.Repository, user model.User) (int, string) {
	t.Helper()
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, user)
		return c.Next()
	})
	NewPosts(board.NewService(repo, &quotaMedia{}), collections.NewService(emptyFavs{})).RegisterRoutes(app)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "x.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xD9})
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/posts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(payload)
}

func getJSON(t *testing.T, app *fiber.App, path string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(payload)
}

type emptyFavs struct{}

func (emptyFavs) AddFavorite(context.Context, string, string) error { return nil }
func (emptyFavs) RemoveFavorite(context.Context, string, string) error {
	return nil
}
func (emptyFavs) FavoriteIDs(context.Context, string) ([]string, error) {
	return []string{}, nil
}
func (emptyFavs) ListFavorites(context.Context, string, bool, int32, int32) ([]model.Post, error) {
	return nil, nil
}
func (emptyFavs) CountFavorites(context.Context, string, bool) (int64, error) { return 0, nil }
func (emptyFavs) CreateList(context.Context, string, string, string, string) (model.List, error) {
	return model.List{}, nil
}
func (emptyFavs) GetList(context.Context, string) (model.List, error) { return model.List{}, nil }
func (emptyFavs) ListsByOwner(context.Context, string, bool) ([]model.List, error) {
	return nil, nil
}
func (emptyFavs) UpdateList(context.Context, string, string, string, string, string) (model.List, error) {
	return model.List{}, nil
}
func (emptyFavs) DeleteList(context.Context, string, string) (int64, error) { return 0, nil }
func (emptyFavs) ListItems(context.Context, string, bool, bool, []string, int32, int32) ([]model.Post, error) {
	return nil, nil
}
func (emptyFavs) CountListItems(context.Context, string, bool, bool, []string) (int64, error) {
	return 0, nil
}
func (emptyFavs) ListItemIDs(context.Context, string) ([]string, error) { return nil, nil }
func (emptyFavs) NextPosition(context.Context, string) (int32, error)   { return 0, nil }
func (emptyFavs) AddItem(context.Context, string, string, int32) error  { return nil }
func (emptyFavs) RemoveItem(context.Context, string, string) error      { return nil }
func (emptyFavs) Reorder(context.Context, string, []string) error       { return nil }
