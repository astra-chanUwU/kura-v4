package dao

import (
	"context"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

func suggestionFrom(id uuid.UUID, kind, source, target, reason, status, author string, created time.Time) model.TagSuggestion {
	return model.TagSuggestion{
		ID: id.String(), Kind: kind, SourceTag: source, TargetTag: target,
		Reason: reason, Status: status, AuthorUsername: author, CreatedAt: created,
	}
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
	return suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.CreatedAt), nil
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
	return suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.CreatedAt), nil
}

func (d *TagPagesDAO) ListSuggestions(ctx context.Context, status string) ([]model.TagSuggestion, error) {
	rows, err := d.q.ListTagSuggestions(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]model.TagSuggestion, 0, len(rows))
	for _, row := range rows {
		out = append(out, suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.CreatedAt))
	}
	return out, nil
}

func (d *TagPagesDAO) ResolveSuggestion(ctx context.Context, id, resolverID, status string) (model.TagSuggestion, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	resolver, err := uuid.Parse(resolverID)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	row, err := d.q.ResolveTagSuggestion(ctx, sqlcgen.ResolveTagSuggestionParams{
		ID: uid, Status: status, ResolvedBy: uuid.NullUUID{UUID: resolver, Valid: true},
	})
	if err != nil {
		return model.TagSuggestion{}, err
	}
	return suggestionFrom(row.ID, row.Kind, row.SourceTag, row.TargetTag, row.Reason, row.Status, row.AuthorUsername, row.CreatedAt), nil
}
