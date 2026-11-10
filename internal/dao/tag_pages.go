package dao

import (
	"context"
	"database/sql"
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type TagPagesDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewTagPages(conn *sql.DB) *TagPagesDAO {
	return &TagPagesDAO{db: conn, q: sqlcgen.New(conn)}
}

func (d *TagPagesDAO) WithTx(tx *sql.Tx) *TagPagesDAO {
	return &TagPagesDAO{db: d.db, q: sqlcgen.New(tx)}
}

func (d *TagPagesDAO) DB() *sql.DB { return d.db }

func (d *TagPagesDAO) GetPage(ctx context.Context, tag string) (model.TagWiki, bool, error) {
	row, err := d.q.GetTagPage(ctx, tag)
	if errors.Is(err, sql.ErrNoRows) {
		return model.TagWiki{}, false, nil
	}
	if err != nil {
		return model.TagWiki{}, false, err
	}
	return model.TagWiki{Tag: row.Tag, Category: row.Category, Description: row.Description, UpdatedByUsername: row.UpdatedByUsername, UpdatedAt: row.UpdatedAt}, true, nil
}

func (d *TagPagesDAO) UpsertPage(ctx context.Context, tag, category, description, userID string) error {
	var updatedBy uuid.NullUUID
	if userID != "" {
		id, err := uuid.Parse(userID)
		if err != nil {
			return err
		}
		updatedBy = uuid.NullUUID{UUID: id, Valid: true}
	}
	return d.q.UpsertTagPage(ctx, sqlcgen.UpsertTagPageParams{Tag: tag, Category: category, Description: description, UpdatedBy: updatedBy})
}

func (d *TagPagesDAO) CountPosts(ctx context.Context, tag string, includeNSFW bool, exclude []string) (int64, error) {
	if exclude == nil {
		exclude = []string{}
	}
	return d.q.CountVisiblePostsForTag(ctx, sqlcgen.CountVisiblePostsForTagParams{Tag: tag, IncludeNsfw: includeNSFW, Exclude: exclude})
}

func (d *TagPagesDAO) RecentPosts(ctx context.Context, tag string, includeNSFW bool, exclude []string) ([]model.Post, error) {
	if exclude == nil {
		exclude = []string{}
	}
	rows, err := d.q.ListRecentPostsForTag(ctx, sqlcgen.ListRecentPostsForTagParams{Tag: tag, IncludeNsfw: includeNSFW, Exclude: exclude})
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, r.Artist, r.Source, r.SourceID, r.SourceUrl, r.ThumbnailKey, r.SampleKey, r.TagsCsv, r.Width, r.Height, r.Rating, r.Hidden, r.Deleted, r.FavoriteCount))
	}
	return out, nil
}

func (d *TagPagesDAO) EnsureArtistCategory(ctx context.Context, tag string) error {
	return d.q.EnsureArtistCategory(ctx, tag)
}

func (d *TagPagesDAO) AliasTarget(ctx context.Context, alias string) (string, bool, error) {
	target, err := d.q.GetTagAlias(ctx, alias)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return target, true, nil
}

func (d *TagPagesDAO) AliasesOf(ctx context.Context, target string) ([]string, error) {
	rows, err := d.q.ListAliasesOfTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []string{}
	}
	return rows, nil
}

func (d *TagPagesDAO) IsAliasTarget(ctx context.Context, tag string) (bool, error) {
	return d.q.TagIsAliasTarget(ctx, tag)
}

func (d *TagPagesDAO) InsertAlias(ctx context.Context, alias, target, userID string) error {
	var createdBy uuid.NullUUID
	if userID != "" {
		id, err := uuid.Parse(userID)
		if err != nil {
			return err
		}
		createdBy = uuid.NullUUID{UUID: id, Valid: true}
	}
	return d.q.InsertTagAlias(ctx, sqlcgen.InsertTagAliasParams{Alias: alias, Target: target, CreatedBy: createdBy})
}

func (d *TagPagesDAO) DeleteAlias(ctx context.Context, alias, target string) (int64, error) {
	return d.q.DeleteTagAlias(ctx, sqlcgen.DeleteTagAliasParams{Alias: alias, Target: target})
}

func (d *TagPagesDAO) MovePostTag(ctx context.Context, alias, target string) error {
	if err := d.q.DeleteConflictingAliasTags(ctx, sqlcgen.DeleteConflictingAliasTagsParams{Alias: alias, Target: target}); err != nil {
		return err
	}
	return d.q.RenamePostTag(ctx, sqlcgen.RenamePostTagParams{Alias: alias, Target: target})
}
