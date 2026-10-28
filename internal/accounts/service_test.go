package accounts

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUsername(t *testing.T) {
	got, err := NormalizeUsername("  Astro_Chan ")
	require.NoError(t, err)
	require.Equal(t, "astro_chan", got)
	_, err = NormalizeUsername("ab")
	require.Error(t, err)
	_, err = NormalizeUsername("bad name!")
	require.Error(t, err)
}

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	require.NoError(t, err)
	require.True(t, CheckPassword("correct horse battery", hash))
	require.False(t, CheckPassword("wrong", hash))
	require.False(t, CheckPassword("correct horse battery", "not-a-hash"))
}

func TestRegisterWithPasswordRejectsShortPassword(t *testing.T) {
	svc, err := NewService(NewMockRepository(t), "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, _, err = svc.RegisterWithPassword(context.Background(), "new_user", "short", "")
	require.Error(t, err)
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
}

func TestRegisterRejectsCommonPassword(t *testing.T) {
	svc, err := NewService(NewMockRepository(t), "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, _, err = svc.RegisterWithPassword(context.Background(), "new_user", "iloveyouiloveyou", "")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
}

func TestPasswordPolicyBounds(t *testing.T) {
	require.Error(t, validatePassword("some_user", "only14chars123")) // 14 < 15
	require.NoError(t, validatePassword("some_user", "a memorable short sentence"))
	require.Error(t, validatePassword("some_user", "some_user plus padding!!")) // contains username
}

func TestNeedsRehashForWeakParams(t *testing.T) {
	weak, err := hashWithParams("correct horse battery", argonParams{memory: 8 * 1024, time: 1, threads: 1})
	require.NoError(t, err)
	require.True(t, CheckPassword("correct horse battery", weak))
	require.True(t, NeedsRehash(weak))
	strong, err := HashPassword("correct horse battery")
	require.NoError(t, err)
	require.False(t, NeedsRehash(strong))
	require.False(t, CheckPassword("correct horse battery", "$argon2id$v=19$m=999999999,t=2,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
}

func TestRecoveryKeyRoundTrip(t *testing.T) {
	key, hash, err := GenerateRecoveryKey()
	require.NoError(t, err)
	require.Len(t, normalizeRecoveryKey(key), 32)
	require.True(t, CheckRecoveryKey(key, hash))
	// Typing it back in a different case, without dashes, still works.
	require.True(t, CheckRecoveryKey(normalizeRecoveryKey(key), hash))
	require.False(t, CheckRecoveryKey("AAAA-AAAA-AAAA-AAAA-AAAA-AAAA-AAAA-AAAA", hash))
}

func TestRecoverAccountWrongKeyFails(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	_, hash, err := GenerateRecoveryKey()
	require.NoError(t, err)
	repo.EXPECT().GetRecoveryKey(mock.Anything, "u1").Return(hash, time.Now(), nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, _, err = svc.RecoverAccount(context.Background(), "some_user", "BBBB-BBBB-BBBB-BBBB-BBBB-BBBB-BBBB-BBBB", "a brand new passphrase")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRecoverAccountRotatesAndSignsIn(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	oldKey, oldHash, err := GenerateRecoveryKey()
	require.NoError(t, err)
	repo.EXPECT().GetRecoveryKey(mock.Anything, "u1").Return(oldHash, time.Now(), nil)
	repo.EXPECT().RecoverAccount(mock.Anything, "u1", mock.Anything, mock.Anything).Return(nil)
	repo.EXPECT().CreateSession(mock.Anything, mock.Anything, "u1", mock.Anything).Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	user, token, newKey, err := svc.RecoverAccount(ctx, "some_user", oldKey, "a brand new passphrase")
	require.NoError(t, err)
	require.Equal(t, "u1", user.ID)
	require.NotEmpty(t, token)
	require.NotEmpty(t, newKey)
	require.NotEqual(t, oldKey, newKey)
}

func TestLoginWrongPasswordIsInvalidCredentials(t *testing.T) {
	repo := NewMockRepository(t)
	hash, err := HashPassword("right-password")
	require.NoError(t, err)
	repo.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	repo.EXPECT().GetPasswordHash(mock.Anything, "u1").Return(hash, nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, err = svc.LoginWithPassword(context.Background(), "some_user", "wrong-password")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRemovePasswordBlockedAsLastMethod(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().ListCredentials(mock.Anything, "u1").Return([]model.StoredCredential{}, nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.ErrorIs(t, svc.RemovePassword(context.Background(), "u1"), ErrLastSignInMethod)
}

func TestRemovePasskeyBlockedAsLastMethod(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().ListCredentials(mock.Anything, "u1").Return([]model.StoredCredential{{RowID: "p1"}}, nil)
	repo.EXPECT().HasPassword(mock.Anything, "u1").Return(false, nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.ErrorIs(t, svc.RemovePasskey(context.Background(), "u1", "p1"), ErrLastSignInMethod)
}

func TestPasskeyLoginForPasswordOnlyAccountFailsCleanly(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "00000000-0000-0000-0000-000000000001", Username: "some_user"}, nil)
	repo.EXPECT().ListCredentials(mock.Anything, mock.Anything).Return([]model.StoredCredential{}, nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, err = svc.BeginPasskeyLogin(context.Background(), "some_user", "")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestBeginPasskeyRegistrationBindsRPIDToRequestOrigin(t *testing.T) {
	ctx := context.Background()
	// IP origin can never be an RPID: fail fast with localhost guidance
	// instead of issuing a ceremony the browser rejects as invalid domain.
	svcIP, err := NewService(NewMockRepository(t), "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, err = svcIP.BeginPasskeyRegistration(ctx, nil, "new_user", "", "http://127.0.0.1:5174")
	require.Error(t, err)
	require.Contains(t, err.Error(), "localhost:5174")

	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "new_user").Return(model.User{}, sql.ErrNoRows)
	repo.EXPECT().CreateChallenge(mock.Anything, "register", "new_user", mock.Anything, mock.Anything, mock.Anything).Return("ceremony-1", nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, creation, err := svc.BeginPasskeyRegistration(ctx, nil, "new_user", "", "http://localhost:5174")
	require.NoError(t, err)
	body, err := json.Marshal(creation)
	require.NoError(t, err)
	require.Contains(t, string(body), `"id":"localhost"`)
	// Default (no origin) keeps the configured RPID.
	repo2 := NewMockRepository(t)
	repo2.EXPECT().GetUserByUsername(mock.Anything, "new_user").Return(model.User{}, sql.ErrNoRows)
	repo2.EXPECT().CreateChallenge(mock.Anything, "register", "new_user", mock.Anything, mock.Anything, mock.Anything).Return("ceremony-2", nil)
	svc2, err := NewService(repo2, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, creation2, err := svc2.BeginPasskeyRegistration(ctx, nil, "new_user", "", "")
	require.NoError(t, err)
	body2, err := json.Marshal(creation2)
	require.NoError(t, err)
	require.Contains(t, string(body2), `"id":"localhost"`)
}

func TestBeginPasskeyLoginDiscoverableWithoutUsername(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().CreateChallenge(mock.Anything, "discoverable", "", mock.Anything, mock.Anything, mock.Anything).Return("ceremony-d", nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	id, assertion, err := svc.BeginPasskeyLogin(context.Background(), "", "http://localhost:5174")
	require.NoError(t, err)
	require.Equal(t, "ceremony-d", id)
	body, err := json.Marshal(assertion)
	require.NoError(t, err)
	// No allowlist: the authenticator lists its saved passkeys instead.
	require.NotContains(t, string(body), "allowCredentials")
	require.Contains(t, string(body), `"rpId":"localhost"`)
}

func TestBeginPasskeyRegistrationRequiresResidentKeyAndNamesUser(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "new_user").Return(model.User{}, sql.ErrNoRows)
	repo.EXPECT().CreateChallenge(mock.Anything, "register", "new_user", mock.Anything, mock.Anything, mock.Anything).Return("ceremony-1", nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, creation, err := svc.BeginPasskeyRegistration(context.Background(), nil, "new_user", "", "")
	require.NoError(t, err)
	body, err := json.Marshal(creation)
	require.NoError(t, err)
	require.Contains(t, string(body), `"residentKey":"required"`)
	require.Contains(t, string(body), `"name":"new_user"`)
}

func TestBeginPasskeyRejectsUnknownOrigin(t *testing.T) {
	svc, err := NewService(NewMockRepository(t), "http://localhost:5174", nil)
	require.NoError(t, err)
	_, _, err = svc.BeginPasskeyRegistration(context.Background(), nil, "new_user", "", "https://evil.example")
	require.Error(t, err)
	_, _, err = svc.BeginPasskeyLogin(context.Background(), "some_user", "https://evil.example")
	require.Error(t, err)
}

func TestUserForSessionNotFound(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().SessionUserID(mock.Anything, mock.Anything).Return("", sql.ErrNoRows)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	_, err = svc.UserForSession(context.Background(), "bogus")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSetAndRemoveAvatar(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository(t)
	repo.EXPECT().SetAvatar(ctx, "u1", "abc123", "image/png").Return(nil)
	repo.EXPECT().SetAvatar(ctx, "u1", "", "").Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NoError(t, svc.SetAvatar(ctx, "u1", "abc123", "image/png"))
	require.NoError(t, svc.RemoveAvatar(ctx, "u1"))
}

func TestSetTagPreferencesNormalizesAndExcludeWins(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository(t)
	// Booru normalization: spaces split, case/underscores fold, dupes
	// dropped, gif excluded even though it was also in the include list.
	repo.EXPECT().UpdateTagPreferences(ctx, "u1",
		[]string{"white_hair", "anime"},
		[]string{"gif"}).Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NoError(t, svc.SetTagPreferences(ctx, "u1",
		[]string{"White_Hair", "white_hair", "GIF", "anime"},
		[]string{"gif"}))
}

func TestSetAppearanceValidatesAndPersists(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository(t)
	repo.EXPECT().UpdateAppearance(ctx, "u1", "umineko", "serif").Return(nil)
	repo.EXPECT().UpdateAppearance(ctx, "u1", "catppuccin", "hack").Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NoError(t, svc.SetAppearance(ctx, "u1", "umineko", "serif"))
	require.NoError(t, svc.SetAppearance(ctx, "u1", "catppuccin", "hack"))
	for _, bad := range [][2]string{{"neon", "serif"}, {"umineko", "comic"}, {"", ""}} {
		err := svc.SetAppearance(ctx, "u1", bad[0], bad[1])
		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr, "theme=%q font=%q", bad[0], bad[1])
	}
}

func TestSetTagSuggestionRowsValidates(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository(t)
	repo.EXPECT().UpdateTagSuggestionRows(ctx, "u1", 0).Return(nil)
	repo.EXPECT().UpdateTagSuggestionRows(ctx, "u1", 10).Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NoError(t, svc.SetTagSuggestionRows(ctx, "u1", 0))
	require.NoError(t, svc.SetTagSuggestionRows(ctx, "u1", 10))
	for _, rows := range []int{-1, 11} {
		err := svc.SetTagSuggestionRows(ctx, "u1", rows)
		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
	}
}

func TestShowNSFWRequiresAdultConfirmation(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository(t)
	repo.EXPECT().UpdateShowNSFW(ctx, "u1", true).Return(nil)
	repo.EXPECT().UpdateShowNSFW(ctx, "u1", false).Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.ErrorIs(t, svc.SetShowNSFW(ctx, "u1", true, false), ErrAdultConfirmationRequired)
	require.NoError(t, svc.SetShowNSFW(ctx, "u1", true, true))
	require.NoError(t, svc.SetShowNSFW(ctx, "u1", false, false))
}
