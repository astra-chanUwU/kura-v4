package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// Moderation composes accounts, posts and audit persistence. Role changes
// and post restores are multi-statement, so this repository owns their
// transactions (change + history row commit or neither does).
type Moderation struct {
	accounts *dao.AccountsDAO
	posts    *Posts
	audit    *dao.AuditDAO
}

func NewModeration(accountsDAO *dao.AccountsDAO, posts *Posts, audit *dao.AuditDAO) *Moderation {
	return &Moderation{accounts: accountsDAO, posts: posts, audit: audit}
}

func (r *Moderation) ListUsers(ctx context.Context) ([]model.User, error) {
	return r.accounts.ListUsers(ctx)
}
func (r *Moderation) GetUser(ctx context.Context, id string) (model.User, error) {
	return r.accounts.GetUserByID(ctx, id)
}
func (r *Moderation) GetPost(ctx context.Context, id string) (model.Post, error) {
	return r.posts.GetPost(ctx, id)
}
func (r *Moderation) CountSuperAdmins(ctx context.Context) (int64, error) {
	return r.accounts.CountUsersByRole(ctx, model.RoleSuperAdmin)
}

// ApplyUserRole sets a role and writes its history row atomically.
func (r *Moderation) ApplyUserRole(ctx context.Context, userID string, role model.Role, entry model.ChangeEntry) (model.User, error) {
	tx, err := r.accounts.DB().BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	user, err := r.accounts.WithTx(tx).UpdateUserRole(ctx, userID, role)
	if err != nil {
		return model.User{}, err
	}
	if _, err := r.audit.WithTx(tx).InsertChangeHistory(ctx, entry); err != nil {
		return model.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (r *Moderation) ApplyPostSnapshot(ctx context.Context, p model.Post, entry model.ChangeEntry) error {
	return r.posts.ApplyPostSnapshotAudited(ctx, p, entry)
}
func (r *Moderation) SetPostHidden(ctx context.Context, postID string, hidden bool, actorID string, entry model.ChangeEntry) error {
	return r.posts.SetPostHiddenAudited(ctx, postID, hidden, actorID, entry)
}
func (r *Moderation) GetHistory(ctx context.Context, id string) (model.ChangeEntry, error) {
	return r.audit.GetChangeHistory(ctx, id)
}
func (r *Moderation) ListHistoryForEntity(ctx context.Context, entityType, entityID string, limit int32) ([]model.ChangeEntry, error) {
	return r.audit.ListChangeHistoryForEntity(ctx, entityType, entityID, limit)
}
func (r *Moderation) ListRecentHistory(ctx context.Context, limit int32) ([]model.ChangeEntry, error) {
	return r.audit.ListRecentChangeHistory(ctx, limit)
}
func (r *Moderation) ListHistoryByActor(ctx context.Context, actorID string, limit int32) ([]model.ChangeEntry, error) {
	return r.audit.ListChangeHistoryByActor(ctx, actorID, limit)
}
