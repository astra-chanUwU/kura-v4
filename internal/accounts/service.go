package accounts

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrUsernameTaken      = errors.New("username is already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrLastSignInMethod   = errors.New("you need at least one way to sign in — add another before removing this one")
	ErrCeremonyExpired    = errors.New("that sign-in request expired — please try again")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
func invalid(msg string) error           { return &ValidationError{Message: msg} }

type Repository interface {
	CreateUserWithPassword(ctx context.Context, username, display, hash string) (model.User, error)
	CreateUserWithCredential(ctx context.Context, id, username, display string, cred model.StoredCredential) (model.User, error)
	CreateUserWithID(ctx context.Context, id, username, display string) (model.User, error)
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	UpdateProfile(ctx context.Context, u model.User) (model.User, error)
	UpsertPassword(ctx context.Context, userID, hash string) error
	GetPasswordHash(ctx context.Context, userID string) (string, error)
	DeletePassword(ctx context.Context, userID string) error
	HasPassword(ctx context.Context, userID string) (bool, error)
	InsertCredential(ctx context.Context, userID string, c model.StoredCredential) (string, error)
	ListCredentials(ctx context.Context, userID string) ([]model.StoredCredential, error)
	TouchCredential(ctx context.Context, rowID string, signCount int64) error
	DeleteCredential(ctx context.Context, rowID, userID string) (int64, error)
	CreateSession(ctx context.Context, hash []byte, userID string, expires time.Time) error
	SessionUserID(ctx context.Context, hash []byte) (string, error)
	DeleteSession(ctx context.Context, hash []byte) error
	CreateChallenge(ctx context.Context, purpose, username string, userID *string, data []byte, expires time.Time) (string, error)
	TakeChallenge(ctx context.Context, id string) (model.Challenge, error)
	SetAvatar(ctx context.Context, userID, key, contentType string) error
	UpdateShowNSFW(ctx context.Context, userID string, show bool) error
	UpdateTagPreferences(ctx context.Context, userID string, includeTags, excludeTags []string) error
	UpdateAppearance(ctx context.Context, userID, theme, font string) error
	SetRecoveryKey(ctx context.Context, userID string, keyHash []byte) error
	GetRecoveryKey(ctx context.Context, userID string) ([]byte, time.Time, error)
	// RecoverAccount atomically replaces the password and recovery key and
	// revokes every passkey and session.
	RecoverAccount(ctx context.Context, userID, passwordHash string, recoveryHash []byte) error
}

var usernameRE = regexp.MustCompile(`^[a-z0-9_]{3,24}$`)

func NormalizeUsername(raw string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(raw))
	if !usernameRE.MatchString(u) {
		return "", invalid("Usernames use 3–24 lowercase letters, numbers or underscores.")
	}
	return u, nil
}

// Media is the blob store profile pictures live in (content-addressed).
type Media interface {
	PutHashed(context.Context, io.Reader) (string, string, error)
	Open(context.Context, string) (io.ReadCloser, error)
}

type Service struct {
	repo    Repository
	wa      *webauthn.WebAuthn
	media   Media
	rpID    string
	origins []string
}

func NewService(repo Repository, publicURL string, media Media) (*Service, error) {
	u, err := url.Parse(publicURL)
	if err != nil {
		return nil, err
	}
	origins := []string{strings.TrimSuffix(publicURL, "/")}
	host := u.Hostname()
	if host == "localhost" {
		origins = append(origins, strings.Replace(origins[0], "localhost", "127.0.0.1", 1))
	}
	if host == "127.0.0.1" {
		origins = append(origins, strings.Replace(origins[0], "127.0.0.1", "localhost", 1))
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Kura",
		RPID:          host,
		RPOrigins:     origins,
		// Passkeys must be discoverable (resident) so sign-in can start
		// from the saved passkey itself — no username typed first.
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationPreferred,
		},
	})
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, wa: wa, media: media, rpID: host, origins: origins}, nil
}

