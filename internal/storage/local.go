package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/google/uuid"
)

type Store interface {
	Put(context.Context, io.Reader) (string, error)
	// PutHashed stores content under its SHA-256 hash, so identical
	// bytes are stored exactly once no matter how often they arrive.
	PutHashed(context.Context, io.Reader) (key string, sha256Hex string, err error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Local struct{ root *os.Root }

// Keys are either legacy UUIDs or SHA-256 content hashes.
var validKey = regexp.MustCompile(`^([a-f0-9-]{36}|[a-f0-9]{64})$`)

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Local{root: root}, nil
}
func (s *Local) Close() error { return s.root.Close() }

// writeTemp streams input into a private temp file, optionally hashing
// as it goes, and returns the temp name (and hash, when requested).
func (s *Local) writeTemp(ctx context.Context, input io.Reader, hashed bool) (string, string, error) {
	temp := uuid.NewString() + ".partial"
	file, err := s.root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", "", err
	}
	// The caller publishes (links) the temp file and removes it after;
	// here we only clean up on failure paths.
	fail := func(err error) (string, string, error) {
		_ = file.Close()
		_ = s.root.Remove(temp)
		return "", "", err
	}
	var dst io.Writer = file
	h := sha256.New()
	if hashed {
		dst = io.MultiWriter(file, h)
	}
	if _, err := io.Copy(dst, contextReader{ctx: ctx, input: input}); err != nil {
		return fail(err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := file.Sync(); err != nil {
		return fail(err)
	}
	if err := file.Close(); err != nil {
		_ = s.root.Remove(temp)
		return "", "", err
	}
	return temp, hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Local) Put(ctx context.Context, input io.Reader) (string, error) {
	key := uuid.NewString()
	temp, _, err := s.writeTemp(ctx, input, false)
	if err != nil {
		return "", err
	}
	defer func() { _ = s.root.Remove(temp) }()
	// Link publishes without overwriting an existing original.
	if err := s.root.Link(temp, key); err != nil {
		return "", err
	}
	return key, nil
}

// PutHashed publishes content under its SHA-256 hash. Uploading the
// same bytes again returns the existing key without storing a copy.
func (s *Local) PutHashed(ctx context.Context, input io.Reader) (string, string, error) {
	temp, sha, err := s.writeTemp(ctx, input, true)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = s.root.Remove(temp) }()
	if _, err := s.root.Stat(sha); err == nil {
		return sha, sha, nil // identical bytes already stored
	}
	if err := s.root.Link(temp, sha); err != nil {
		if _, statErr := s.root.Stat(sha); statErr == nil {
			return sha, sha, nil // lost a concurrent race; content is there
		}
		return "", "", err
	}
	return sha, sha, nil
}
func (s *Local) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validKey.MatchString(key) {
		return nil, fmt.Errorf("invalid storage key")
	}
	return s.root.Open(key)
}
func (s *Local) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validKey.MatchString(key) {
		return fmt.Errorf("invalid storage key")
	}
	return s.root.Remove(key)
}

type contextReader struct {
	ctx   context.Context
	input io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.input.Read(p)
}
