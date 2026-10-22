package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/google/uuid"
)

type Store interface {
	Put(context.Context, io.Reader) (string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Local struct{ root *os.Root }

var validKey = regexp.MustCompile(`^[a-f0-9-]{36}$`)

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
func (s *Local) Put(ctx context.Context, input io.Reader) (string, error) {
	key := uuid.NewString()
	temp := key + ".partial"
	file, err := s.root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close(); _ = s.root.Remove(temp) }()
	_, err = io.Copy(file, contextReader{ctx: ctx, input: input})
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	// Link publishes without overwriting an existing original.
	if err := s.root.Link(temp, key); err != nil {
		return "", err
	}
	return key, nil
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