// bindingForOrigin picks the RPID for a ceremony begun at requestOrigin.
// Browsers reject a passkey ceremony whose rp.id is not a domain suffix
// of the page origin ("This is an invalid domain"), and an IP literal
// can never be an RPID, so a ceremony begun at http://127.0.0.1:5174
// can never complete — it fails fast here with a usable message, and
// the SPA redirects loopback-IP visits to localhost before this runs.
// An empty origin keeps the configured default (non-browser callers).
func (s *Service) bindingForOrigin(requestOrigin string) (rpID, origin string, err error) {
	requestOrigin = strings.TrimSuffix(strings.TrimSpace(requestOrigin), "/")
	if requestOrigin == "" {
		return s.rpID, "", nil
	}
	parsed, err := url.Parse(requestOrigin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", "", invalid("That sign-in origin is not allowed. Please try again.")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil {
		return "", "", invalid("Passkeys don't work at an IP address (" + parsed.Host + "). Open " + parsed.Scheme + "://localhost" + portSuffix(parsed) + " instead — it's the same app, and browsers only allow passkeys on a name like localhost.")
	}
	for _, allowed := range s.origins {
		if strings.EqualFold(strings.TrimSuffix(allowed, "/"), requestOrigin) {
			return parsed.Hostname(), requestOrigin, nil
		}
	}
	return "", "", invalid("That sign-in origin is not allowed. Please try again.")
}

func portSuffix(u *url.URL) string {
	if p := u.Port(); p != "" {
		return ":" + p
	}
	return ""
}

func isNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// --- password accounts ---

func (s *Service) RegisterWithPassword(ctx context.Context, username, password, display string) (model.User, string, string, error) {
	name, err := NormalizeUsername(username)
	if err != nil {
		return model.User{}, "", "", err
	}
	if err := validatePassword(name, password); err != nil {
		return model.User{}, "", "", err
	}
	display = strings.TrimSpace(display)
	if len([]rune(display)) > 60 {
		return model.User{}, "", "", invalid("Display names are capped at 60 characters.")
	}
	if display == "" {
		display = name
	}
	hash, err := HashPassword(password)
	if err != nil {
		return model.User{}, "", "", err
	}
	user, err := s.repo.CreateUserWithPassword(ctx, name, display, hash)
	if isUnique(err) {
		return model.User{}, "", "", ErrUsernameTaken
	}
	if err != nil {
		return model.User{}, "", "", fmt.Errorf("create account: %w", err)
	}
	key, err := s.issueRecoveryKey(ctx, user.ID)
	if err != nil {
		return model.User{}, "", "", err
	}
	token, err := s.newSession(ctx, user.ID)
	return user, token, key, err
}

func (s *Service) LoginWithPassword(ctx context.Context, username, password string) (model.User, string, error) {
	name, err := NormalizeUsername(username)
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	user, err := s.repo.GetUserByUsername(ctx, name)
	if isNotFound(err) {
		_ = CheckPassword(password, dummyHash) // equalize timing; result unused
		return model.User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return model.User{}, "", err
	}
	hash, err := s.repo.GetPasswordHash(ctx, user.ID)
	if isNotFound(err) || !CheckPassword(password, hash) {
		return model.User{}, "", ErrInvalidCredentials
	}
	if NeedsRehash(hash) {
		if upgraded, err := HashPassword(password); err == nil {
			_ = s.repo.UpsertPassword(ctx, user.ID, upgraded)
		}
	}
	token, err := s.newSession(ctx, user.ID)
	return user, token, err
}

// --- sessions ---

func tokenHash(token string) []byte { h := sha256.Sum256([]byte(token)); return h[:] }

