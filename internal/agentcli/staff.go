package agentcli

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

func (e *Engine) history(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("history", "usage: kura agent history <post|recent|actor|rollback>")
	}
	switch args[0] {
	case "post":
		return e.historyPost(ctx, args[1:])
	case "recent":
		return e.historyRecent(ctx, args[1:])
	case "actor":
		return e.historyActor(ctx, args[1:])
	case "rollback":
		return e.historyRollback(ctx, args[1:])
	default:
		return e.badUsage("history", "usage: kura agent history <post|recent|actor|rollback>")
	}
}

func (e *Engine) historyPost(ctx context.Context, args []string) int {
	const command = "history.post"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("limit"); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(command, catalogUsage(command))
	}
	limit, err := parsed.Int("limit", 100, 1, 200)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if _, ok := e.requireActor(ctx, command); !ok {
		return 1
	}
	entries, err := e.deps.Moderation.HistoryForPost(ctx, parsed.Pos[0], int32(limit))
	if err != nil {
		return e.failErr(command, err)
	}
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return e.ok(command, presentHistories(entries))
}

func (e *Engine) historyRecent(ctx context.Context, args []string) int {
	const command = "history.recent"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("limit"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	limit, err := parsed.Int("limit", 50, 1, 200)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "moderator") {
		return 1
	}
	entries, err := e.deps.Moderation.RecentHistory(ctx, user, int32(limit))
	if err != nil {
		return e.failErr(command, err)
	}
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return e.ok(command, presentHistories(entries))
}

func (e *Engine) historyActor(ctx context.Context, args []string) int {
	const command = "history.actor"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("limit"); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(command, catalogUsage(command))
	}
	limit, err := parsed.Int("limit", 50, 1, 200)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "moderator") {
		return 1
	}
	if e.deps.Accounts == nil {
		return e.fail(command, "internal", "Something went wrong. Please try again.")
	}
	target, err := e.deps.Accounts.GetByUsername(ctx, parsed.Pos[0])
	if err != nil {
		return e.failErr(command, err)
	}
	entries, err := e.deps.Moderation.HistoryByActor(ctx, user, target.ID, int32(limit))
	if err != nil {
		return e.failErr(command, err)
	}
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return e.ok(command, presentHistories(entries))
}

func (e *Engine) historyRollback(ctx context.Context, args []string) int {
	const command = "history.rollback"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(command, catalogUsage(command))
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "moderator") {
		return 1
	}
	entry, err := e.deps.Moderation.Rollback(ctx, user, parsed.Pos[0])
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentHistory(entry))
}

func (e *Engine) users(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("users", "usage: kura agent users <list|role>")
	}
	switch args[0] {
	case "list":
		return e.usersList(ctx, args[1:])
	case "role":
		return e.usersRole(ctx, args[1:])
	default:
		return e.badUsage("users", "usage: kura agent users <list|role>")
	}
}

func (e *Engine) usersList(ctx context.Context, args []string) int {
	const command = "users.list"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "admin") {
		return 1
	}
	users, err := e.deps.Moderation.ListUsers(ctx, user)
	if err != nil {
		return e.failErr(command, err)
	}
	if users == nil {
		users = []model.User{}
	}
	return e.ok(command, presentUsers(users))
}

func (e *Engine) usersRole(ctx context.Context, args []string) int {
	const command = "users.role"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 2 {
		return e.badUsage(command, catalogUsage(command))
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "admin") {
		return 1
	}
	roleName := parsed.Pos[1]
	if !model.Role(roleName).Valid() {
		return e.badUsage(command, "role must be member, moderator, admin or super_admin")
	}
	if e.deps.Accounts == nil {
		return e.fail(command, "internal", "Something went wrong. Please try again.")
	}
	target, err := e.deps.Accounts.GetByUsername(ctx, parsed.Pos[0])
	if err != nil {
		return e.failErr(command, err)
	}
	updated, err := e.deps.Moderation.SetRole(ctx, user, target.ID, model.Role(roleName))
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentUser(updated))
}
