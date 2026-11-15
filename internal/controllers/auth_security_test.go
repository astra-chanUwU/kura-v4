package controllers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

// authRepo embeds the accounts store so an unexpected call panics.
// Tests fill in only the methods the handler is supposed to reach.
type authRepo struct {
	accounts.Repository
	replaced bool
}

func (r *authRepo) GetUserByID(context.Context, string) (model.User, error) {
	return model.User{ID: "u1", Username: "some_user"}, nil
}

func (r *authRepo) ReplacePassword(context.Context, string, string) error {
	r.replaced = true
	return nil
}

func (r *authRepo) CreateSession(context.Context, []byte, string, time.Time) error {
	return nil
}

func (r *authRepo) HasPassword(context.Context, string) (bool, error) { return true, nil }

func (r *authRepo) ListCredentials(context.Context, string) ([]model.StoredCredential, error) {
	return []model.StoredCredential{}, nil
}

func (r *authRepo) GetRecoveryKey(context.Context, string) ([]byte, time.Time, error) {
	return nil, time.Time{}, model.ErrNotFound
}

func TestSetPasswordRotatesSessionCookie(t *testing.T) {
	repo := &authRepo{}
	svc, err := accounts.NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "u1", Username: "some_user"})
		return c.Next()
	})
	NewAuth(svc, nil, nil, false).RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodPost, "/me/password", strings.NewReader(`{"password":"a brand new passphrase"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, repo.replaced)

	var session *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == sessionCookie {
			session = cookie
		}
	}
	require.NotNil(t, session)
	require.NotEmpty(t, session.Value)
	require.True(t, session.HttpOnly)
}

func TestSetAvatarRejectsFilenameSpoofedNonImage(t *testing.T) {
	svc, err := accounts.NewService(&authRepo{}, "http://localhost:5174", nil)
	require.NoError(t, err)
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals(localUser, model.User{ID: "u1", Username: "some_user"})
		return c.Next()
	})
	NewAuth(svc, nil, nil, false).RegisterRoutes(app)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "x.png")
	require.NoError(t, err)
	// No image signature, so the type is only an image if the .png name is trusted.
	_, err = part.Write([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07})
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/me/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, string(payload), "must be an image")
}