func (s *Service) newSession(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := uuid.NewString() + "." + fmt.Sprintf("%x", raw)
	if err := s.repo.CreateSession(ctx, tokenHash(token), userID, time.Now().Add(30*24*time.Hour)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) UserForSession(ctx context.Context, token string) (model.User, error) {
	if token == "" {
		return model.User{}, ErrNotFound
	}
	id, err := s.repo.SessionUserID(ctx, tokenHash(token))
	if isNotFound(err) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, err
	}
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, tokenHash(token))
}

// --- profile & security ---

func (s *Service) GetUser(ctx context.Context, id string) (model.User, error) {
	u, err := s.repo.GetUserByID(ctx, id)
	if isNotFound(err) {
		return model.User{}, ErrNotFound
	}
	return u, err
}

func (s *Service) GetByUsername(ctx context.Context, username string) (model.User, error) {
	name, err := NormalizeUsername(username)
	if err != nil {
		return model.User{}, ErrNotFound
	}
	u, err := s.repo.GetUserByUsername(ctx, name)
	if isNotFound(err) {
		return model.User{}, ErrNotFound
	}
	return u, err
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, in model.User) (model.User, error) {
	in.ID = userID
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Bio = strings.TrimSpace(in.Bio)
	in.Location = strings.TrimSpace(in.Location)
	in.Website = strings.TrimSpace(in.Website)
	if in.DisplayName == "" || len([]rune(in.DisplayName)) > 60 {
		return model.User{}, invalid("Add a display name (up to 60 characters).")
	}
	if len([]rune(in.Bio)) > 500 || len([]rune(in.Location)) > 80 {
		return model.User{}, invalid("One of those profile fields is too long.")
	}
	if in.Website != "" {
		u, err := url.Parse(in.Website)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return model.User{}, invalid("Website must be an http(s) link.")
		}
	}
	return s.repo.UpdateProfile(ctx, in)
}

func (s *Service) Security(ctx context.Context, userID string) (model.AccountSecurity, error) {
	has, err := s.repo.HasPassword(ctx, userID)
	if err != nil {
		return model.AccountSecurity{}, err
	}
	creds, err := s.repo.ListCredentials(ctx, userID)
	if err != nil {
		return model.AccountSecurity{}, err
	}
	out := model.AccountSecurity{HasPassword: has, Passkeys: []model.Passkey{}}
	if _, created, err := s.repo.GetRecoveryKey(ctx, userID); err == nil {
		out.HasRecoveryKey = true
		out.RecoveryKeyCreatedAt = &created
	} else if !isNotFound(err) {
		return model.AccountSecurity{}, err
	}
	for _, c := range creds {
		out.Passkeys = append(out.Passkeys, model.Passkey{ID: c.RowID, Name: c.Name, CreatedAt: c.CreatedAt, LastUsedAt: c.LastUsedAt})
	}
	return out, nil
}

