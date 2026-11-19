package agentcli

import (
	"context"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const commentSafetyUsage = "usage: kura agent comment-safety <list|resolvers|dismiss|remove|reopen>"

func (e *Engine) commentSafety(ctx context.Context, args []string) int {
	if e.deps.CommentSafety == nil {
		return e.fail("comment-safety", "internal", "Comment safety is not available.")
	}
	if len(args) == 0 {
		return e.badUsage("comment-safety", commentSafetyUsage)
	}
	switch args[0] {
	case "list":
		return e.commentSafetyList(ctx, args[1:])
	case "resolvers":
		return e.commentSafetyResolvers(ctx, args[1:])
	case "dismiss":
		return e.commentSafetyDismiss(ctx, args[1:])
	case "remove":
		return e.commentSafetyRemove(ctx, args[1:])
	case "reopen":
		return e.commentSafetyReopen(ctx, args[1:])
	default:
		return e.badUsage("comment-safety", commentSafetyUsage)
	}
}

func (e *Engine) commentSafetyList(ctx context.Context, args []string) int {
	const cmd = "comment-safety.list"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only("limit", "offset", "status", "action", "category", "q", "resolver", "sort"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(cmd, catalogUsage(cmd))
	}
	limit, offset, ok := e.pageArgs(cmd, parsed)
	if !ok {
		return 2
	}
	status := parsed.Get("status")
	if status == "" {
		status = "open"
	}
	resolver := parsed.Get("resolver")
	if resolver == "me" {
		resolver = user.ID
	}
	page, err := e.deps.CommentSafety.List(ctx, user, model.CommentSafetyFilter{
		Status: status, Action: parsed.Get("action"), Category: parsed.Get("category"),
		Query: parsed.Get("q"), ResolverID: resolver, Sort: parsed.Get("sort"),
	}, limit, offset)
	if err != nil {
		return e.failErr(cmd, err)
	}
	out := make([]map[string]any, 0, len(page.Events))
	for _, event := range page.Events {
		out = append(out, commentSafetyView(event))
	}
	return e.ok(cmd, map[string]any{"total": page.Total, "events": out})
}

func (e *Engine) commentSafetyResolvers(ctx context.Context, args []string) int {
	const cmd = "comment-safety.resolvers"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(cmd, catalogUsage(cmd))
	}
	people, err := e.deps.CommentSafety.Resolvers(ctx, user)
	if err != nil {
		return e.failErr(cmd, err)
	}
	out := make([]map[string]any, 0, len(people))
	for _, person := range people {
		out = append(out, commentSafetyResolverView(person))
	}
	return e.ok(cmd, map[string]any{"resolvers": out})
}

func (e *Engine) commentSafetyDismiss(ctx context.Context, args []string) int {
	const cmd = "comment-safety.dismiss"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, "usage: kura agent comment-safety dismiss <event-id>")
	}
	event, err := e.deps.CommentSafety.Dismiss(ctx, user, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, commentSafetyView(event))
}

func (e *Engine) commentSafetyRemove(ctx context.Context, args []string) int {
	const cmd = "comment-safety.remove"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, "usage: kura agent comment-safety remove <event-id>")
	}
	event, err := e.deps.CommentSafety.Remove(ctx, user, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, commentSafetyView(event))
}

func (e *Engine) commentSafetyReopen(ctx context.Context, args []string) int {
	const cmd = "comment-safety.reopen"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, "usage: kura agent comment-safety reopen <event-id>")
	}
	event, err := e.deps.CommentSafety.Reopen(ctx, user, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, commentSafetyView(event))
}

func commentSafetyView(event model.CommentSafetyEvent) map[string]any {
	categories := event.Categories
	if categories == nil {
		categories = []string{}
	}
	return map[string]any{
		"id": event.ID, "comment_id": event.CommentID, "post_id": event.PostID,
		"author_username": event.AuthorUsername, "action": event.Action,
		"categories": categories, "body": event.Body, "status": event.Status,
		"decisions": commentSafetyDecisionViews(event.Decisions),
	}
}

func commentSafetyDecisionViews(steps []model.CommentSafetyDecision) []map[string]any {
	out := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		out = append(out, map[string]any{
			"actor_id": step.ActorID, "actor_username": step.ActorUsername,
			"action": step.Action, "created_at": step.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func commentSafetyResolverView(person model.CommentSafetyResolver) map[string]any {
	categories := person.LastCategories
	if categories == nil {
		categories = []string{}
	}
	lastAt := ""
	if !person.LastAt.IsZero() {
		lastAt = person.LastAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id": person.ID, "username": person.Username, "decisions": person.Decisions,
		"last_status": person.LastStatus, "last_action": person.LastAction,
		"last_categories": categories, "last_body": person.LastBody, "last_at": lastAt,
	}
}
