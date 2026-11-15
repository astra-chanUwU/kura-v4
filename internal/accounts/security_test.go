package accounts

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetPasswordRevokesSessionsAndRotates(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByID(mock.Anything, "u1").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	repo.EXPECT().ReplacePassword(mock.Anything, "u1", mock.Anything).Return(nil)
	repo.EXPECT().CreateSession(mock.Anything, mock.Anything, "u1", mock.Anything).Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)

	token, err := svc.SetPassword(context.Background(), "u1", "a brand new passphrase")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLoginDoesNotRevealPasswordlessAccounts(t *testing.T) {
	_ = CheckPassword("warmup-password", dummyHash)

	unknown := NewMockRepository(t)
	unknown.EXPECT().GetUserByUsername(mock.Anything, "missing_user").Return(model.User{}, model.ErrNotFound)
	svcUnknown, err := NewService(unknown, "http://localhost:5174", nil)
	require.NoError(t, err)

	passwordless := NewMockRepository(t)
	passwordless.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	passwordless.EXPECT().GetPasswordHash(mock.Anything, "u1").Return("", model.ErrNotFound)
	svcKnown, err := NewService(passwordless, "http://localhost:5174", nil)
	require.NoError(t, err)

	start := time.Now()
	_, _, errUnknown := svcUnknown.LoginWithPassword(context.Background(), "missing_user", "not the password!!")
	unknownDur := time.Since(start)
	start = time.Now()
	_, _, errKnown := svcKnown.LoginWithPassword(context.Background(), "some_user", "not the password!!")
	knownDur := time.Since(start)

	require.ErrorIs(t, errUnknown, ErrInvalidCredentials)
	require.ErrorIs(t, errKnown, ErrInvalidCredentials)
	require.NotErrorIs(t, errKnown, model.ErrNotFound)
	require.Greater(t, unknownDur, 15*time.Millisecond)
	require.Greater(t, knownDur, 15*time.Millisecond)
	ratio := float64(unknownDur) / float64(knownDur)
	if ratio < 1 {
		ratio = 1 / ratio
	}
	require.Less(t, ratio, 4.0)
}

func TestLoginPasswordLookupErrorIsNotInvalidCredentials(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	repo.EXPECT().GetPasswordHash(mock.Anything, "u1").Return("", errors.New("db down"))
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)

	_, _, err = svc.LoginWithPassword(context.Background(), "some_user", "not the password!!")
	require.EqualError(t, err, "db down")
	require.NotErrorIs(t, err, ErrInvalidCredentials)
}

func TestRecoverAccountDatabaseErrorIsNotInvalidCredentials(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().GetUserByUsername(mock.Anything, "some_user").Return(model.User{ID: "u1", Username: "some_user"}, nil)
	repo.EXPECT().GetRecoveryKey(mock.Anything, "u1").Return(nil, time.Time{}, errors.New("db down"))
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)

	_, _, _, err = svc.RecoverAccount(context.Background(), "some_user", "BBBB-BBBB-BBBB-BBBB-BBBB-BBBB-BBBB-BBBB", "a brand new passphrase")
	require.EqualError(t, err, "db down")
	require.NotErrorIs(t, err, ErrInvalidCredentials)
}

func TestPasskeyConfigOriginsAreNotIPAddresses(t *testing.T) {
	svc, err := NewService(NewMockRepository(t), "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NotEmpty(t, svc.origins)
	for _, origin := range svc.origins {
		parsed, err := url.Parse(origin)
		require.NoError(t, err)
		require.Nil(t, net.ParseIP(parsed.Hostname()), origin)
	}
	_, _, err = svc.BeginPasskeyRegistration(context.Background(), nil, "new_user", "", "http://[::1]:5174")
	require.Error(t, err)
	_, _, err = svc.BeginPasskeyLogin(context.Background(), "", "http://0.0.0.0:5174")
	require.Error(t, err)
}

func TestUpdateProfileRejectsUnsafeWebsites(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().UpdateProfile(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, u model.User) (model.User, error) { return u, nil }).Twice()
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)

	long := "https://example.com/" + strings.Repeat("a", 181) // 201 runes
	for _, site := range []string{
		"javascript:alert(1)",
		"data:text/html,hi",
		long,
		"http://example.com/path with space",
		`https://example.com/foo\bar`,
		"http://example.com:99999",
		"http://.",
	} {
		_, err := svc.UpdateProfile(context.Background(), "u1", model.User{DisplayName: "Ada", Website: site})
		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr, "website %q", site)
	}

	ok, err := svc.UpdateProfile(context.Background(), "u1", model.User{DisplayName: "Ada", Website: "https://example.com/a"})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/a", ok.Website)
	exact := "https://example.com/" + strings.Repeat("b", 180) // 200 runes
	ok, err = svc.UpdateProfile(context.Background(), "u1", model.User{DisplayName: "Ada", Website: exact})
	require.NoError(t, err)
	require.Equal(t, exact, ok.Website)
}

func TestRemovePasswordAllowedWhenPasskeyRemains(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().DeletePassword(mock.Anything, "u1").Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NoError(t, svc.RemovePassword(context.Background(), "u1"))
}

func TestRemovePasskeyAllowedWhenAnotherRemains(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().DeleteCredential(mock.Anything, "p1", "u1").Return(int64(1), nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.NoError(t, svc.RemovePasskey(context.Background(), "u1", "p1"))
}

func TestRemoveMissingPasskeyIsNotLastMethod(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().DeleteCredential(mock.Anything, "missing", "u1").Return(int64(0), nil)
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)
	require.ErrorIs(t, svc.RemovePasskey(context.Background(), "u1", "missing"), model.ErrNotFound)
}

func TestSignInRemovalBlocked(t *testing.T) {
	require.True(t, repository.SignInRemovalBlocked(0, true, true, true))
	require.False(t, repository.SignInRemovalBlocked(1, true, true, true))
	require.True(t, repository.SignInRemovalBlocked(1, false, false, true))
	require.False(t, repository.SignInRemovalBlocked(1, true, false, true))
	require.False(t, repository.SignInRemovalBlocked(2, false, false, true))
	require.False(t, repository.SignInRemovalBlocked(1, false, false, false))
}

// lockedPasskeys is the repository's locked removal, without a database:
// the check and the delete happen together, and the service must not
// decide from a separate read.
type lockedPasskeys struct {
	*MockRepository
	mu    sync.Mutex
	creds map[string]struct{}
}

func (a *lockedPasskeys) DeleteCredential(_ context.Context, rowID, _ string) (int64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, found := a.creds[rowID]
	if repository.SignInRemovalBlocked(len(a.creds), false, false, found) {
		return 0, repository.ErrLastSignInMethod
	}
	if !found {
		return 0, nil
	}
	delete(a.creds, rowID)
	return 1, nil
}

func TestConcurrentPasskeyRemovalCannotLockOut(t *testing.T) {
	repo := &lockedPasskeys{
		MockRepository: NewMockRepository(t),
		creds:          map[string]struct{}{"p1": {}, "p2": {}},
	}
	svc, err := NewService(repo, "http://localhost:5174", nil)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []string{"p1", "p2"} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			errs[i] = svc.RemovePasskey(context.Background(), "u1", id)
		}(i, id)
	}
	wg.Wait()

	var removed, blocked int
	for _, err := range errs {
		switch {
		case err == nil:
			removed++
		case errors.Is(err, ErrLastSignInMethod):
			blocked++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	require.Equal(t, 1, removed)
	require.Equal(t, 1, blocked)
	require.Len(t, repo.creds, 1)
}
