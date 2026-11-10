package dao

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type PostsDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewPosts(conn *sql.DB) *PostsDAO { return &PostsDAO{db: conn, q: sqlcgen.New(conn)} }
func (d *PostsDAO) WithTx(tx *sql.Tx) *PostsDAO {
	return &PostsDAO{db: d.db, q: sqlcgen.New(tx)}
}
func (d *PostsDAO) DB() *sql.DB { return d.db }

func splitTags(csv string) []string {
	if csv == "" {
		return []string{}
	}
	return strings.Split(csv, ",")
}

func postFrom(id uuid.UUID, owner uuid.NullUUID, orig, key, ct string, size int64, created time.Time, ownerName, artist, source, sourceID, sourceURL, thumbKey, tagsCSV string, width, height int32, rating string, hidden, deleted bool, favoriteCount int32) model.Post {
	p := model.Post{ID: id.String(), OriginalName: orig, StorageKey: key, ContentType: ct, SizeBytes: size, CreatedAt: created, OwnerUsername: ownerName,
		Artist: artist, Source: source, SourceID: sourceID, SourceURL: sourceURL, ThumbnailKey: thumbKey, Tags: splitTags(tagsCSV), Width: int(width), Height: int(height), Rating: rating, Hidden: hidden, Deleted: deleted, FavoriteCount: int64(favoriteCount)}
	if owner.Valid {
		s := owner.UUID.String()
		p.OwnerID = &s
	}
	return p
}

func (d *PostsDAO) CreatePost(ctx context.Context, p model.Post, sha string) (model.Post, error) {
	var owner uuid.NullUUID
	if p.OwnerID != nil {
		id, err := uuid.Parse(*p.OwnerID)
		if err != nil {
			return model.Post{}, err
		}
		owner = uuid.NullUUID{UUID: id, Valid: true}
	}
	r, err := d.q.CreatePost(ctx, sqlcgen.CreatePostParams{OwnerID: owner, OriginalName: p.OriginalName, StorageKey: p.StorageKey, ContentType: p.ContentType, SizeBytes: p.SizeBytes, Sha256: sha,
		Artist: p.Artist, Source: p.Source, SourceID: p.SourceID, SourceUrl: p.SourceURL, ThumbnailKey: p.ThumbnailKey, Width: int32(p.Width), Height: int32(p.Height), Rating: p.Rating})
	if err != nil {
		return model.Post{}, err
	}
	out := postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, "", r.Artist, r.Source, r.SourceID, r.SourceUrl, r.ThumbnailKey, "", r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount)
	out.Tags = p.Tags
	return out, nil
}
func (d *PostsDAO) GetPost(ctx context.Context, id string) (model.Post, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.Post{}, err
	}
	r, err := d.q.GetPost(ctx, uid)
	if err != nil {
		return model.Post{}, err
	}
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.SourceUrl, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount), nil
}
func (d *PostsDAO) ListPosts(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	rows, err := d.q.ListPosts(ctx, sqlcgen.ListPostsParams{Limit: limit, Offset: offset, Column3: includeNSFW})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, "", r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount))
	}
	return out, nil
}

func matchParams(q model.PostQuery) sqlcgen.CountBoardMatchesParams {
	return sqlcgen.CountBoardMatchesParams{
		Tags: q.Tags, Exclude: q.Exclude, Q: q.Q, QLike: q.QLike, ArtistLike: q.ArtistLike, IncludeNsfw: q.IncludeNSFW,
	}
}

func (d *PostsDAO) CountBoardMatches(ctx context.Context, q model.PostQuery) (int64, error) {
	return d.q.CountBoardMatches(ctx, matchParams(q))
}

func (d *PostsDAO) SearchPosts(ctx context.Context, q model.PostQuery) ([]model.Post, error) {
	var afterID uuid.NullUUID
	var afterAt sql.NullTime
	if q.AfterID != "" {
		id, err := uuid.Parse(q.AfterID)
		if err != nil {
			return nil, err
		}
		afterID = uuid.NullUUID{UUID: id, Valid: true}
		afterAt = sql.NullTime{Time: q.AfterAt, Valid: true}
	}
	rows, err := d.q.SearchPosts(ctx, sqlcgen.SearchPostsParams{
		Tags: q.Tags, Exclude: q.Exclude, Q: q.Q, QLike: q.QLike, ArtistLike: q.ArtistLike, IncludeNsfw: q.IncludeNSFW,
		AfterID: afterID, Sort: q.Sort, AfterAt: afterAt, AfterFavorites: q.AfterFavorites, RowOffset: q.Offset, RowLimit: q.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, "", r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount))
	}
	return out, nil
}

func (d *PostsDAO) SuggestTags(ctx context.Context, prefix string, filter model.PostQuery, limit int32) ([]model.TagCount, error) {
	tags, exclude := filter.Tags, filter.Exclude
	if tags == nil {
		tags = []string{}
	}
	if exclude == nil {
		exclude = []string{}
	}
	rows, err := d.q.SuggestTags(ctx, sqlcgen.SuggestTagsParams{
		Tags: tags, Exclude: exclude, Match: filter.Q, MatchLike: filter.QLike, IncludeNsfw: filter.IncludeNSFW,
		Q: prefix, RowLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.TagCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.TagCount{Tag: r.Tag, Count: r.Count})
	}
	return out, nil
}

func (d *PostsDAO) PostNeighbors(ctx context.Context, id string, q model.PostQuery) (model.Neighbors, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.Neighbors{Index: -1}, nil
	}
	row, err := d.q.PostNeighbors(ctx, sqlcgen.PostNeighborsParams{
		PostID: uid, Sort: q.Sort, Tags: q.Tags, Exclude: q.Exclude, Q: q.Q, QLike: q.QLike, ArtistLike: q.ArtistLike, IncludeNsfw: q.IncludeNSFW,
	})
	if err != nil {
		return model.Neighbors{}, err
	}
	return model.Neighbors{Index: int(row.Idx), Total: row.Total, PrevID: sqlText(row.PrevID), NextID: sqlText(row.NextID)}, nil
}