func (s *Service) SetPassword(ctx context.Context, userID, password string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := validatePassword(user.Username, password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.UpsertPassword(ctx, userID, hash)
}

func (s *Service) RemovePassword(ctx context.Context, userID string) error {
	creds, err := s.repo.ListCredentials(ctx, userID)
	if err != nil {
		return err
	}
	if len(creds) == 0 {
		return ErrLastSignInMethod
	}
	return s.repo.DeletePassword(ctx, userID)
}

func (s *Service) RemovePasskey(ctx context.Context, userID, passkeyID string) error {
	creds, err := s.repo.ListCredentials(ctx, userID)
	if err != nil {
		return err
	}
	has, err := s.repo.HasPassword(ctx, userID)
	if err != nil {
		return err
	}
	if len(creds) <= 1 && !has {
		return ErrLastSignInMethod
	}
	n, err := s.repo.DeleteCredential(ctx, passkeyID, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetAvatar records a new profile picture (blob stored by the caller).
func (s *Service) SetAvatar(ctx context.Context, userID, key, contentType string) error {
	return s.repo.SetAvatar(ctx, userID, key, contentType)
}

// StoreAvatar stores a downscaled copy of the picture — never the
// uploaded original — and records it as a JPEG. A picture no decoder can
// read is rejected rather than stored, so what is stored is always a
// small file that is safe to serve at any size.
func (s *Service) StoreAvatar(ctx context.Context, userID string, r io.Reader, _ string) error {
	if s.media == nil {
		return ErrNotFound
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	resized, err := resizedAvatarJPEG(ctx, data)
	if err != nil {
		return invalid("That picture could not be read — try a JPEG or PNG instead.")
	}
	key, _, err := s.media.PutHashed(ctx, bytes.NewReader(resized))
	if err != nil {
		return err
	}
	return s.repo.SetAvatar(ctx, userID, key, "image/jpeg")
}

// Avatar opens the user's profile picture, if they have one. Pictures
// stored before avatars were downscaled on upload are converted on
// first open and the converted copy is stored back, like post
// thumbnails; afterwards the small JPEG is served as-is. When a legacy
// picture cannot be converted (no decoder, no ffmpeg), the original is
// served so the avatar still renders.
func (s *Service) Avatar(ctx context.Context, userID string) (model.User, io.ReadCloser, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user.AvatarKey == "" || s.media == nil {
		if err == nil {
			err = ErrNotFound
		}
		return user, nil, err
	}
	file, err := s.media.Open(ctx, user.AvatarKey)
	if err != nil {
		return user, nil, err
	}
	data, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		return user, nil, err
	}
	if !avatarNeedsResize(data, user.AvatarType) {
		return user, io.NopCloser(bytes.NewReader(data)), nil
	}
	resized, err := resizedAvatarJPEG(ctx, data)
	if err != nil {
		return user, io.NopCloser(bytes.NewReader(data)), nil
	}
	user.AvatarType = "image/jpeg"
	if key, _, putErr := s.media.PutHashed(ctx, bytes.NewReader(resized)); putErr == nil {
		if setErr := s.repo.SetAvatar(ctx, userID, key, "image/jpeg"); setErr == nil {
			user.AvatarKey = key
		}
	}
	return user, io.NopCloser(bytes.NewReader(resized)), nil
}

var ErrAdultConfirmationRequired = errors.New("confirm that you are 18 or older to show NSFW posts")

// SetShowNSFW records the account's content preference. Turning NSFW
// on requires an explicit 18+ confirmation, every time it's enabled.
func (s *Service) SetShowNSFW(ctx context.Context, userID string, show, over18 bool) error {
	if show && !over18 {
		return ErrAdultConfirmationRequired
	}
	return s.repo.UpdateShowNSFW(ctx, userID, show)
}

// Appearance ids, mirrored in frontend/src/domain/appearance.ts.
var Themes = []string{"kura", "safebooru", "rule34", "umineko", "bernkastel", "lambdadelta", "beatrice", "erika", "battler", "virgilia", "catppuccin", "danbooru"}
var Fonts = []string{"system", "serif", "rounded", "mono", "hack"}

// SetAppearance records the account's default theme and font. Unknown
// ids are rejected (fail closed) rather than stored.
func (s *Service) SetAppearance(ctx context.Context, userID, theme, font string) error {
	if !contains(Themes, theme) {
		return invalid("Choose one of the available themes.")
	}
	if !contains(Fonts, font) {
		return invalid("Choose one of the available fonts.")
	}
	return s.repo.UpdateAppearance(ctx, userID, theme, font)
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// SetTagPreferences records the account's default board tag filters.
// Tags are normalized the booru way; a tag in both lists is excluded
// (exclusion wins), so the stored lists never overlap.
func (s *Service) SetTagPreferences(ctx context.Context, userID string, includeTags, excludeTags []string) error {
	includeTags = normalizeTagList(includeTags)
	excludeTags = normalizeTagList(excludeTags)
	excluded := make(map[string]bool, len(excludeTags))
	for _, t := range excludeTags {
		excluded[t] = true
	}
	kept := includeTags[:0]
	for _, t := range includeTags {
		if !excluded[t] {
			kept = append(kept, t)
		}
	}
	return s.repo.UpdateTagPreferences(ctx, userID, kept, excludeTags)
}

var tagCleanRE = regexp.MustCompile(`[^a-z0-9_-]+`)

// normalizeTagList mirrors board.NormalizeTags without importing the
// board service: split on spaces/commas, lowercase, underscores join.
func normalizeTagList(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range in {
		for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ',' || r == '\n'
		}) {
			tag := tagCleanRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(token)), "_")
			tag = strings.ReplaceAll(tag, "-", "_")
			tag = strings.Trim(tag, "_")
			if tag == "" || len(tag) > 40 || seen[tag] {
				continue
			}
			seen[tag] = true
			out = append(out, tag)
			if len(out) >= 30 {
				return out
			}
		}
	}
	return out
}

