package agentcli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestPostsListVisitorSkipsNSFW(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	var limit, offset int32
	var includeNSFW bool
	var called bool
	b := &postsBoard{
		list: func(ctx context.Context, lim, off int32, nsfw bool) ([]model.Post, error) {
			called = true
			limit, offset, includeNSFW = lim, off, nsfw
			return nil, nil
		},
	}
	code, env, _ := postsRun(t, Deps{Board: b}, []string{"posts", "list"}, path)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "posts.list", env.Command)
	require.True(t, called)
	require.False(t, includeNSFW)
	require.Equal(t, int32(40), limit)
	require.Equal(t, int32(0), offset)
}

func TestPostsGetHiddenVisitor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	const id = "hidden-post-id"
	b := &postsBoard{
		get: func(ctx context.Context, got string) (model.Post, error) {
			require.Equal(t, id, got)
			return model.Post{ID: id, Rating: "sfw", Hidden: true}, nil
		},
	}
	code, env, raw := postsRun(t, Deps{Board: b}, []string{"posts", "get", id}, path)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.Equal(t, "posts.get", env.Command)
	require.NotNil(t, env.Error)
	require.Equal(t, "not_found", env.Error.Code)
	require.NotContains(t, raw, id)
}

func TestPostsUploadJPEG(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	file := filepath.Join(t.TempDir(), "x.jpg")
	require.NoError(t, os.WriteFile(file, []byte{0xFF, 0xD8, 0xFF, 0xD9}, 0o644))
	var created, recorded bool
	b := &postsBoard{
		upload: func(ctx context.Context, r io.Reader) (string, string, error) {
			return "key-1", "sha-1", nil
		},
		getBySHA: func(ctx context.Context, sha string) (model.Post, error) {
			require.Equal(t, "sha-1", sha)
			return model.Post{}, board.ErrNotFound
		},
		create: func(ctx context.Context, p model.Post, sha string) (model.Post, error) {
			created = true
			require.Equal(t, "sha-1", sha)
			require.Equal(t, "image/jpeg", p.ContentType)
			require.Equal(t, "x.jpg", p.OriginalName)
			require.Equal(t, int64(4), p.SizeBytes)
			require.NotNil(t, p.OwnerID)
			require.Equal(t, "user-1", *p.OwnerID)
			p.ID = "new-post"
			return p, nil
		},
		record: func(ctx context.Context, actor model.User, post model.Post) {
			recorded = true
			require.Equal(t, "ada", actor.Username)
			require.Equal(t, "user-1", actor.ID)
			require.Equal(t, "ada", post.OwnerUsername)
			require.Equal(t, "new-post", post.ID)
		},
	}
	code, env, _ := postsRun(t, Deps{Accounts: acc, Board: b}, []string{"posts", "upload", file}, session)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "posts.upload", env.Command)
	require.True(t, created)
	require.True(t, recorded)
}

func TestPostsUploadDuplicate(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	file := filepath.Join(t.TempDir(), "x.jpg")
	require.NoError(t, os.WriteFile(file, []byte{0xFF, 0xD8, 0xFF, 0xD9}, 0o644))
	created := false
	b := &postsBoard{
		upload: func(ctx context.Context, r io.Reader) (string, string, error) {
			return "key-1", "sha-1", nil
		},
		getBySHA: func(ctx context.Context, sha string) (model.Post, error) {
			return model.Post{ID: "existing-post-7", Rating: "sfw"}, nil
		},
		create: func(ctx context.Context, p model.Post, sha string) (model.Post, error) {
			created = true
			return model.Post{}, nil
		},
	}
	code, env, _ := postsRun(t, Deps{Accounts: acc, Board: b}, []string{"posts", "upload", file}, session)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.NotNil(t, env.Error)
	require.Equal(t, "duplicate_post", env.Error.Code)
	require.False(t, created)
	post, ok := postsData(t, env)["post"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "existing-post-7", post["id"])
}

func TestPostsUploadRejectsText(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	file := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0o644))
	uploaded := false
	b := &postsBoard{
		upload: func(ctx context.Context, r io.Reader) (string, string, error) {
			uploaded = true
			return "", "", nil
		},
	}
	code, env, _ := postsRun(t, Deps{Accounts: acc, Board: b}, []string{"posts", "upload", file}, session)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.NotNil(t, env.Error)
	require.Equal(t, "validation", env.Error.Code)
	require.False(t, uploaded)
}

func TestPostsEditRatingKeepsArtistAndTags(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	b := &postsBoard{
		get: func(ctx context.Context, id string) (model.Post, error) {
			require.Equal(t, "p1", id)
			return model.Post{ID: "p1", Artist: "mori", Tags: []string{"bird", "cat"}, Rating: "sfw"}, nil
		},
		setMeta: func(ctx context.Context, actor *model.User, id, artist string, tags []string, rating string) (model.Post, error) {
			require.NotNil(t, actor)
			require.Equal(t, "p1", id)
			require.Equal(t, "mori", artist)
			require.Equal(t, []string{"bird", "cat"}, tags)
			require.Equal(t, "nsfw", rating)
			return model.Post{ID: id, Artist: artist, Tags: tags, Rating: rating}, nil
		},
	}
	code, env, _ := postsRun(t, Deps{Accounts: acc, Board: b}, []string{"posts", "edit", "p1", "--rating", "nsfw"}, session)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "posts.edit", env.Command)
}

