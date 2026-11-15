package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Accounts struct {
	dao *dao.AccountsDAO
}

func NewAccounts(d *dao.AccountsDAO) *Accounts { return &Accounts{dao: d} }

func (r *Accounts) DB() *sql.DB { return r.dao.DB() }

func (r *Accounts) WithTx(tx *sql.Tx) *dao.AccountsDAO { return r.dao.WithTx(tx) }

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

func (r *Accounts) CreateUserWithID(ctx context.Context, id, username, display string) (model.User, error) {
	return r.dao.CreateUserWithID(ctx, id, username, display)
}

func (r *Accounts) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	user, err := r.dao.GetUserByUsername(ctx, username)
	return user, NotFound(err)
}

func (r *Accounts) GetUserByID(ctx context.Context, id string) (model.User, error) {
	user, err := r.dao.GetUserByID(ctx, id)
	return user, NotFound(err)
}

func (r *Accounts) UpdateProfile(ctx context.Context, u model.User) (model.User, error) {
	user, err := r.dao.UpdateProfile(ctx, u)
	return user, NotFound(err)
}

func (r *Accounts) UpsertPassword(ctx context.Context, userID, hash string) error {
	return r.dao.UpsertPassword(ctx, userID, hash)
}

func (r *Accounts) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	hash, err := r.dao.GetPasswordHash(ctx, userID)
	return hash, NotFound(err)
}

func (r *Accounts) DeletePassword(ctx context.Context, userID string) error {
	return r.dao.DeletePassword(ctx, userID)
}

func (r *Accounts) HasPassword(ctx context.Context, userID string) (bool, error) {
	return r.dao.HasPassword(ctx, userID)
}

func (r *Accounts) InsertCredential(ctx context.Context, userID string, c model.StoredCredential) (string, error) {
	return r.dao.InsertCredential(ctx, userID, c)
}

func (r *Accounts) ListCredentials(ctx context.Context, userID string) ([]model.StoredCredential, error) {
	return r.dao.ListCredentials(ctx, userID)
}

func (r *Accounts) TouchCredential(ctx context.Context, rowID string, signCount int64) error {
	return r.dao.TouchCredential(ctx, rowID, signCount)
}

func (r *Accounts) DeleteCredential(ctx context.Context, rowID, userID string) (int64, error) {
	return r.dao.DeleteCredential(ctx, rowID, userID)
}

func (r *Accounts) CreateSession(ctx context.Context, hash []byte, userID string, expires time.Time) error {
	return r.dao.CreateSession(ctx, hash, userID, expires)
}

func (r *Accounts) SessionUserID(ctx context.Context, hash []byte) (string, error) {
	id, err := r.dao.SessionUserID(ctx, hash)
	return id, NotFound(err)
}

func (r *Accounts) DeleteSession(ctx context.Context, hash []byte) error {
	return r.dao.DeleteSession(ctx, hash)
}

func (r *Accounts) CreateChallenge(ctx context.Context, purpose, username string, userID *string, data []byte, expires time.Time) (string, error) {
	return r.dao.CreateChallenge(ctx, purpose, username, userID, data, expires)
}

func (r *Accounts) TakeChallenge(ctx context.Context, id string) (model.Challenge, error) {
	challenge, err := r.dao.TakeChallenge(ctx, id)
	return challenge, NotFound(err)
}

func (r *Accounts) SetAvatar(ctx context.Context, userID, key, contentType string) error {
	return r.dao.SetAvatar(ctx, userID, key, contentType)
}

func (r *Accounts) UpdateShowNSFW(ctx context.Context, userID string, show bool) error {
	return r.dao.UpdateShowNSFW(ctx, userID, show)
}

func (r *Accounts) UpdateTagPreferences(ctx context.Context, userID string, includeTags, excludeTags []string) error {
	return r.dao.UpdateTagPreferences(ctx, userID, includeTags, excludeTags)
}

func (r *Accounts) UpdateAppearance(ctx context.Context, userID, theme, font string) error {
	return r.dao.UpdateAppearance(ctx, userID, theme, font)
}

func (r *Accounts) UpdateTagSuggestionRows(ctx context.Context, userID string, rows int) error {
	return r.dao.UpdateTagSuggestionRows(ctx, userID, rows)
}

func (r *Accounts) UpdatePostsPerPage(ctx context.Context, userID string, n int) error {
	return r.dao.UpdatePostsPerPage(ctx, userID, n)
}

func (r *Accounts) SetRecoveryKey(ctx context.Context, userID string, keyHash []byte) error {
	return r.dao.SetRecoveryKey(ctx, userID, keyHash)
}

func (r *Accounts) GetRecoveryKey(ctx context.Context, userID string) ([]byte, time.Time, error) {
	keyHash, created, err := r.dao.GetRecoveryKey(ctx, userID)
	return keyHash, created, NotFound(err)
}
