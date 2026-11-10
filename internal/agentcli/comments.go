package agentcli

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const commentsUsage = "usage: kura agent comments <list|create|update|delete>"

func (e *Engine) comments(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("comments", commentsUsage)
	}
	switch args[0] {
	case "list":
		return e.commentsList(ctx, args[1:])
	case "create":
		return e.commentsCreate(ctx, args[1:])
	case "update":
		return e.commentsUpdate(ctx, args[1:])
	case "delete":
		return e.commentsDelete(ctx, args[1:])
	default:
		return e.badUsage("comments", commentsUsage)
	}
}

func (e *Engine) commentsList(ctx context.Context, args []string) int {
	const cmd = "comments.list"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only("limit", "offset"); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, catalogUsage(cmd))
	}
	limit, offset, ok := e.pageArgs(cmd, parsed)
	if !ok {
		return 2
	}
	viewer, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	if e.deps.Comments == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	page, err := e.deps.Comments.List(ctx, viewer, parsed.Pos[0], limit, offset)
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentCommentPage(page, viewer))
}

func (e *Engine) commentsCreate(ctx context.Context, args []string) int {
	const cmd = "comments.create"
	parsed, code, ok := e.commentBodyArgs(args, cmd)
	if !ok {
		return code
	}
	user, ok := e.requireMember(ctx, cmd)
	if !ok {
		return 1
	}
	if e.deps.Comments == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	comment, err := e.deps.Comments.Create(ctx, user, parsed.Pos[0], parsed.Get("body"))
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentComment(comment, &user))
}

func (e *Engine) commentsUpdate(ctx context.Context, args []string) int {
	const cmd = "comments.update"
	parsed, code, ok := e.commentBodyArgs(args, cmd)
	if !ok {
		return code
	}
	user, ok := e.requireMember(ctx, cmd)
	if !ok {
		return 1
	}
	if e.deps.Comments == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	comment, err := e.deps.Comments.Update(ctx, user, parsed.Pos[0], parsed.Get("body"))
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentComment(comment, &user))
}

func (e *Engine) commentsDelete(ctx context.Context, args []string) int {
	const cmd = "comments.delete"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, catalogUsage(cmd))
	}
	user, ok := e.requireMember(ctx, cmd)
	if !ok {
		return 1
	}
	if e.deps.Comments == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	id := parsed.Pos[0]
	if err := e.deps.Comments.Delete(ctx, user, id); err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, map[string]any{"comment_id": id})
}

func (e *Engine) requireMember(ctx context.Context, command string) (model.User, bool) {
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return model.User{}, false
	}
	if !e.requireMin(command, user, "member") {
		return model.User{}, false
	}
	return user, true
}

func (e *Engine) commentBodyArgs(args []string, command string) (Args, int, bool) {
	parsed, err := parseArgs(args)
	if err != nil {
		return Args{}, e.badUsage(command, err.Error()), false
	}
	if err := parsed.Only("body"); err != nil || len(parsed.Pos) != 1 || !parsed.Has("body") {
		return Args{}, e.badUsage(command, catalogUsage(command)), false
	}
	return parsed, 0, true
}

func presentCommentPage(page model.CommentPage, viewer *model.User) dto.CommentPage {
	return dto.CommentPage{
		Total:    page.Total,
		Comments: presentComments(page.Comments, viewer),
	}
}

func presentComments(comments []model.Comment, viewer *model.User) []dto.Comment {
	out := make([]dto.Comment, 0, len(comments))
	for _, comment := range comments {
		out = append(out, presentComment(comment, viewer))
	}
	return out
}

func presentComment(comment model.Comment, viewer *model.User) dto.Comment {
	canEdit := false
	canDelete := false
	if viewer != nil && viewer.ID != "" {
		if viewer.ID == comment.AuthorID {
			canEdit = true
			canDelete = true
		}
		if viewer.Role.AtLeast(model.RoleModerator) {
			canDelete = true
		}
	}
	return dto.Comment{
		ID:              comment.ID,
		PostID:          comment.PostID,
		AuthorUsername:  comment.AuthorUsername,
		AuthorAvatarURL: commentAvatarURL(comment.AuthorID, comment.AuthorAvatarKey),
		Body:            comment.Body,
		CreatedAt:       comment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       comment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		CanEdit:         canEdit,
		CanDelete:       canDelete,
	}
}

func commentAvatarURL(authorID, key string) string {
	if key == "" {
		return ""
	}
	v := key
	if len(v) > 12 {
		v = v[:12]
	}
	return "/media/avatars/" + authorID + "?v=" + v
}