func sqlText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}
func (d *PostsDAO) GetPostBySHA(ctx context.Context, sha string) (model.Post, error) {
	r, err := d.q.GetPostBySHA(ctx, sha)
	if err != nil {
		return model.Post{}, err
	}
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, "", r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount), nil
}
func (d *PostsDAO) CountVisiblePostsByOwner(ctx context.Context, ownerID string, includeNSFW bool) (int64, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return 0, err
	}
	return d.q.CountVisiblePostsByOwner(ctx, sqlcgen.CountVisiblePostsByOwnerParams{
		OwnerID: uuid.NullUUID{UUID: uid, Valid: true}, IncludeNsfw: includeNSFW,
	})
}

func (d *PostsDAO) ListPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool, limit, offset int32) ([]model.Post, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListPostsByOwner(ctx, sqlcgen.ListPostsByOwnerParams{
		OwnerID: uuid.NullUUID{UUID: uid, Valid: true}, IncludeNsfw: includeNSFW, RowLimit: limit, RowOffset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, "", r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount))
	}
	return out, nil
}
func (d *PostsDAO) CountPostsByOwner(ctx context.Context, ownerID string) (int64, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return 0, err
	}
	return d.q.CountPostsByOwner(ctx, uuid.NullUUID{UUID: uid, Valid: true})
}

// CountPostsByOwnerSince counts every post this owner created inside within,
// including hidden and soft-deleted posts.
func (d *PostsDAO) CountPostsByOwnerSince(ctx context.Context, ownerID string, within time.Duration) (int64, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return 0, err
	}
	return d.q.CountPostsByOwnerSince(ctx, sqlcgen.CountPostsByOwnerSinceParams{
		OwnerID: uuid.NullUUID{UUID: uid, Valid: true}, WithinSeconds: within.Seconds(),
	})
}
func (d *PostsDAO) ExistsBySHA(ctx context.Context, sha string) (bool, error) {
	return d.q.PostExistsBySHA(ctx, sha)
}
func (d *PostsDAO) SetPostDimensions(ctx context.Context, id string, width, height int) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return d.q.SetPostDimensions(ctx, sqlcgen.SetPostDimensionsParams{ID: uid, Width: int32(width), Height: int32(height)})
}
func (d *PostsDAO) SetPostThumbnail(ctx context.Context, id, key string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return d.q.SetPostThumbnail(ctx, sqlcgen.SetPostThumbnailParams{ID: uid, ThumbnailKey: key})
}
func (d *PostsDAO) UpdatePostMeta(ctx context.Context, id, artist, rating, sourceURL string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return d.q.UpdatePostMeta(ctx, sqlcgen.UpdatePostMetaParams{ID: uid, Artist: artist, Rating: rating, SourceUrl: sourceURL})
}
func (d *PostsDAO) DeletePostTags(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return d.q.DeletePostTags(ctx, uid)
}
func (d *PostsDAO) InsertPostTag(ctx context.Context, id, tag string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return d.q.InsertPostTag(ctx, sqlcgen.InsertPostTagParams{PostID: uid, Tag: tag})
}

func (d *PostsDAO) SetPostHidden(ctx context.Context, id string, hidden, deleted bool, actorID string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	var actor uuid.UUID
	if actorID != "" {
		if parsed, err := uuid.Parse(actorID); err == nil {
			actor = parsed
		}
	}
	return d.q.SetPostHidden(ctx, sqlcgen.SetPostHiddenParams{ID: uid, Column2: hidden, Column3: deleted, Column4: actor})
}

func (d *PostsDAO) CountRemovedPosts(ctx context.Context) (int64, int64, error) {
	row, err := d.q.CountRemovedPosts(ctx)
	if err != nil {
		return 0, 0, err
	}
	return row.Hidden, row.Deleted, nil
}

func (d *PostsDAO) RelatedPosts(ctx context.Context, id string, includeNSFW bool, exclude []string, limit int32) ([]model.Post, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	if exclude == nil {
		exclude = []string{}
	}
	rows, err := d.q.RelatedPosts(ctx, sqlcgen.RelatedPostsParams{PostID: uid, IncludeNsfw: includeNSFW, Exclude: exclude, RowLimit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, "", r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount))
	}
	return out, nil
}

func (d *PostsDAO) RandomPost(ctx context.Context, q model.PostQuery) (model.Post, error) {
	r, err := d.q.RandomPost(ctx, sqlcgen.RandomPostParams{
		Tags: q.Tags, Exclude: q.Exclude, Q: q.Q, QLike: q.QLike, ArtistLike: q.ArtistLike, IncludeNsfw: q.IncludeNSFW,
	})
	if err != nil {
		return model.Post{}, err
	}
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.SourceUrl, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount), nil
}

func (d *PostsDAO) TagAliasTargets(ctx context.Context, tags []string) (map[string]string, error) {
	if len(tags) == 0 {
		return map[string]string{}, nil
	}
	rows, err := d.q.ListTagAliasTargets(ctx, tags)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Alias] = row.Target
	}
	return out, nil
}

func (d *PostsDAO) ListHiddenPosts(ctx context.Context, kind string, limit, offset int32) ([]model.Post, error) {
	rows, err := d.q.ListHiddenPosts(ctx, sqlcgen.ListHiddenPostsParams{Kind: kind, RowLimit: limit, RowOffset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, "", r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount))
	}
	return out, nil
}