func TestPostsHideRequiresModerator(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	called := false
	mod := &postsModeration{
		setHidden: func(ctx context.Context, actor model.User, postID string, hidden, deleted bool) (model.Post, error) {
			called = true
			return model.Post{}, nil
		},
	}
	code, env, _ := postsRun(t, Deps{Accounts: acc, Moderation: mod}, []string{"posts", "hide", "post-9"}, session)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.NotNil(t, env.Error)
	require.Equal(t, "forbidden", env.Error.Code)
	require.False(t, called)

	session, acc = postsSession(t, model.RoleModerator)
	var gotID string
	var gotHidden, gotDeleted bool
	mod = &postsModeration{
		setHidden: func(ctx context.Context, actor model.User, postID string, hidden, deleted bool) (model.Post, error) {
			gotID, gotHidden, gotDeleted = postID, hidden, deleted
			return model.Post{ID: postID, Hidden: hidden, Deleted: deleted}, nil
		},
	}
	code, env, _ = postsRun(t, Deps{Accounts: acc, Moderation: mod}, []string{"posts", "hide", "post-9"}, session)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "posts.hide", env.Command)
	require.Equal(t, "post-9", gotID)
	require.True(t, gotHidden)
	require.False(t, gotDeleted)
}

func TestPostsDeleteRequiresAdmin(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	called := false
	mod := &postsModeration{
		setHidden: func(ctx context.Context, actor model.User, postID string, hidden, deleted bool) (model.Post, error) {
			called = true
			return model.Post{}, nil
		},
	}
	code, env, _ := postsRun(t, Deps{Accounts: acc, Moderation: mod}, []string{"posts", "delete", "post-9"}, session)
	require.Equal(t, 1, code)
	require.False(t, env.OK)
	require.NotNil(t, env.Error)
	require.Equal(t, "forbidden", env.Error.Code)
	require.False(t, called)

	session, acc = postsSession(t, model.RoleAdmin)
	var gotID string
	var gotHidden, gotDeleted bool
	mod = &postsModeration{
		setHidden: func(ctx context.Context, actor model.User, postID string, hidden, deleted bool) (model.Post, error) {
			gotID, gotHidden, gotDeleted = postID, hidden, deleted
			return model.Post{ID: postID, Hidden: hidden, Deleted: deleted}, nil
		},
	}
	code, env, _ = postsRun(t, Deps{Accounts: acc, Moderation: mod}, []string{"posts", "delete", "post-9"}, session)
	require.Equal(t, 0, code)
	require.True(t, env.OK)
	require.Equal(t, "posts.delete", env.Command)
	require.Equal(t, "post-9", gotID)
	require.True(t, gotHidden)
	require.True(t, gotDeleted)
}

func postsRun(t *testing.T, deps Deps, args []string, session string) (int, Envelope, string) {
	t.Helper()
	var buf bytes.Buffer
	code := Execute(context.Background(), deps, args, &buf, Options{SessionPath: session})
	var env Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	return code, env, buf.String()
}

func postsData(t *testing.T, env Envelope) map[string]any {
	t.Helper()
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	return data
}

func postsSession(t *testing.T, role model.Role) (string, *postsAccounts) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session")
	require.NoError(t, SaveSession(path, "posts-token"))
	acc := &postsAccounts{
		session: func(ctx context.Context, token string) (model.User, error) {
			require.Equal(t, "posts-token", token)
			return model.User{ID: "user-1", Username: "ada", DisplayName: "Ada", Role: role}, nil
		},
	}
	return path, acc
}

type postsAccounts struct {
	Accounts
	session func(context.Context, string) (model.User, error)
}

func (f *postsAccounts) UserForSession(ctx context.Context, token string) (model.User, error) {
	if f.session == nil {
		panic("UserForSession")
	}
	return f.session(ctx, token)
}

type postsBoard struct {
	Board
	upload   func(context.Context, io.Reader) (string, string, error)
	getBySHA func(context.Context, string) (model.Post, error)
	create   func(context.Context, model.Post, string) (model.Post, error)
	setMeta  func(context.Context, *model.User, string, string, []string, string) (model.Post, error)
	get      func(context.Context, string) (model.Post, error)
	list     func(context.Context, int32, int32, bool) ([]model.Post, error)
	record   func(context.Context, model.User, model.Post)
}

func (f *postsBoard) Upload(ctx context.Context, r io.Reader) (string, string, error) {
	if f.upload == nil {
		panic("Upload")
	}
	return f.upload(ctx, r)
}

func (f *postsBoard) GetBySHA(ctx context.Context, sha string) (model.Post, error) {
	if f.getBySHA == nil {
		panic("GetBySHA")
	}
	return f.getBySHA(ctx, sha)
}

func (f *postsBoard) Create(ctx context.Context, p model.Post, sha string) (model.Post, error) {
	if f.create == nil {
		panic("Create")
	}
	return f.create(ctx, p, sha)
}

func (f *postsBoard) SetMetaAs(ctx context.Context, actor *model.User, id, artist string, tags []string, rating string) (model.Post, error) {
	if f.setMeta == nil {
		panic("SetMetaAs")
	}
	return f.setMeta(ctx, actor, id, artist, tags, rating)
}

func (f *postsBoard) Get(ctx context.Context, id string) (model.Post, error) {
	if f.get == nil {
		panic("Get")
	}
	return f.get(ctx, id)
}

func (f *postsBoard) List(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	if f.list == nil {
		panic("List")
	}
	return f.list(ctx, limit, offset, includeNSFW)
}

func (f *postsBoard) RecordCreation(ctx context.Context, actor model.User, post model.Post) {
	if f.record == nil {
		panic("RecordCreation")
	}
	f.record(ctx, actor, post)
}

type postsModeration struct {
	Moderation
	setHidden func(context.Context, model.User, string, bool, bool) (model.Post, error)
}

func (f *postsModeration) SetPostHidden(ctx context.Context, actor model.User, postID string, hidden bool, deleted bool) (model.Post, error) {
	if f.setHidden == nil {
		panic("SetPostHidden")
	}
	return f.setHidden(ctx, actor, postID, hidden, deleted)
}