// RemoveAvatar clears the profile picture.
func (s *Service) RemoveAvatar(ctx context.Context, userID string) error {
	return s.repo.SetAvatar(ctx, userID, "", "")
}

// --- recovery keys ---

// issueRecoveryKey stores a fresh key for the user and returns the
// plaintext, which is shown exactly once and never stored.
func (s *Service) issueRecoveryKey(ctx context.Context, userID string) (string, error) {
	key, hash, err := GenerateRecoveryKey()
	if err != nil {
		return "", err
	}
	if err := s.repo.SetRecoveryKey(ctx, userID, hash); err != nil {
		return "", err
	}
	return key, nil
}

// RegenerateRecoveryKey replaces the signed-in user's key; the old key
// stops working immediately.
func (s *Service) RegenerateRecoveryKey(ctx context.Context, userID string) (string, error) {
	return s.issueRecoveryKey(ctx, userID)
}

// RecoverAccount is the last-resort sign-in: username + recovery key +
// a new password. It sets the password, revokes every passkey and
// session, rotates the recovery key, and returns a fresh session and
// the new key (shown once).
func (s *Service) RecoverAccount(ctx context.Context, username, recoveryKey, newPassword string) (model.User, string, string, error) {
	name, err := NormalizeUsername(username)
	if err != nil {
		return model.User{}, "", "", ErrInvalidCredentials
	}
	user, err := s.repo.GetUserByUsername(ctx, name)
	if isNotFound(err) {
		_ = CheckRecoveryKey(recoveryKey, RecoveryKeyHash("kura-unknown-user")) // equalize timing
		return model.User{}, "", "", ErrInvalidCredentials
	}
	if err != nil {
		return model.User{}, "", "", err
	}
	stored, _, err := s.repo.GetRecoveryKey(ctx, user.ID)
	if isNotFound(err) || !CheckRecoveryKey(recoveryKey, stored) {
		return model.User{}, "", "", ErrInvalidCredentials
	}
	if err := validatePassword(user.Username, newPassword); err != nil {
		return model.User{}, "", "", err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return model.User{}, "", "", err
	}
	key, keyHash, err := GenerateRecoveryKey()
	if err != nil {
		return model.User{}, "", "", err
	}
	if err := s.repo.RecoverAccount(ctx, user.ID, hash, keyHash); err != nil {
		return model.User{}, "", "", err
	}
	token, err := s.newSession(ctx, user.ID)
	return user, token, key, err
}

// --- passkeys (WebAuthn) ---

type waUser struct {
	user  model.User
	creds []model.StoredCredential
}

func (u *waUser) WebAuthnID() []byte {
	id, err := uuid.Parse(u.user.ID)
	if err != nil {
		return []byte(u.user.ID)
	}
	return id[:]
}
func (u *waUser) WebAuthnName() string        { return u.user.Username }
func (u *waUser) WebAuthnDisplayName() string { return u.user.DisplayName }
func (u *waUser) WebAuthnCredentials() []webauthn.Credential {
	out := make([]webauthn.Credential, 0, len(u.creds))
	for _, c := range u.creds {
		out = append(out, toWACredential(c))
	}
	return out
}

