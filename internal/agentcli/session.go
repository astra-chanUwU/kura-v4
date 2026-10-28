package agentcli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrExpiredSession means a stored token is no longer a live session.
// Commands must not silently continue as a visitor.
var ErrExpiredSession = errors.New("session expired")

// DefaultSessionPath is the operator session file. .local/ is gitignored.
func DefaultSessionPath() string {
	return filepath.Join(".local", "agent-session")
}

// LoadSession reads a previously stored session token.
// A missing file is os.ErrNotExist.
func LoadSession(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(b))
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return "", ErrExpiredSession
	}
	return token, nil
}

// SaveSession writes the token so only the current user can read it.
func SaveSession(path, token string) error {
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, "\r\n \t") {
		return errors.New("refusing to store an empty session")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-session-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(token + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		// Windows refuses to rename onto an existing file.
		if rmErr := os.Remove(path); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return err
		}
		if err := os.Rename(tmpName, path); err != nil {
			return err
		}
	}
	ok = true
	return nil
}

// ClearSession removes the local session file. A missing file is success.
func ClearSession(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
