package repository

import (
	"context"
	"database/sql"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Comments struct {
	dao *dao.CommentsDAO
}

func NewComments(d *dao.CommentsDAO) *Comments { return &Comments{dao: d} }

func (r *Comments) DB() *sql.DB { return r.dao.DB() }

func (r *Comments) WithTx(tx *sql.Tx) *dao.CommentsDAO { return r.dao.WithTx(tx) }

func (r *Comments) CommentPostAccess(ctx context.Context, postID string) (model.CommentPostAccess, error) {
	access, err := r.dao.CommentPostAccess(ctx, postID)
	return access, NotFound(err)
}

func (r *Comments) ListCommentsByPost(ctx context.Context, postID string, limit, offset int32) ([]model.Comment, error) {
	return r.dao.ListCommentsByPost(ctx, postID, limit, offset)
}

func (r *Comments) CountCommentsByPost(ctx context.Context, postID string) (int64, error) {
	return r.dao.CountCommentsByPost(ctx, postID)
}

func (r *Comments) CreateComment(ctx context.Context, postID, authorID, body string) (model.Comment, error) {
	comment, err := r.dao.CreateComment(ctx, postID, authorID, body)
	return comment, NotFound(err)
}

func (r *Comments) GetComment(ctx context.Context, id string) (model.Comment, error) {
	comment, err := r.dao.GetComment(ctx, id)
	return comment, NotFound(err)
}

func (r *Comments) UpdateCommentBody(ctx context.Context, id, body string) (model.Comment, error) {
	comment, err := r.dao.UpdateCommentBody(ctx, id, body)
	return comment, NotFound(err)
}

func (r *Comments) DeleteComment(ctx context.Context, id string) (int64, error) {
	return r.dao.DeleteComment(ctx, id)
}
