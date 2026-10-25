package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type AccountsDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewAccounts(conn *sql.DB) *AccountsDAO { return &AccountsDAO{db: conn, q: sqlcgen.New(conn)} }
func (d *AccountsDAO) WithTx(tx *sql.Tx) *AccountsDAO {
	return &AccountsDAO{db: d.db, q: sqlcgen.New(tx)}
}
func (d *AccountsDAO) DB() *sql.DB { return d.db }

func parseUUID(s string) (uuid.UUID, error) { return uuid.Parse(s) }
func nullUUID(s *string) uuid.NullUUID {
	if s == nil {
		return uuid.NullUUID{}
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
func tsPtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func userFrom(id uuid.UUID, username, display, bio, location, website string, created time.Time, avatarKey, avatarType string, showNSFW bool) model.User {
	return model.User{ID: id.String(), Username: username, DisplayName: display, Bio: bio, Location: location, Website: website, CreatedAt: created, AvatarKey: avatarKey, AvatarType: avatarType, ShowNSFW: showNSFW}
}

func (d *AccountsDAO) CreateUser(ctx context.Context, username, display string) (model.User, error) {
	r, err := d.q.CreateUser(ctx, sqlcgen.CreateUserParams{Username: username, DisplayName: display})
	if err != nil {
		return model.User{}, err
	}
	return userFrom(r.ID, r.Username, r.DisplayName, r.Bio, r.Location, r.Website, r.CreatedAt, r.AvatarKey, r.AvatarType, r.ShowNsfw), nil
}
func (d *AccountsDAO) CreateUserWithID(ctx context.Context, id, username, display string) (model.User, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return model.User{}, err
	}
	r, err := d.q.CreateUserWithID(ctx, sqlcgen.CreateUserWithIDParams{ID: uid, Username: username, DisplayName: display})
	if err != nil {
		return model.User{}, err
	}
	return userFrom(r.ID, r.Username, r.DisplayName, r.Bio, r.Location, r.Website, r.CreatedAt, r.AvatarKey, r.AvatarType, r.ShowNsfw), nil
}
func (d *AccountsDAO) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	r, err := d.q.GetUserByUsername(ctx, username)
	if err != nil {
		return model.User{}, err
	}
	return userFrom(r.ID, r.Username, r.DisplayName, r.Bio, r.Location, r.Website, r.CreatedAt, r.AvatarKey, r.AvatarType, r.ShowNsfw), nil
}
func (d *AccountsDAO) GetUserByID(ctx context.Context, id string) (model.User, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return model.User{}, err
	}
	r, err := d.q.GetUserByID(ctx, uid)
	if err != nil {
		return model.User{}, err
	}
	return userFrom(r.ID, r.Username, r.DisplayName, r.Bio, r.Location, r.Website, r.CreatedAt, r.AvatarKey, r.AvatarType, r.ShowNsfw), nil
}
func (d *AccountsDAO) UpdateProfile(ctx context.Context, u model.User) (model.User, error) {
	uid, err := parseUUID(u.ID)
	if err != nil {
		return model.User{}, err
	}
	r, err := d.q.UpdateProfile(ctx, sqlcgen.UpdateProfileParams{ID: uid, DisplayName: u.DisplayName, Bio: u.Bio, Location: u.Location, Website: u.Website})
	if err != nil {
		return model.User{}, err
	}
	return userFrom(r.ID, r.Username, r.DisplayName, r.Bio, r.Location, r.Website, r.CreatedAt, r.AvatarKey, r.AvatarType, r.ShowNsfw), nil
}
func (d *AccountsDAO) UpdateShowNSFW(ctx context.Context, userID string, show bool) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.UpdateShowNSFW(ctx, sqlcgen.UpdateShowNSFWParams{ID: uid, ShowNsfw: show})
}
func (d *AccountsDAO) SetAvatar(ctx context.Context, userID, key, contentType string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.UpdateAvatar(ctx, sqlcgen.UpdateAvatarParams{ID: uid, AvatarKey: key, AvatarType: contentType})
}
func (d *AccountsDAO) UpsertPassword(ctx context.Context, userID, hash string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.UpsertPassword(ctx, sqlcgen.UpsertPasswordParams{UserID: uid, PasswordHash: hash})
}
func (d *AccountsDAO) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return "", err
	}
	return d.q.GetPasswordHash(ctx, uid)
}
func (d *AccountsDAO) DeletePassword(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.DeletePassword(ctx, uid)
}
func (d *AccountsDAO) HasPassword(ctx context.Context, userID string) (bool, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return false, err
	}
	return d.q.HasPassword(ctx, uid)
}
func (d *AccountsDAO) InsertCredential(ctx context.Context, userID string, c model.StoredCredential) (string, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return "", err
	}
	id, err := d.q.InsertWebauthnCredential(ctx, sqlcgen.InsertWebauthnCredentialParams{
		UserID: uid, CredentialID: c.CredentialID, PublicKey: c.PublicKey, AttestationType: c.AttestationType,
		Transports: c.Transports, UserPresent: c.UserPresent, UserVerified: c.UserVerified,
		BackupEligible: c.BackupEligible, BackupState: c.BackupState, Aaguid: c.AAGUID, SignCount: c.SignCount, DisplayName: c.Name})
	return id.String(), err
}
func (d *AccountsDAO) ListCredentials(ctx context.Context, userID string) ([]model.StoredCredential, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListWebauthnCredentials(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]model.StoredCredential, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.StoredCredential{RowID: r.ID.String(), CredentialID: r.CredentialID, PublicKey: r.PublicKey,
			AttestationType: r.AttestationType, Transports: r.Transports, UserPresent: r.UserPresent, UserVerified: r.UserVerified,
			BackupEligible: r.BackupEligible, BackupState: r.BackupState, AAGUID: r.Aaguid, SignCount: r.SignCount,
			Name: r.DisplayName, CreatedAt: r.CreatedAt, LastUsedAt: tsPtr(r.LastUsedAt)})
	}
	return out, nil
}
func (d *AccountsDAO) TouchCredential(ctx context.Context, rowID string, signCount int64) error {
	uid, err := parseUUID(rowID)
	if err != nil {
		return err
	}
	return d.q.TouchWebauthnCredential(ctx, sqlcgen.TouchWebauthnCredentialParams{ID: uid, SignCount: signCount})
}
func (d *AccountsDAO) DeleteCredential(ctx context.Context, rowID, userID string) (int64, error) {
	a, err := parseUUID(rowID)
	if err != nil {
		return 0, err
	}
	b, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	return d.q.DeleteWebauthnCredential(ctx, sqlcgen.DeleteWebauthnCredentialParams{ID: a, UserID: b})
}
func (d *AccountsDAO) CreateSession(ctx context.Context, hash []byte, userID string, expires time.Time) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.CreateSession(ctx, sqlcgen.CreateSessionParams{TokenHash: hash, UserID: uid, ExpiresAt: expires})
}
func (d *AccountsDAO) SessionUserID(ctx context.Context, hash []byte) (string, error) {
	uid, err := d.q.GetSessionUserID(ctx, hash)
	return uid.String(), err
}
func (d *AccountsDAO) DeleteSession(ctx context.Context, hash []byte) error {
	return d.q.DeleteSession(ctx, hash)
}
func (d *AccountsDAO) CreateChallenge(ctx context.Context, purpose, username string, userID *string, data []byte, expires time.Time) (string, error) {
	id, err := d.q.CreateChallenge(ctx, sqlcgen.CreateChallengeParams{Purpose: purpose, Username: username, UserID: nullUUID(userID), SessionData: data, ExpiresAt: expires})
	return id.String(), err
}
func (d *AccountsDAO) TakeChallenge(ctx context.Context, id string) (model.Challenge, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return model.Challenge{}, err
	}
	r, err := d.q.TakeChallenge(ctx, uid)
	if err != nil {
		return model.Challenge{}, err
	}
	ch := model.Challenge{Purpose: r.Purpose, Username: r.Username, SessionData: r.SessionData}
	if r.UserID.Valid {
		s := r.UserID.UUID.String()
		ch.UserID = &s
	}
	return ch, nil
}

func (d *AccountsDAO) SetRecoveryKey(ctx context.Context, userID string, keyHash []byte) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.SetRecoveryKey(ctx, sqlcgen.SetRecoveryKeyParams{UserID: uid, KeyHash: keyHash})
}
func (d *AccountsDAO) GetRecoveryKey(ctx context.Context, userID string) ([]byte, time.Time, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, time.Time{}, err
	}
	r, err := d.q.GetRecoveryKey(ctx, uid)
	if err != nil {
		return nil, time.Time{}, err
	}
	return r.KeyHash, r.CreatedAt, nil
}
func (d *AccountsDAO) DeleteSessionsForUser(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.DeleteSessionsForUser(ctx, uid)
}
func (d *AccountsDAO) DeleteCredentialsForUser(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return d.q.DeleteCredentialsForUser(ctx, uid)
}
