package app

import (
	"context"
	"fmt"
	"io"

	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
)

// Promote sets a user's role from the operator shell:
//
//	kura promote <username> <member|moderator|admin|super_admin>
//
// This is the bootstrap path for the first super admin (no account can
// grant super_admin until one exists) and an emergency recovery path.
// It bypasses the in-app hierarchy by design — shell/DB access is already
// operator trust — but still writes a history row (actor "cli").
func Promote(ctx context.Context, cfg config.Config, username, roleName string, out io.Writer) error {
	role := model.Role(roleName)
	if !role.Valid() {
		return fmt.Errorf("role must be member, moderator, admin or super_admin")
	}
	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck
	accountsDAO := dao.NewAccounts(conn)
	user, err := accountsDAO.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("no such user %q", username)
	}
	if user.Role == role {
		fmt.Fprintf(out, "%s is already %s\n", username, role) //nolint:errcheck
		return nil
	}
	repo := repository.NewModeration(accountsDAO, repository.NewPosts(dao.NewPosts(conn)), dao.NewAudit(conn))
	entry := model.ChangeEntry{EntityType: "user", EntityID: user.ID, ActorUsername: "cli:promote",
		Action: "user.role_changed", Before: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: user.Role}),
		After: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: role})}
	if _, err := repo.ApplyUserRole(ctx, user.ID, role, entry); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s: %s -> %s\n", username, user.Role, role) //nolint:errcheck
	return nil
}
