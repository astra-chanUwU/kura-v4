package agentcli

import (
	"context"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

func catalogUsage(name string) string {
	for _, cmd := range Catalog() {
		if cmd.Name == name {
			return cmd.Usage
		}
	}
	return ""
}

func (e *Engine) favorites(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("favorites", "usage: kura agent favorites <list|add|remove>")
	}
	switch args[0] {
	case "list":
		return e.favoritesList(ctx, args[1:])
	case "add":
		return e.favoritesAdd(ctx, args[1:])
	case "remove":
		return e.favoritesRemove(ctx, args[1:])
	default:
		return e.badUsage("favorites", "usage: kura agent favorites <list|add|remove>")
	}
}

func (e *Engine) favoritesList(ctx context.Context, args []string) int {
	const command = "favorites.list"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("limit", "offset"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	limit, offset, ok := e.pageArgs(command, parsed)
	if !ok {
		return 2
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	page, err := e.deps.Collections.Favorites(ctx, user.ID, SeesNSFW(&user), limit, offset)
	if err != nil {
		return e.failErr(command, err)
	}
	fav := make(map[string]bool, len(page.Posts))
	for _, post := range page.Posts {
		fav[post.ID] = true
	}
	return e.ok(command, map[string]any{
		"total": page.Total,
		"posts": presentPosts(page.Posts, fav),
	})
}

func (e *Engine) favoritesAdd(ctx context.Context, args []string) int {
	const command = "favorites.add"
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
	id := parsed.Pos[0]
	if err := e.deps.Collections.Favorite(ctx, user.ID, id); err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, map[string]any{"post_id": id})
}

func (e *Engine) favoritesRemove(ctx context.Context, args []string) int {
	const command = "favorites.remove"
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
	id := parsed.Pos[0]
	if err := e.deps.Collections.Unfavorite(ctx, user.ID, id); err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, map[string]any{"post_id": id})
}

func (e *Engine) lists(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("lists", "usage: kura agent lists <list|get|create|add|remove>")
	}
	switch args[0] {
	case "list":
		return e.listsList(ctx, args[1:])
	case "get":
		return e.listsGet(ctx, args[1:])
	case "create":
		return e.listsCreate(ctx, args[1:])
	case "add":
		return e.listsAdd(ctx, args[1:])
	case "remove":
		return e.listsRemove(ctx, args[1:])
	default:
		return e.badUsage("lists", "usage: kura agent lists <list|get|create|add|remove>")
	}
}

func (e *Engine) listsList(ctx context.Context, args []string) int {
	const command = "lists.list"
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
	lists, err := e.deps.Collections.MyLists(ctx, user.ID)
	if err != nil {
		return e.failErr(command, err)
	}
	if lists == nil {
		lists = []model.List{}
	}
	return e.ok(command, presentLists(lists))
}

func (e *Engine) listsGet(ctx context.Context, args []string) int {
	const command = "lists.get"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("limit", "offset"); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(command, catalogUsage(command))
	}
	limit, offset, ok := e.pageArgs(command, parsed)
	if !ok {
		return 2
	}
	user, err := e.actor(ctx)
	if err != nil {
		return e.failErr(command, err)
	}
	var viewer *string
	if user != nil {
		viewer = &user.ID
	}
	detail, err := e.deps.Collections.GetList(ctx, viewer, parsed.Pos[0], SeesNSFW(user), limit, offset)
	if err != nil {
		return e.failErr(command, err)
	}
	fav := map[string]bool{}
	if user != nil {
		fav = e.favs(ctx, user.ID)
	}
	return e.ok(command, dto.ListDetail{
		List:  presentList(detail.List),
		Items: presentPosts(detail.Items, fav),
		Total: detail.Total,
	})
}

func (e *Engine) listsCreate(ctx context.Context, args []string) int {
	const command = "lists.create"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(command, err.Error())
	}
	if err := parsed.Only("title", "description", "visibility"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(command, catalogUsage(command))
	}
	title := strings.TrimSpace(parsed.Get("title"))
	if title == "" {
		return e.badUsage(command, catalogUsage(command))
	}
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	list, err := e.deps.Collections.CreateList(ctx, user.ID, title, parsed.Get("description"), parsed.Get("visibility"))
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentList(list))
}

func (e *Engine) listsAdd(ctx context.Context, args []string) int {
	const command = "lists.add"
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
	listID, postID := parsed.Pos[0], parsed.Pos[1]
	if err := e.deps.Collections.AddItem(ctx, user.ID, listID, postID); err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, map[string]any{"list_id": listID, "post_id": postID})
}

func (e *Engine) listsRemove(ctx context.Context, args []string) int {
	const command = "lists.remove"
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
	listID, postID := parsed.Pos[0], parsed.Pos[1]
	if err := e.deps.Collections.RemoveItem(ctx, user.ID, listID, postID); err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, map[string]any{"list_id": listID, "post_id": postID})
}
