package agentcli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCatalogRoles(t *testing.T) {
	require.True(t, Allows("visitor", "visitor"))
	require.False(t, Allows("visitor", "member"))
	require.False(t, Allows("", "member"))
	require.True(t, Allows("member", "member"))
	require.False(t, Allows("member", "moderator"))
	require.True(t, Allows("moderator", "member"))
	require.True(t, Allows("admin", "moderator"))
	require.True(t, Allows("super_admin", "admin"))
	require.False(t, Allows("moderator", "admin"))

	names := map[string]bool{}
	for _, cmd := range Catalog() {
		require.NotEmpty(t, cmd.Name)
		require.NotEmpty(t, cmd.Usage)
		require.NotContains(t, names, cmd.Name)
		names[cmd.Name] = true
		require.Contains(t, []string{"visitor", "member", "moderator", "admin"}, cmd.MinRole)
	}
	require.Contains(t, names, "posts.upload")
	require.Contains(t, names, "posts.delete")
	require.Contains(t, names, "users.role")
	require.Contains(t, names, "history.rollback")

	visitor := Allowed("visitor")
	for _, cmd := range visitor {
		require.Equal(t, "visitor", cmd.MinRole)
	}
	require.Greater(t, len(Allowed("super_admin")), len(visitor))
}

func TestParseArgs(t *testing.T) {
	got, err := parseArgs([]string{"upload", "pic.jpg", "--artist", "Ada", "--tags=a,b", "--rating", "sfw"})
	require.NoError(t, err)
	require.Equal(t, []string{"upload", "pic.jpg"}, got.Pos)
	require.Equal(t, "Ada", got.Get("artist"))
	require.Equal(t, "a,b", got.Get("tags"))
	require.True(t, got.Has("rating"))
	require.False(t, got.Has("title"))

	_, err = parseArgs([]string{"--artist"})
	require.Error(t, err)
	_, err = parseArgs([]string{"--artist", "--tags", "a"})
	require.Error(t, err)
	_, err = parseArgs([]string{"--nope=1", "--nope=2"})
	require.Error(t, err)

	n, err := got.Int("limit", 40, 1, 100)
	require.NoError(t, err)
	require.Equal(t, 40, n)
	withLimit, err := parseArgs([]string{"--limit", "10"})
	require.NoError(t, err)
	n, err = withLimit.Int("limit", 40, 1, 100)
	require.NoError(t, err)
	require.Equal(t, 10, n)
	_, err = withLimit.Int("limit", 40, 1, 5)
	require.Error(t, err)
	require.NoError(t, got.Only("artist", "tags", "rating"))
	require.Error(t, got.Only("artist"))
}

func TestSessionRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "agent-session")
	_, err := LoadSession(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, SaveSession(path, "token.abc"))
	info, err := os.Stat(path)
	require.NoError(t, err)
	if info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Fatalf("session file is group/world readable: %v", info.Mode().Perm())
	}
	got, err := LoadSession(path)
	require.NoError(t, err)
	require.Equal(t, "token.abc", got)
	require.NoError(t, SaveSession(path, "token.replaced"))
	got, err = LoadSession(path)
	require.NoError(t, err)
	require.Equal(t, "token.replaced", got)
	require.NoError(t, ClearSession(path))
	require.NoError(t, ClearSession(path))
	_, err = LoadSession(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Error(t, SaveSession(path, "has space"))
}

func TestCanSee(t *testing.T) {
	owner := "owner"
	post := model.Post{ID: "p", Rating: "sfw", OwnerID: &owner}
	require.True(t, CanSee(nil, post))
	post.Hidden = true
	require.False(t, CanSee(nil, post))
	member := &model.User{ID: "other", Role: model.RoleMember}
	require.False(t, CanSee(member, post))
	ownerUser := &model.User{ID: owner, Role: model.RoleMember}
	require.True(t, CanSee(ownerUser, post))
	mod := &model.User{ID: "mod", Role: model.RoleModerator}
	require.True(t, CanSee(mod, post))
	post.Hidden = false
	post.Rating = "nsfw"
	require.False(t, CanSee(mod, post))
	mod.ShowNSFW = true
	require.True(t, CanSee(mod, post))
}

func TestSchemaAndUsage(t *testing.T) {
	var buf bytes.Buffer
	code := Execute(context.Background(), Deps{}, []string{"schema"}, &buf, Options{SessionPath: filepath.Join(t.TempDir(), "none")})
	require.Equal(t, 0, code)
	var env Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	require.True(t, env.OK)
	require.Equal(t, "schema", env.Command)

	buf.Reset()
	code = Execute(context.Background(), Deps{}, nil, &buf, Options{})
	require.Equal(t, 2, code)
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	require.False(t, env.OK)
	require.Equal(t, "usage", env.Error.Code)
}