func toWACredential(c model.StoredCredential) webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, 0, len(c.Transports))
	for _, t := range c.Transports {
		transports = append(transports, protocol.AuthenticatorTransport(t))
	}
	return webauthn.Credential{
		ID: c.CredentialID, PublicKey: c.PublicKey, AttestationType: c.AttestationType, Transport: transports,
		Flags:         webauthn.CredentialFlags{UserPresent: c.UserPresent, UserVerified: c.UserVerified, BackupEligible: c.BackupEligible, BackupState: c.BackupState},
		Authenticator: webauthn.Authenticator{AAGUID: c.AAGUID, SignCount: uint32(c.SignCount)},
	}
}

func fromWACredential(c *webauthn.Credential, name string) model.StoredCredential {
	transports := make([]string, 0, len(c.Transport))
	for _, t := range c.Transport {
		transports = append(transports, string(t))
	}
	if name == "" {
		name = "Passkey"
	}
	return model.StoredCredential{
		CredentialID: c.ID, PublicKey: c.PublicKey, AttestationType: c.AttestationType, Transports: transports,
		UserPresent: c.Flags.UserPresent, UserVerified: c.Flags.UserVerified, BackupEligible: c.Flags.BackupEligible,
		BackupState: c.Flags.BackupState, AAGUID: c.Authenticator.AAGUID, SignCount: int64(c.Authenticator.SignCount), Name: name,
	}
}

func (s *Service) loadWAUser(ctx context.Context, user model.User) (*waUser, error) {
	creds, err := s.repo.ListCredentials(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &waUser{user: user, creds: creds}, nil
}

func (s *Service) storeCeremony(ctx context.Context, purpose string, user *model.User, session *webauthn.SessionData) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	var uid *string
	username := ""
	if user != nil {
		uid = &user.ID
		username = user.Username
	}
	return s.repo.CreateChallenge(ctx, purpose, username, uid, data, time.Now().Add(5*time.Minute))
}

