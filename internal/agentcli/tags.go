package agentcli

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const tagsUsage = "usage: kura agent tags <show|edit|alias|unalias|imply|unimply>"

func (e *Engine) tags(ctx context.Context, args []string) int {
	if e.deps.Tags == nil {
		return e.fail("tags", "internal", "Tag pages are not available.")
	}
	if len(args) == 0 {
		return e.badUsage("tags", tagsUsage)
	}
	switch args[0] {
	case "show":
		return e.tagsShow(ctx, args[1:])
	case "edit":
		return e.tagsEdit(ctx, args[1:])
	case "alias":
		return e.tagsAlias(ctx, args[1:], true)
	case "unalias":
		return e.tagsAlias(ctx, args[1:], false)
	case "imply":
		return e.tagsImply(ctx, args[1:], true)
	case "unimply":
		return e.tagsImply(ctx, args[1:], false)
	default:
		return e.badUsage("tags", tagsUsage)
	}
}

func (e *Engine) tagsShow(ctx context.Context, args []string) int {
	const cmd = "tags.show"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, "usage: kura agent tags show <tag>")
	}
	user, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	page, err := e.deps.Tags.Get(ctx, user, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentTagPage(page))
}

func (e *Engine) tagsEdit(ctx context.Context, args []string) int {
	const cmd = "tags.edit"
	user, ok := e.requireActor(ctx, cmd)
	if !ok {
		return 1
	}
	if !e.requireMin(cmd, user, "member") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only("category", "description"); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, "usage: kura agent tags edit <tag> --category <name> --description <text>")
	}
	if !parsed.Has("category") || !parsed.Has("description") {
		return e.badUsage(cmd, "tags edit needs both --category and --description.")
	}
	page, err := e.deps.Tags.Update(ctx, user, parsed.Pos[0], parsed.Get("category"), parsed.Get("description"))
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentTagPage(page))
}

func (e *Engine) tagsAlias(ctx context.Context, args []string, add bool) int {
	cmd := "tags.unalias"
	usage := "usage: kura agent tags unalias <target> <alias>"
	if add {
		cmd = "tags.alias"
		usage = "usage: kura agent tags alias <target> <alias>"
	}
	user, ok := e.requireActor(ctx, cmd)
	if !ok {
		return 1
	}
	if !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 2 {
		return e.badUsage(cmd, usage)
	}
	var page model.TagPage
	if add {
		page, err = e.deps.Tags.AddAlias(ctx, user, parsed.Pos[0], parsed.Pos[1])
	} else {
		page, err = e.deps.Tags.RemoveAlias(ctx, user, parsed.Pos[0], parsed.Pos[1])
	}
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentTagPage(page))
}

func (e *Engine) tagsImply(ctx context.Context, args []string, add bool) int {
	cmd := "tags.unimply"
	usage := "usage: kura agent tags unimply <antecedent> <consequent>"
	if add {
		cmd = "tags.imply"
		usage = "usage: kura agent tags imply <antecedent> <consequent>"
	}
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 2 {
		return e.badUsage(cmd, usage)
	}
	var page model.TagPage
	if add {
		page, err = e.deps.Tags.AddImplication(ctx, user, parsed.Pos[0], parsed.Pos[1])
	} else {
		page, err = e.deps.Tags.RemoveImplication(ctx, user, parsed.Pos[0], parsed.Pos[1])
	}
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentTagPage(page))
}

func stringList(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

func presentTagPage(page model.TagPage) map[string]any {
	aliases := page.Aliases
	if aliases == nil {
		aliases = []string{}
	}
	updated := ""
	if !page.UpdatedAt.IsZero() {
		updated = page.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return map[string]any{
		"tag": page.Tag, "alias_of": page.AliasOf, "aliases": aliases,
		"implies": stringList(page.Implies), "implied_by": stringList(page.ImpliedBy),
		"category": page.Category, "description": page.Description,
		"post_count": page.PostCount, "updated_by_username": page.UpdatedByUsername,
		"updated_at": updated, "posts": presentPosts(page.Posts, map[string]bool{}),
	}
}
