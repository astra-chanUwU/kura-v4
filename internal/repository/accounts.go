package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Accounts struct{ *dao.AccountsDAO }

func NewAccounts(d *dao.AccountsDAO) *Accounts { return &Accounts{AccountsDAO: d} }

// CreateUserWithPassword inserts the account and its password atomically.
func (r *Accounts) CreateUserWithPassword(ctx context.Context, username, display, hash string) (model.User, error) {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	user, err := d.CreateUser(ctx, username, display)
	if err != nil {
		return model.User{}, err
	}
	if err := d.UpsertPassword(ctx, user.ID, hash); err != nil {
		return model.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, err
	}
	return user, nil
}

// RecoverAccount replaces the password and recovery key and revokes every
// passkey and session, atomically: after recovery only the new password
// (and the newly issued recovery key) can get back in.
func (r *Accounts) RecoverAccount(ctx context.Context, userID, passwordHash string, recoveryHash []byte) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.UpsertPassword(ctx, userID, passwordHash); err != nil {
		return err
	}
	if err := d.SetRecoveryKey(ctx, userID, recoveryHash); err != nil {
		return err
	}
	if err := d.DeleteCredentialsForUser(ctx, userID); err != nil {
		return err
	}
	if err := d.DeleteSessionsForUser(ctx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateUserWithCredential inserts the account and its first passkey atomically.
func (r *Accounts) CreateUserWithCredential(ctx context.Context, id, username, display string, cred model.StoredCredential) (model.User, error) {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	user, err := d.CreateUserWithID(ctx, id, username, display)
	if err != nil {
		return model.User{}, err
	}
	if _, err := d.InsertCredential(ctx, user.ID, cred); err != nil {
		return model.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.User{}, err
	}
	return user, nil
}
