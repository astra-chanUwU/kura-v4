package controllers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

// quotaRepo is the board store the upload handler talks to. Only the
// quota count is real; storing the original goes through quotaMedia.
type quotaRepo struct {
	windows []time.Duration
}

func (r *quotaRepo) CountPostsByOwnerSince(_ context.Context, ownerID string, within time.Duration) (int64, error) {
	r.windows = append(r.windows, within)
	if ownerID == "user-1" && within == time.Hour {
		return 20, nil
	}
	return 0, nil
}

func (r *quotaRepo) CreatePost(context.Context, model.Post, string) (model.Post, error) {
	return model.Post{}, nil
}
func (r *quotaRepo) GetPost(context.Context, string) (model.Post, error) {
	return model.Post{}, nil
}
func (r *quotaRepo) ListPosts(context.Context, int32, int32, bool) ([]model.Post, error) {
	return nil, nil
}
func (r *quotaRepo) ExistsBySHA(context.Context, string) (bool, error)      { return false, nil }
func (r *quotaRepo) SetPostThumbnail(context.Context, string, string) error { return nil }
func (r *quotaRepo) SetPostDimensions(context.Context, string, int, int) error {
	return nil
}
func (r *quotaRepo) ReplacePostMeta(context.Context, model.Post) error { return nil }
func (r *quotaRepo) ReplacePostMetaAudited(context.Context, model.Post, model.ChangeEntry) error {
	return nil
}
func (r *quotaRepo) InsertHistory(context.Context, model.ChangeEntry) (model.ChangeEntry, error) {
	return model.ChangeEntry{}, nil
}
func (r *quotaRepo) SearchPosts(context.Context, model.PostQuery) ([]model.Post, error) {
	return nil, nil
}
func (r *quotaRepo) CountBoardMatches(context.Context, model.PostQuery) (int64, error) {
	return 0, nil
}
func (r *quotaRepo) SuggestTags(context.Context, string, model.PostQuery, int32) ([]model.TagCount, error) {
	return nil, nil
}
func (r *quotaRepo) PostNeighbors(context.Context, string, model.PostQuery) (model.Neighbors, error) {
	return model.Neighbors{}, nil
}
func (r *quotaRepo) GetPostBySHA(context.Context, string) (model.Post, error) {
	return model.Post{}, nil
}
func (r *quotaRepo) CountPostsByOwner(context.Context, string) (int64, error) { return 0, nil }
func (r *quotaRepo) CountVisiblePostsByOwner(context.Context, string, bool) (int64, error) {
	return 0, nil
}
func (r *quotaRepo) ListPostsByOwner(context.Context, string, bool, int32, int32) ([]model.Post, error) {
	return nil, nil
}
func (r *quotaRepo) RelatedPosts(context.Context, string, bool, []string, int32) ([]model.Post, error) {
	return nil, nil
}
func (r *quotaRepo) RandomPost(context.Context, model.PostQuery) (model.Post, error) {
	return model.Post{}, nil
}
func (r *quotaRepo) EnsureArtistPage(context.Context, string) error { return nil }
func (r *quotaRepo) TagAliasTargets(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

type quotaMedia struct{ stored bool }

func (m *quotaMedia) Put(context.Context, io.Reader) (string, error) {
	m.stored = true
	return "", nil
}
func (m *quotaMedia) PutHashed(context.Context, io.Reader) (string, string, error) {
	m.stored = true
	return "", "", nil
}
func (m *quotaMedia) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (m *quotaMedia) Delete(context.Context, string) error { return nil }

func TestCreateRejectsFullUploadQuotaBeforeStoring(t *testing.T) {
	repo := &quotaRepo{}
	media := &quotaMedia{}
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "user-1", Username: "ada"})
		return c.Next()
	})
	NewPosts(board.NewService(repo, media), nil).RegisterRoutes(app)

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
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, "Upload limit reached. Try again later.", string(payload))
	require.False(t, media.stored)
	require.Equal(t, []time.Duration{time.Hour}, repo.windows)
}