func (s *Service) takeCeremony(ctx context.Context, ceremonyID string) (model.Challenge, webauthn.SessionData, error) {
	ch, err := s.repo.TakeChallenge(ctx, ceremonyID)
	if isNotFound(err) {
		return ch, webauthn.SessionData{}, ErrCeremonyExpired
	}
	if err != nil {
		return ch, webauthn.SessionData{}, err
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(ch.SessionData, &session); err != nil {
		return ch, session, ErrCeremonyExpired
	}
	return ch, session, nil
}

// BeginPasskeyRegistration serves both new accounts (current == nil) and
// adding a passkey to a signed-in account. requestOrigin is the browser
// Origin of the begin request ("" keeps the configured RPID).
func (s *Service) BeginPasskeyRegistration(ctx context.Context, current *model.User, username, display, requestOrigin string) (string, any, error) {
	rpID, origin, err := s.bindingForOrigin(requestOrigin)
	if err != nil {
		return "", nil, err
	}
	var wau *waUser
	var purpose string
	if current != nil {
		var err error
		wau, err = s.loadWAUser(ctx, *current)
		if err != nil {
			return "", nil, err
		}
		purpose = "add"
	} else {
		name, err := NormalizeUsername(username)
		if err != nil {
			return "", nil, err
		}
		if _, err := s.repo.GetUserByUsername(ctx, name); err == nil {
			return "", nil, ErrUsernameTaken
		} else if !isNotFound(err) {
			return "", nil, err
		}
		display = strings.TrimSpace(display)
		if display == "" {
			display = name
		}
		wau = &waUser{user: model.User{ID: uuid.NewString(), Username: name, DisplayName: display}}
		purpose = "register"
	}
	opts := []webauthn.RegistrationOption{}
	if len(wau.creds) > 0 {
		exclusions := make([]protocol.CredentialDescriptor, 0, len(wau.creds))
		for _, c := range wau.creds {
			exclusions = append(exclusions, protocol.CredentialDescriptor{Type: protocol.PublicKeyCredentialType, CredentialID: c.CredentialID})
		}
		opts = append(opts, webauthn.WithExclusions(exclusions))
	}
	opts = append(opts, webauthn.WithRegistrationRelyingPartyID(rpID))
	if origin != "" {
		opts = append(opts, webauthn.WithRegistrationOrigin(origin))
	}
	creation, session, err := s.wa.BeginRegistration(wau, opts...)
	if err != nil {
		return "", nil, err
	}
	ceremonyFor := &wau.user
	if purpose == "register" {
		ceremonyFor = &model.User{Username: wau.user.Username} // no row yet; keep username only
		ceremonyFor.ID = ""
		id, err := s.repo.CreateChallenge(ctx, purpose, wau.user.Username, nil, mustJSON(session), time.Now().Add(5*time.Minute))
		return id, creation, err
	}
	id, err := s.storeCeremony(ctx, purpose, ceremonyFor, session)
	return id, creation, err
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func (s *Service) FinishPasskeyRegistration(ctx context.Context, ceremonyID string, body []byte, passkeyName string) (model.User, string, string, error) {
	ch, session, err := s.takeCeremony(ctx, ceremonyID)
	if err != nil {
		return model.User{}, "", "", err
	}
	if ch.Purpose != "register" && ch.Purpose != "add" {
		return model.User{}, "", "", ErrCeremonyExpired
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(body))
	if err != nil {
		return model.User{}, "", "", invalid("That passkey could not be read. Please try again.")
	}
	if ch.Purpose == "add" {
		if ch.UserID == nil {
			return model.User{}, "", "", ErrCeremonyExpired
		}
		user, err := s.repo.GetUserByID(ctx, *ch.UserID)
		if err != nil {
			return model.User{}, "", "", err
		}
		wau, err := s.loadWAUser(ctx, user)
		if err != nil {
			return model.User{}, "", "", err
		}
		cred, err := s.wa.CreateCredential(wau, session, parsed)
		if err != nil {
			return model.User{}, "", "", invalid("That passkey could not be verified. Please try again.")
		}
		if passkeyName == "" {
			passkeyName = user.Username
		}
		if _, err := s.repo.InsertCredential(ctx, user.ID, fromWACredential(cred, passkeyName)); err != nil {
			return model.User{}, "", "", err
		}
		return user, "", "", nil
	}
	// New account: the pending user id lives in the ceremony session.
	pendingID := uuid.UUID(session.UserID).String()
	wau := &waUser{user: model.User{ID: pendingID, Username: ch.Username, DisplayName: ch.Username}}
	cred, err := s.wa.CreateCredential(wau, session, parsed)
	if err != nil {
		return model.User{}, "", "", invalid("That passkey could not be verified. Please try again.")
	}
	if passkeyName == "" {
		passkeyName = ch.Username
	}
	user, err := s.repo.CreateUserWithCredential(ctx, pendingID, ch.Username, ch.Username, fromWACredential(cred, passkeyName))
	if isUnique(err) {
		return model.User{}, "", "", ErrUsernameTaken
	}
	if err != nil {
		return model.User{}, "", "", err
	}
	key, err := s.issueRecoveryKey(ctx, user.ID)
	if err != nil {
		return model.User{}, "", "", err
	}
	token, err := s.newSession(ctx, user.ID)
	return user, token, key, err
}

// BeginPasskeyLogin starts a login ceremony. With a username it offers
// only that account's passkeys; with no username it starts a discoverable
// (usernameless) ceremony — the browser/password manager lists the saved
// passkeys for this site and the chosen one identifies the account.
func (s *Service) BeginPasskeyLogin(ctx context.Context, username, requestOrigin string) (string, any, error) {
	rpID, origin, err := s.bindingForOrigin(requestOrigin)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(username) == "" {
		loginOpts := []webauthn.LoginOption{webauthn.WithLoginRelyingPartyID(rpID)}
		if origin != "" {
			loginOpts = append(loginOpts, webauthn.WithLoginOrigin(origin))
		}
		assertion, session, err := s.wa.BeginDiscoverableLogin(loginOpts...)
		if err != nil {
			return "", nil, err
		}
		id, err := s.repo.CreateChallenge(ctx, "discoverable", "", nil, mustJSON(session), time.Now().Add(5*time.Minute))
		return id, assertion, err
	}
	name, err := NormalizeUsername(username)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	user, err := s.repo.GetUserByUsername(ctx, name)
	if isNotFound(err) {
		return "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, err
	}
	wau, err := s.loadWAUser(ctx, user)
	if err != nil {
		return "", nil, err
	}
	if len(wau.creds) == 0 {
		return "", nil, ErrInvalidCredentials
	}
	loginOpts := []webauthn.LoginOption{webauthn.WithLoginRelyingPartyID(rpID)}
	if origin != "" {
		loginOpts = append(loginOpts, webauthn.WithLoginOrigin(origin))
	}
	assertion, session, err := s.wa.BeginLogin(wau, loginOpts...)
	if err != nil {
		return "", nil, err
	}
	id, err := s.storeCeremony(ctx, "login", &user, session)
	return id, assertion, err
}

func (s *Service) FinishPasskeyLogin(ctx context.Context, ceremonyID string, body []byte) (model.User, string, error) {
	ch, session, err := s.takeCeremony(ctx, ceremonyID)
	if err != nil {
		return model.User{}, "", err
	}
	if ch.Purpose == "discoverable" {
		return s.finishDiscoverableLogin(ctx, session, body)
	}
	if ch.Purpose != "login" || ch.UserID == nil {
		return model.User{}, "", ErrCeremonyExpired
	}
	user, err := s.repo.GetUserByID(ctx, *ch.UserID)
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	wau, err := s.loadWAUser(ctx, user)
	if err != nil {
		return model.User{}, "", err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(body))
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	cred, err := s.wa.ValidateLogin(wau, session, parsed)
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	// Persist the new sign count for the credential that was used.
	for _, stored := range wau.creds {
		if bytes.Equal(stored.CredentialID, cred.ID) {
			_ = s.repo.TouchCredential(ctx, stored.RowID, int64(cred.Authenticator.SignCount))
		}
	}
	token, err := s.newSession(ctx, user.ID)
	return user, token, err
}

// finishDiscoverableLogin completes a usernameless ceremony: the
// credential's user handle (the account UUID from registration) tells
// us who is signing in, so no username was ever typed.
func (s *Service) finishDiscoverableLogin(ctx context.Context, session webauthn.SessionData, body []byte) (model.User, string, error) {
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(body))
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	var resolved model.User
	var resolvedWAU *waUser
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		id, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
		user, err := s.repo.GetUserByID(ctx, id.String())
		if err != nil {
			return nil, ErrInvalidCredentials
		}
		wau, err := s.loadWAUser(ctx, user)
		if err != nil {
			return nil, err
		}
		resolved, resolvedWAU = user, wau
		return wau, nil
	}
	cred, err := s.wa.ValidateDiscoverableLogin(handler, session, parsed)
	if err != nil {
		return model.User{}, "", ErrInvalidCredentials
	}
	if resolvedWAU != nil {
		for _, stored := range resolvedWAU.creds {
			if bytes.Equal(stored.CredentialID, cred.ID) {
				_ = s.repo.TouchCredential(ctx, stored.RowID, int64(cred.Authenticator.SignCount))
			}
		}
	}
	token, err := s.newSession(ctx, resolved.ID)
	return resolved, token, err
}
