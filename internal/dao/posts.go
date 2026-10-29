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

func postFrom(id uuid.UUID, owner uuid.NullUUID, orig, key, ct string, size int64, created time.Time, ownerName, artist, source, sourceID, thumbKey, tagsCSV string, width, height int32, rating string, hidden, deleted bool) model.Post {
	p := model.Post{ID: id.String(), OriginalName: orig, StorageKey: key, ContentType: ct, SizeBytes: size, CreatedAt: created, OwnerUsername: ownerName,
		Artist: artist, Source: source, SourceID: sourceID, ThumbnailKey: thumbKey, Tags: splitTags(tagsCSV), Width: int(width), Height: int(height), Rating: rating, Hidden: hidden, Deleted: deleted}
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
		Artist: p.Artist, Source: p.Source, SourceID: p.SourceID, ThumbnailKey: p.ThumbnailKey, Width: int32(p.Width), Height: int32(p.Height), Rating: p.Rating})
	if err != nil {
		return model.Post{}, err
	}
	out := postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, "", r.Artist, r.Source, r.SourceID, r.ThumbnailKey, "", r.Width, r.Height, r.Rating, r.Hidden, r.Deleted)
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
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted), nil
}
func (d *PostsDAO) ListPosts(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	rows, err := d.q.ListPosts(ctx, sqlcgen.ListPostsParams{Limit: limit, Offset: offset, Column3: includeNSFW})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted))
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
	rows, err := d.q.SearchPosts(ctx, sqlcgen.SearchPostsParams{
		Tags: q.Tags, Exclude: q.Exclude, Q: q.Q, QLike: q.QLike, ArtistLike: q.ArtistLike, IncludeNsfw: q.IncludeNSFW,
		Sort: q.Sort, RowOffset: q.Offset, RowLimit: q.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted))
	}
	return out, nil
}

func (d *PostsDAO) SuggestTags(ctx context.Context, q string, limit int32, includeNSFW bool) ([]model.TagCount, error) {
	rows, err := d.q.SuggestTags(ctx, sqlcgen.SuggestTagsParams{Q: q, IncludeNsfw: includeNSFW, RowLimit: limit})
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
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted), nil
}
func (d *PostsDAO) ListPostsByOwner(ctx context.Context, ownerID string) ([]model.Post, error) {
	uid, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListPostsByOwner(ctx, uuid.NullUUID{UUID: uid, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted))
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
func (d *PostsDAO) UpdatePostArtist(ctx context.Context, id, artist, rating string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return d.q.UpdatePostArtist(ctx, sqlcgen.UpdatePostArtistParams{ID: uid, Artist: artist, Rating: rating})
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

func (d *PostsDAO) ListHiddenPosts(ctx context.Context) ([]model.Post, error) {
	rows, err := d.q.ListHiddenPosts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted))
	}
	return out, nil
}
