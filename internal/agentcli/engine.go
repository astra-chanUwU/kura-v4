package agentcli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// Engine runs one agent invocation. Command methods live in auth.go,
// posts.go, collect.go, and staff.go. They call services and write one
// JSON envelope. They do not open the database and they do not bypass
// service permission checks.
type Engine struct {
	deps        Deps
	out         io.Writer
	sessionPath string
}

// Execute runs one agent command. args is everything after `kura agent`.
// The return value is the process exit code.
func Execute(ctx context.Context, deps Deps, args []string, out io.Writer, opt Options) int {
	if out == nil {
		out = io.Discard
	}
	e := &Engine{deps: deps, out: out, sessionPath: opt.SessionPath}
	if e.sessionPath == "" {
		e.sessionPath = DefaultSessionPath()
	}
	return e.dispatch(ctx, args)
}

func (e *Engine) dispatch(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.usage()
	}
	switch args[0] {
	case "schema", "help":
		return e.ok("schema", map[string]any{"commands": Catalog()})
	case "login":
		return e.login(ctx, args[1:])
	case "logout":
		return e.logout(ctx)
	case "whoami":
		return e.whoami(ctx)
	case "capabilities":
		return e.capabilities(ctx)
	case "site":
		return e.site(ctx)
	case "posts":
		return e.posts(ctx, args[1:])
	case "favorites":
		return e.favorites(ctx, args[1:])
	case "lists":
		return e.lists(ctx, args[1:])
	case "history":
		return e.history(ctx, args[1:])
	case "users":
		return e.users(ctx, args[1:])
	default:
		return e.usage()
	}
}

// actor loads the stored session. A missing file is a visitor (nil user).
// An unreadable or expired session is an error; callers must not treat
// that as a visitor, because a moderator command would then silently drop
// privileges.
func (e *Engine) actor(ctx context.Context) (*model.User, error) {
	token, err := LoadSession(e.sessionPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if e.deps.Accounts == nil {
		return nil, errors.New("accounts service is not available")
	}
	user, err := e.deps.Accounts.UserForSession(ctx, token)
	if errors.Is(err, accounts.ErrNotFound) {
		return nil, ErrExpiredSession
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// requireActor fails the command when nobody is signed in.
func (e *Engine) requireActor(ctx context.Context, command string) (model.User, bool) {
	user, err := e.actor(ctx)
	if err != nil {
		e.failErr(command, err)
		return model.User{}, false
	}
	if user == nil {
		e.fail(command, "unauthenticated", "Sign in first. Run `kura agent login --username <name>`.")
		return model.User{}, false
	}
	return *user, true
}

// requireMin applies the catalog role gate before the service is called.
// The service still enforces its own rules.
func (e *Engine) requireMin(command string, user model.User, minRole string) bool {
	if Allows(string(user.Role), minRole) {
		return true
	}
	e.fail(command, "forbidden", "You don't have permission to do that.")
	return false
}

func (e *Engine) favs(ctx context.Context, userID string) map[string]bool {
	if userID == "" || e.deps.Collections == nil {
		return map[string]bool{}
	}
	ids, err := e.deps.Collections.FavoriteIDs(ctx, userID)
	if err != nil || ids == nil {
		return map[string]bool{}
	}
	return ids
}

func (e *Engine) actorID(user *model.User) string {
	if user == nil {
		return ""
	}
	return user.ID
}
