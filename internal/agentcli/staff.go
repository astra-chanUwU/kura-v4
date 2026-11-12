package agentcli

import (
	"context"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

func (e *Engine) history(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("history", "usage: kura agent history <post|recent|actor|actors|rollback>")
	}
	switch args[0] {
	case "post":
		return e.historyPost(ctx, args[1:])
	case "recent":
		return e.historyRecent(ctx, args[1:])
	case "actor":
		return e.historyActor(ctx, args[1:])
	case "actors":
		return e.historyActors(ctx, args[1:])
	case "rollback":
		return e.historyRollback(ctx, args[1:])
	default:
		return e.badUsage("history", "usage: kura agent history <post|recent|actor|actors|rollback>")
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
	if err := parsed.Only("q", "actor", "action", "entity", "sort", "limit", "offset"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	if !historyRecentFiltered(parsed) {
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
	limit, offset, ok := e.pageArgs(command, parsed)
	if !ok {
		return 2
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "moderator") {
		return 1
	}
	actorID := ""
	if parsed.Has("actor") {
		if e.deps.Accounts == nil {
			return e.fail(command, "internal", "Something went wrong. Please try again.")
		}
		target, err := e.deps.Accounts.GetByUsername(ctx, parsed.Get("actor"))
		if err != nil {
			return e.failErr(command, err)
		}
		actorID = target.ID
	}
	page, err := e.deps.Moderation.ListHistory(ctx, user, actorID, strings.TrimSpace(parsed.Get("q")), parsed.Get("action"), parsed.Get("entity"), parsed.Get("sort"), limit, offset)
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentHistoryPage(page))
}

func (e *Engine) historyActor(ctx context.Context, args []string) int {
	const command = "history.actor"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("q", "action", "entity", "sort", "limit", "offset"); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(command, catalogUsage(command))
	}
	if !historyActorFiltered(parsed) {
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
	limit, offset, ok := e.pageArgs(command, parsed)
	if !ok {
		return 2
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
	page, err := e.deps.Moderation.ListHistory(ctx, user, target.ID, strings.TrimSpace(parsed.Get("q")), parsed.Get("action"), parsed.Get("entity"), parsed.Get("sort"), limit, offset)
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentHistoryPage(page))
}

func (e *Engine) historyActors(ctx context.Context, args []string) int {
	const command = "history.actors"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("q"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "moderator") {
		return 1
	}
	actors, err := e.deps.Moderation.SearchHistoryActors(ctx, user, strings.TrimSpace(parsed.Get("q")))
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentHistoryActors(actors))
}

func historyRecentFiltered(parsed Args) bool {
	return parsed.Has("q") || parsed.Has("actor") || parsed.Has("action") || parsed.Has("entity") || parsed.Has("sort") || parsed.Has("offset")
}

func historyActorFiltered(parsed Args) bool {
	return parsed.Has("q") || parsed.Has("action") || parsed.Has("entity") || parsed.Has("sort") || parsed.Has("offset")
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
	if err := parsed.Only("q", "role", "sort", "limit", "offset"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	if !usersListFiltered(parsed) {
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
	limit, offset, ok := e.pageArgs(command, parsed)
	if !ok {
		return 2
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	if !e.requireMin(command, user, "admin") {
		return 1
	}
	page, err := e.deps.Moderation.ListUserPage(ctx, user, strings.TrimSpace(parsed.Get("q")), parsed.Get("role"), parsed.Get("sort"), limit, offset)
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentUserPage(page))
}

func usersListFiltered(parsed Args) bool {
	return parsed.Has("q") || parsed.Has("role") || parsed.Has("sort") || parsed.Has("limit") || parsed.Has("offset")
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
