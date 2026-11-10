package agentcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/comments"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/astra-chanUwU/kura-v4/internal/tags"
)

// Envelope is the only stdout document an agent command writes.
// Exit 0 is ok true. Exit 1 is an operation error. Exit 2 is usage.
type Envelope struct {
	OK      bool      `json:"ok"`
	Command string    `json:"command"`
	Data    any       `json:"data,omitempty"`
	Error   *CLIError `json:"error,omitempty"`
}

// CLIError is a stable machine-readable failure.
type CLIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Engine) ok(command string, data any) int {
	return e.write(Envelope{OK: true, Command: command, Data: data})
}

func (e *Engine) fail(command, code, message string) int {
	return e.write(Envelope{OK: false, Command: command, Error: &CLIError{Code: code, Message: message}})
}

func (e *Engine) failData(command, code, message string, data any) int {
	return e.write(Envelope{OK: false, Command: command, Data: data, Error: &CLIError{Code: code, Message: message}})
}

func (e *Engine) badUsage(command, message string) int {
	return e.fail(command, "usage", message)
}

func (e *Engine) usage() int {
	return e.fail("agent", "usage", "usage: kura agent <command>. Run `kura agent schema` for the command list.")
}

func (e *Engine) write(env Envelope) int {
	enc := json.NewEncoder(e.out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		fmt.Fprintf(os.Stderr, "kura agent: %v\n", err)
		return 1
	}
	if env.OK {
		return 0
	}
	if env.Error != nil && env.Error.Code == "usage" {
		return 2
	}
	return 1
}

// failErr maps a service error onto the same codes the HTTP API uses.
// Unexpected failures keep their detail on stderr and a generic message
// on stdout, so a database error is not pasted into an agent transcript.
func (e *Engine) failErr(command string, err error) int {
	if err == nil {
		return e.fail(command, "internal", "Something went wrong. Please try again.")
	}
	code, message, unexpected := classify(err)
	if unexpected {
		fmt.Fprintf(os.Stderr, "kura agent: %s: %v\n", command, err)
	}
	return e.fail(command, code, message)
}

func classify(err error) (code, message string, unexpected bool) {
	var aErr *accounts.ValidationError
	var bErr *board.ValidationError
	var cErr *collections.ValidationError
	var commentErr *comments.ValidationError
	var mErr *moderation.ValidationError
	var tErr *tags.ValidationError
	switch {
	case errors.As(err, &aErr):
		return "validation", aErr.Message, false
	case errors.As(err, &bErr):
		return "validation", bErr.Message, false
	case errors.As(err, &cErr):
		return "validation", cErr.Message, false
	case errors.As(err, &commentErr):
		return "validation", commentErr.Message, false
	case errors.As(err, &mErr):
		return "validation", mErr.Message, false
	case errors.As(err, &tErr):
		return "validation", tErr.Message, false
	case errors.Is(err, board.ErrBadSort):
		return "validation", "Sort must be newest, oldest, or favorites.", false
	case errors.Is(err, board.ErrUploadLimit):
		return "validation", board.ErrUploadLimit.Error(), false
	case errors.Is(err, ErrExpiredSession):
		return "unauthenticated", "That session expired. Sign in again with `kura agent login`.", false
	case errors.Is(err, accounts.ErrInvalidCredentials):
		return "unauthenticated", "That username and sign-in didn't match. Check it and try again.", false
	case errors.Is(err, moderation.ErrForbidden), errors.Is(err, collections.ErrForbidden), errors.Is(err, tags.ErrForbidden), errors.Is(err, comments.ErrForbidden):
		return "forbidden", "You don't have permission to do that.", false
	case errors.Is(err, accounts.ErrNotFound), errors.Is(err, board.ErrNotFound), errors.Is(err, collections.ErrNotFound), errors.Is(err, moderation.ErrNotFound), errors.Is(err, tags.ErrNotFound), errors.Is(err, comments.ErrNotFound):
		return "not_found", "Not found.", false
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled", "The command was canceled.", false
	default:
		return "internal", "Something went wrong. Please try again.", true
	}
}
