package board

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestSourceURLRejectedUnlessHTTP(t *testing.T) {
	repo := &fakeRepo{post: model.Post{ID: "p1", Artist: "ada", Rating: "sfw", Tags: []string{"cat"}}}
	svc := NewService(repo, nil)
	for _, raw := range []string{"ftp://example.com/a", "javascript:alert(1)", "https://", "/relative", "notaurl", "http:///missing-host"} {
		_, err := svc.SetMeta(context.Background(), "p1", "ada", []string{"cat"}, "sfw", raw)
		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		require.Equal(t, sourceURLMessage, vErr.Message)
	}
	require.Empty(t, repo.meta)
	require.Empty(t, repo.audited)

	long := "https://example.com/" + strings.Repeat("a", 300)
	_, err := svc.Create(context.Background(), model.Post{SourceURL: long}, "sha")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Equal(t, sourceURLMessage, vErr.Message)
	require.Empty(t, repo.created)
}

func TestEmptySourceURLAllowed(t *testing.T) {
	repo := &fakeRepo{post: model.Post{ID: "p1", Artist: "ada", Rating: "sfw", Tags: []string{"cat"}, SourceURL: "https://example.com/old"}}
	svc := NewService(repo, nil)
	actor := model.User{ID: "u1", Username: "ada"}
	got, err := svc.SetMetaAs(context.Background(), &actor, "p1", "ada", []string{"cat"}, "sfw", "   ")
	require.NoError(t, err)
	require.Equal(t, "", got.SourceURL)
	require.Len(t, repo.audited, 1)

	created, err := svc.Create(context.Background(), model.Post{SourceURL: "  "}, "sha")
	require.NoError(t, err)
	require.Equal(t, "", created.SourceURL)
	require.Equal(t, "", repo.created[0].SourceURL)
}

func TestSetMetaHistoryIncludesSourceURL(t *testing.T) {
	repo := &fakeRepo{post: model.Post{ID: "p1", Artist: "ada", Rating: "sfw", Tags: []string{"cat"}, SourceURL: ""}}
	svc := NewService(repo, nil)
	actor := model.User{ID: "u1", Username: "ada"}
	got, err := svc.SetMetaAs(context.Background(), &actor, "p1", "ada", []string{"cat"}, "sfw", "  https://example.com/a  ")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/a", got.SourceURL)
	require.Len(t, repo.audited, 1)
	before, err := repo.audited[0].PostBefore()
	require.NoError(t, err)
	after, err := repo.audited[0].PostAfter()
	require.NoError(t, err)
	require.True(t, before.HasSourceURL)
	require.Equal(t, "", before.SourceURL)
	require.True(t, after.HasSourceURL)
	require.Equal(t, "https://example.com/a", after.SourceURL)
	require.Contains(t, string(repo.audited[0].Before), `"source_url":""`)
	require.Contains(t, string(repo.audited[0].After), `"source_url":"https://example.com/a"`)

	_, err = svc.SetMetaAs(context.Background(), &actor, "p1", "ada", []string{"cat"}, "sfw", "https://example.com/a")
	require.NoError(t, err)
	require.Len(t, repo.audited, 1)

	old, err := (model.ChangeEntry{Before: json.RawMessage(`{"artist":"ada","tags":["cat"],"rating":"sfw","hidden":false,"deleted":false}`)}).PostBefore()
	require.NoError(t, err)
	require.False(t, old.HasSourceURL)
	require.Equal(t, "", old.SourceURL)
	kept, err := (model.ChangeEntry{Before: json.RawMessage(`{"artist":"ada","tags":["cat"],"rating":"sfw","hidden":false,"deleted":false,"source_url":""}`)}).PostBefore()
	require.NoError(t, err)
	require.True(t, kept.HasSourceURL)
}

// ownedPost is one stored upload. Hidden and deleted rows still count:
// the quota query does not look at hidden_at or deleted_at.
type ownedPost struct {
	ownerID   string
	createdAt time.Time
	hidden    bool
	deleted   bool
}

type fakeRepo struct {
	post           model.Post
	meta           []model.Post
	audited        []model.ChangeEntry
	created        []model.Post
	artistPages    []string
	relatedLimit   int32
	relatedNSFW    bool
	relatedExclude []string
	related        []model.Post
	randomErr      error
	random         model.Post
	aliases        map[string]string
	lastQuery      model.PostQuery
	owned          []ownedPost
}

func (f *fakeRepo) CreatePost(_ context.Context, p model.Post, _ string) (model.Post, error) {
	f.created = append(f.created, p)
	if p.ID == "" {
		p.ID = "created"
	}
	return p, nil
}

func (f *fakeRepo) GetPost(_ context.Context, id string) (model.Post, error) {
	if f.post.ID != id {
		return model.Post{}, sql.ErrNoRows
	}
	return f.post, nil
}

