package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

func suggestionFrom(id uuid.UUID, kind, source, target, reason, status, author, resolver string, created time.Time, resolved sql.NullTime) model.TagSuggestion {
	item := model.TagSuggestion{
		ID: id.String(), Kind: kind, SourceTag: source, TargetTag: target,
		Reason: reason, Status: status, AuthorUsername: author, ResolverUsername: resolver,
		CreatedAt: created,
	}
	if resolved.Valid {
		item.ResolvedAt = resolved.Time
	}
	return item
}

func (d *TagPagesDAO) CreateSuggestion(ctx context.Context, kind, sourceTag, targetTag, reason, authorID string) (model.TagSuggestion, error) {
	author, err := uuid.Parse(authorID)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	row, err := d.q.CreateTagSuggestion(ctx, sqlcgen.CreateTagSuggestionParams{
		Kind: kind, SourceTag: sourceTag, TargetTag: targetTag, Reason: reason, AuthorID: author,
	})
	if err != nil {
		return model.TagSuggestion{}, err
	}
	return suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.ResolverUsername, row.CreatedAt, row.ResolvedAt), nil
}

func (d *TagPagesDAO) GetSuggestion(ctx context.Context, id string) (model.TagSuggestion, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	row, err := d.q.GetTagSuggestion(ctx, uid)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	return suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.ResolverUsername, row.CreatedAt, row.ResolvedAt), nil
}

func (d *TagPagesDAO) ListSuggestions(ctx context.Context, status string) ([]model.TagSuggestion, error) {
	rows, err := d.q.ListTagSuggestions(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]model.TagSuggestion, 0, len(rows))
	for _, row := range rows {
		out = append(out, suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.ResolverUsername, row.CreatedAt, row.ResolvedAt))
	}
	return out, nil
}

func (d *TagPagesDAO) ResolveSuggestion(ctx context.Context, id, resolverID, status, fromStatus string) (model.TagSuggestion, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	resolver, err := uuid.Parse(resolverID)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	row, err := d.q.ResolveTagSuggestion(ctx, sqlcgen.ResolveTagSuggestionParams{
		ID: uid, Status: status, FromStatus: fromStatus, ResolvedBy: uuid.NullUUID{UUID: resolver, Valid: true},
	})
	if err != nil {
		return model.TagSuggestion{}, err
	}
	return suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.ResolverUsername, row.CreatedAt, row.ResolvedAt), nil
}
