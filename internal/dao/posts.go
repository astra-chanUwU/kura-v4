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

func postFrom(id uuid.UUID, owner uuid.NullUUID, orig, key, ct string, size int64, created time.Time, ownerName, artist, source, sourceID, thumbKey, tagsCSV string, width, height int32, rating string) model.Post {
	p := model.Post{ID: id.String(), OriginalName: orig, StorageKey: key, ContentType: ct, SizeBytes: size, CreatedAt: created, OwnerUsername: ownerName,
		Artist: artist, Source: source, SourceID: sourceID, ThumbnailKey: thumbKey, Tags: splitTags(tagsCSV), Width: int(width), Height: int(height), Rating: rating}
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
	out := postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, "", r.Artist, r.Source, r.SourceID, r.ThumbnailKey, "", r.Width, r.Height, r.Rating)
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
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating), nil
}
func (d *PostsDAO) ListPosts(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	rows, err := d.q.ListPosts(ctx, sqlcgen.ListPostsParams{Limit: limit, Offset: offset, Column3: includeNSFW})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating))
	}
	return out, nil
}

// ListPostIndex returns the lightweight fields the client-side fuzzy
// search runs over (no storage keys or owner data).
func (d *PostsDAO) ListPostIndex(ctx context.Context, includeNSFW bool) ([]model.Post, error) {
	rows, err := d.q.ListPostIndex(ctx, includeNSFW)
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.Post{ID: r.ID.String(), Artist: r.Artist, Source: r.Source, SourceID: r.SourceID,
			ThumbnailKey: r.ThumbnailKey, ContentType: r.ContentType, Width: int(r.Width), Height: int(r.Height), CreatedAt: r.CreatedAt, Tags: splitTags(r.TagsCsv), Rating: r.Rating})
	}
	return out, nil
}
func (d *PostsDAO) GetPostBySHA(ctx context.Context, sha string) (model.Post, error) {
	r, err := d.q.GetPostBySHA(ctx, sha)
	if err != nil {
		return model.Post{}, err
	}
	return postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating), nil
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
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.ThumbnailKey, r.TagsCsv, r.Width, r.Height, r.Rating))
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
