package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/agentcli"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/site"
	"github.com/astra-chanUwU/kura-v4/internal/storage"
)

// Agent runs one `kura agent` invocation. schema, help, and a missing
// subcommand never open the database or the media directory.
func Agent(ctx context.Context, cfg config.Config, args []string, out io.Writer) int {
	if len(args) == 0 || args[0] == "schema" || args[0] == "help" {
		return agentcli.Execute(ctx, agentcli.Deps{}, args, out, agentcli.Options{})
	}
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, err := db.Open(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return failAgent(out, err)
	}
	defer conn.Close()
	media, err := storage.NewLocal(cfg.MediaDir)
	if err != nil {
		return failAgent(out, err)
	}
	defer media.Close()
	svc := site.NewService(repository.NewSite(dao.NewSite(conn)))
	accountsSvc, err := accounts.NewService(repository.NewAccounts(dao.NewAccounts(conn)), cfg.PublicURL, media)
	if err != nil {
		return failAgent(out, err)
	}
	postsRepo := repository.NewPosts(dao.NewPosts(conn))
	boardSvc := board.NewService(postsRepo, media)
	collectionsSvc := collections.NewService(repository.NewCollections(dao.NewCollections(conn)))
	moderationSvc := moderation.NewService(repository.NewModeration(dao.NewAccounts(conn), postsRepo, dao.NewAudit(conn)))
	// Fail clearly if migrations have not been applied.
	readyCtx, readyCancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = svc.Get(readyCtx)
	readyCancel()
	if err != nil {
		return failAgent(out, fmt.Errorf("site schema unavailable; run the migrate command before using the agent CLI: %w", err))
	}
	return agentcli.Execute(ctx, agentcli.Deps{
		Accounts:    accountsSvc,
		Board:       boardSvc,
		Collections: collectionsSvc,
		Moderation:  moderationSvc,
		Site:        svc,
	}, args, out, agentcli.Options{})
}

func failAgent(out io.Writer, err error) int {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(agentcli.Envelope{
		OK:      false,
		Command: "agent",
		Error: &agentcli.CLIError{
			Code:    "internal",
			Message: "Something went wrong. Please try again.",
		},
	})
	fmt.Fprintf(os.Stderr, "kura agent: %s\n", err)
	return 1
}
