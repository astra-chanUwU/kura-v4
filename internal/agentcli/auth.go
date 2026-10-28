package agentcli

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
)

func (e *Engine) login(ctx context.Context, args []string) int {
	const cmd = "login"
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only("username", "password"); err != nil || len(parsed.Pos) != 0 {
		return e.badUsage(cmd, loginUsage())
	}
	if !parsed.Has("username") || strings.TrimSpace(parsed.Get("username")) == "" {
		return e.badUsage(cmd, loginUsage())
	}
	password := os.Getenv("KURA_AGENT_PASSWORD")
	if parsed.Has("password") {
		password = parsed.Get("password")
	}
	if password == "" {
		return e.badUsage(cmd, "Set KURA_AGENT_PASSWORD or pass --password.")
	}
	if e.deps.Accounts == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	user, token, err := e.deps.Accounts.LoginWithPassword(ctx, parsed.Get("username"), password)
	if err != nil {
		return e.failErr(cmd, err)
	}
	if err := SaveSession(e.sessionPath, token); err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, Identity{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        string(user.Role),
		ShowNSFW:    user.ShowNSFW,
	})
}

func (e *Engine) logout(ctx context.Context) int {
	const cmd = "logout"
	token, err := LoadSession(e.sessionPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := ClearSession(e.sessionPath); err != nil {
			return e.failErr(cmd, err)
		}
		return e.ok(cmd, map[string]any{"signed_out": true})
	}
	if err != nil {
		return e.failErr(cmd, err)
	}
	if e.deps.Accounts != nil {
		err := e.deps.Accounts.Logout(ctx, token)
		if err != nil && !errors.Is(err, accounts.ErrNotFound) {
			return e.failErr(cmd, err)
		}
	}
	if err := ClearSession(e.sessionPath); err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, map[string]any{"signed_out": true})
}

func (e *Engine) whoami(ctx context.Context) int {
	const cmd = "whoami"
	user, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	if user == nil {
		return e.ok(cmd, Identity{Role: "visitor"})
	}
	var contributions int64
	if e.deps.Board != nil {
		contributions, err = e.deps.Board.Contributions(ctx, user.ID)
		if err != nil {
			return e.failErr(cmd, err)
		}
	}
	return e.ok(cmd, Identity{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Role:          string(user.Role),
		ShowNSFW:      user.ShowNSFW,
		Contributions: contributions,
	})
}

func (e *Engine) capabilities(ctx context.Context) int {
	const cmd = "capabilities"
	user, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	role := "visitor"
	actor := Identity{Role: role}
	if user != nil {
		role = string(user.Role)
		actor = Identity{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        role,
			ShowNSFW:    user.ShowNSFW,
		}
	}
	return e.ok(cmd, map[string]any{
		"actor":    actor,
		"commands": Allowed(role),
	})
}

func (e *Engine) site(ctx context.Context) int {
	const cmd = "site"
	if e.deps.Site == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	site, err := e.deps.Site.Get(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, dto.Site{Name: site.Name, Description: site.Description})
}

func loginUsage() string {
	for _, cmd := range Catalog() {
		if cmd.Name == "login" {
			return cmd.Usage
		}
	}
	return "kura agent login --username <name> [--password <secret>]"
}
