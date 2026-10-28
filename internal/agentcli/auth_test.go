package agentcli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestLoginStoresTokenWithoutPrintingIt(t *testing.T) {
	const token = "login-session-token"
	const password = "login-password-value"
	path := filepath.Join(t.TempDir(), "session")
	acc := &authAccounts{
		login: func(ctx context.Context, username, pw string) (model.User, string, error) {
			require.Equal(t, "ada", username)
			require.Equal(t, password, pw)
			return model.User{
				ID: "u-ada", Username: "ada", DisplayName: "Ada",
				Role: model.RoleModerator, ShowNSFW: true,
			}, token, nil
		},
	}
	code, env, raw := authRun(t, Deps{Accounts: acc}, []string{"login", "--username", "ada", "--password", password}, path)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "login", env.Command)
	require.Equal(t, "moderator", authData(t, env)["role"])
	require.NotContains(t, raw, token)
	require.NotContains(t, raw, password)
	got, err := LoadSession(path)
	require.NoError(t, err)
	require.Equal(t, token, got)
}

func TestLoginMissingPassword(t *testing.T) {
	t.Setenv("KURA_AGENT_PASSWORD", "")
	path := filepath.Join(t.TempDir(), "session")
	code, env, _ := authRun(t, Deps{Accounts: &authAccounts{}}, []string{"login", "--username", "ada"}, path)
	require.Equal(t, 2, code)
	require.False(t, env.OK)
	require.Equal(t, "login", env.Command)
	require.NotNil(t, env.Error)
	require.Equal(t, "usage", env.Error.Code)
}

func TestLoginInvalidCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	acc := &authAccounts{
		login: func(ctx context.Context, username, password string) (model.User, string, error) {
			return model.User{}, "", accounts.ErrInvalidCredentials
		},
	}
	code, env, _ := authRun(t, Deps{Accounts: acc}, []string{"login", "--username", "ada", "--password", "nope"}, path)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.NotNil(t, env.Error)
	require.Equal(t, "unauthenticated", env.Error.Code)
	_, err := LoadSession(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestWhoamiVisitor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	code, env, _ := authRun(t, Deps{}, []string{"whoami"}, path)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "whoami", env.Command)
	require.Equal(t, "visitor", authData(t, env)["role"])
}

func TestWhoamiSession(t *testing.T) {
	path, acc := authSignIn(t, model.RoleMember, "ada")
	code, env, _ := authRun(t, Deps{Accounts: acc}, []string{"whoami"}, path)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "ada", authData(t, env)["username"])
}

func TestWhoamiExpiredSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	require.NoError(t, SaveSession(path, "expired-token"))
	acc := &authAccounts{
		session: func(ctx context.Context, token string) (model.User, error) {
			return model.User{}, accounts.ErrNotFound
		},
	}
	code, env, _ := authRun(t, Deps{Accounts: acc}, []string{"whoami"}, path)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.Equal(t, "whoami", env.Command)
	require.NotNil(t, env.Error)
	require.Equal(t, "unauthenticated", env.Error.Code)
}

func TestCapabilitiesByRole(t *testing.T) {
	memberPath, member := authSignIn(t, model.RoleMember, "ada")
	code, env, _ := authRun(t, Deps{Accounts: member}, []string{"capabilities"}, memberPath)
	require.Equal(t, 0, code)
	names := authCommandNames(t, env)
	require.Contains(t, names, "posts.upload")
	require.NotContains(t, names, "posts.hide")

	modPath, mod := authSignIn(t, model.RoleModerator, "mod")
	code, env, _ = authRun(t, Deps{Accounts: mod}, []string{"capabilities"}, modPath)
	require.Equal(t, 0, code)
	require.Contains(t, authCommandNames(t, env), "posts.hide")
}

func TestSiteName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	site := &authSite{
		get: func(ctx context.Context) (model.Site, error) {
			return model.Site{Name: "Aster", Description: "a board"}, nil
		},
	}
	code, env, _ := authRun(t, Deps{Site: site}, []string{"site"}, path)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "site", env.Command)
	require.Equal(t, "Aster", authData(t, env)["name"])
}

func TestLogoutRemovesSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	require.NoError(t, SaveSession(path, "logout-token"))
	var saw string
	acc := &authAccounts{
		logout: func(ctx context.Context, token string) error {
			saw = token
			return nil
		},
	}
	code, env, _ := authRun(t, Deps{Accounts: acc}, []string{"logout"}, path)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, true, authData(t, env)["signed_out"])
	require.Equal(t, "logout-token", saw)
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func authRun(t *testing.T, deps Deps, args []string, session string) (int, Envelope, string) {
	t.Helper()
	var buf bytes.Buffer
	code := Execute(context.Background(), deps, args, &buf, Options{SessionPath: session})
	var env Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	return code, env, buf.String()
}

func authData(t *testing.T, env Envelope) map[string]any {
	t.Helper()
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	return data
}

func authCommandNames(t *testing.T, env Envelope) []string {
	t.Helper()
	raw, ok := authData(t, env)["commands"].([]any)
	require.True(t, ok)
	names := make([]string, 0, len(raw))
	for _, item := range raw {
		cmd, ok := item.(map[string]any)
		require.True(t, ok)
		name, ok := cmd["name"].(string)
		require.True(t, ok)
		names = append(names, name)
	}
	return names
}

func authSignIn(t *testing.T, role model.Role, username string) (string, *authAccounts) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session")
	token := "token-" + username
	require.NoError(t, SaveSession(path, token))
	acc := &authAccounts{
		session: func(ctx context.Context, got string) (model.User, error) {
			require.Equal(t, token, got)
			return model.User{ID: "id-" + username, Username: username, DisplayName: username, Role: role}, nil
		},
	}
	return path, acc
}

type authAccounts struct {
	Accounts
	login   func(context.Context, string, string) (model.User, string, error)
	session func(context.Context, string) (model.User, error)
	logout  func(context.Context, string) error
}

func (f *authAccounts) LoginWithPassword(ctx context.Context, username, password string) (model.User, string, error) {
	if f.login == nil {
		panic("LoginWithPassword")
	}
	return f.login(ctx, username, password)
}

func (f *authAccounts) UserForSession(ctx context.Context, token string) (model.User, error) {
	if f.session == nil {
		panic("UserForSession")
	}
	return f.session(ctx, token)
}

func (f *authAccounts) Logout(ctx context.Context, token string) error {
	if f.logout == nil {
		panic("Logout")
	}
	return f.logout(ctx, token)
}

type authSite struct {
	get func(context.Context) (model.Site, error)
}

func (s *authSite) Get(ctx context.Context) (model.Site, error) {
	if s.get == nil {
		panic("Get")
	}
	return s.get(ctx)
}