func (f *fakeRepo) ListPosts(context.Context, int32, int32, bool) ([]model.Post, error) {
	return nil, nil
}
func (f *fakeRepo) ExistsBySHA(context.Context, string) (bool, error) { return false, nil }
func (f *fakeRepo) SetPostThumbnail(context.Context, string, string) error {
	return nil
}
func (f *fakeRepo) SetPostSample(context.Context, string, string) error       { return nil }
func (f *fakeRepo) SetPostDimensions(context.Context, string, int, int) error { return nil }
func (f *fakeRepo) ReplacePostMeta(_ context.Context, p model.Post) error {
	f.meta = append(f.meta, p)
	f.post = p
	return nil
}
func (f *fakeRepo) ReplacePostMetaAudited(_ context.Context, p model.Post, entry model.ChangeEntry) error {
	f.audited = append(f.audited, entry)
	f.post = p
	return nil
}
func (f *fakeRepo) InsertHistory(context.Context, model.ChangeEntry) (model.ChangeEntry, error) {
	return model.ChangeEntry{}, nil
}
func (f *fakeRepo) SearchPosts(_ context.Context, q model.PostQuery) ([]model.Post, error) {
	f.lastQuery = q
	return []model.Post{}, nil
}
func (f *fakeRepo) CountBoardMatches(context.Context, model.PostQuery) (int64, error) {
	return 0, nil
}
func (f *fakeRepo) SuggestTags(context.Context, string, model.PostQuery, int32) ([]model.TagCount, error) {
	return nil, nil
}
func (f *fakeRepo) PostNeighbors(context.Context, string, model.PostQuery) (model.Neighbors, error) {
	return model.Neighbors{}, nil
}
func (f *fakeRepo) GetPostBySHA(context.Context, string) (model.Post, error) {
	return model.Post{}, sql.ErrNoRows
}
func (f *fakeRepo) CountPostsByOwner(context.Context, string) (int64, error) { return 0, nil }

// CountPostsByOwnerSince counts every stored post inside the window.
// Hidden and deleted rows count. createdAt must be strictly after
// now-within, matching created_at > now() - interval.
func (f *fakeRepo) CountPostsByOwnerSince(_ context.Context, ownerID string, within time.Duration) (int64, error) {
	cutoff := time.Now().Add(-within)
	var n int64
	for _, post := range f.owned {
		if post.ownerID == ownerID && post.createdAt.After(cutoff) {
			n++
		}
	}
	return n, nil
}
func (f *fakeRepo) CountVisiblePostsByOwner(context.Context, string, bool) (int64, error) {
	return 0, nil
}
func (f *fakeRepo) ListPostsByOwner(context.Context, string, bool, int32, int32) ([]model.Post, error) {
	return nil, nil
}
func (f *fakeRepo) RelatedPosts(_ context.Context, _ string, includeNSFW bool, exclude []string, limit int32) ([]model.Post, error) {
	f.relatedLimit = limit
	f.relatedNSFW = includeNSFW
	if exclude == nil {
		f.relatedExclude = []string{}
	} else {
		f.relatedExclude = append([]string{}, exclude...)
	}
	if f.related == nil {
		return []model.Post{}, nil
	}
	return f.related, nil
}
func (f *fakeRepo) EnsureArtistPage(_ context.Context, tag string) error {
	f.artistPages = append(f.artistPages, tag)
	return nil
}
func (f *fakeRepo) ListImplications(context.Context) ([]model.TagImplication, error) {
	return nil, nil
}
func (f *fakeRepo) CountPublicPostsByOwner(context.Context, string, bool) (int64, error) {
	return 0, nil
}
func (f *fakeRepo) ListPublicPostsByOwner(context.Context, string, bool, int32, int32) ([]model.Post, error) {
	return nil, nil
}
func (f *fakeRepo) CountMetaEditsByActor(context.Context, string) (int64, error) { return 0, nil }
func (f *fakeRepo) LastMetaEditor(context.Context, string) (string, error) {
	return "", sql.ErrNoRows
}
func (f *fakeRepo) TagAliasTargets(_ context.Context, tags []string) (map[string]string, error) {
	out := map[string]string{}
	for _, tag := range tags {
		if target, ok := f.aliases[tag]; ok {
			out[tag] = target
		}
	}
	return out, nil
}
func (f *fakeRepo) RandomPost(context.Context, model.PostQuery) (model.Post, error) {
	if f.randomErr != nil {
		return model.Post{}, f.randomErr
	}
	if f.random.ID == "" {
		return model.Post{}, sql.ErrNoRows
	}
	return f.random, nil
}

func TestSetMetaStoresArtistTag(t *testing.T) {
	repo := &fakeRepo{post: model.Post{ID: "p1", Rating: "nsfw", Tags: []string{"cat"}}}
	svc := NewService(repo, nil)
	got, err := svc.SetMeta(context.Background(), "p1", "Ada Lovelace", []string{"cat"}, "", "")
	require.NoError(t, err)
	require.Equal(t, "Ada Lovelace", got.Artist)
	require.Equal(t, []string{"ada_lovelace", "cat"}, got.Tags)
	require.Equal(t, "nsfw", got.Rating)
	require.Equal(t, []string{"ada_lovelace"}, repo.artistPages)

	got, err = svc.SetMeta(context.Background(), "p1", "Bea", []string{"cat"}, "", "")
	require.NoError(t, err)
	require.Equal(t, "Bea", got.Artist)
	require.Equal(t, []string{"bea", "cat"}, got.Tags)
	require.Equal(t, "nsfw", got.Rating)
	require.Equal(t, []string{"ada_lovelace", "bea"}, repo.artistPages)
}

func TestSetMetaArtistAliasStoresCanonicalTag(t *testing.T) {
	repo := &fakeRepo{
		aliases: map[string]string{"ada_lovelace": "ada"},
		post:    model.Post{ID: "p1", Artist: "Ada Lovelace", Rating: "sfw", Tags: []string{"ada", "cat"}},
	}
	svc := NewService(repo, nil)
	got, err := svc.SetMeta(context.Background(), "p1", "Ada Lovelace", []string{"cat"}, "sfw", "")
	require.NoError(t, err)
	require.Equal(t, "Ada Lovelace", got.Artist)
	require.Equal(t, []string{"ada", "cat"}, got.Tags)
	require.Equal(t, []string{"ada"}, repo.artistPages)

	got, err = svc.SetMeta(context.Background(), "p1", "", []string{"cat", "ada"}, "sfw", "")
	require.NoError(t, err)
	require.Equal(t, "", got.Artist)
	require.Equal(t, []string{"cat"}, got.Tags)
}
