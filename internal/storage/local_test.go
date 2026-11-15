package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOriginalRoundTripAndTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	key, err := store.Put(context.Background(), strings.NewReader("original bytes"))
	require.NoError(t, err)
	file, err := store.Open(context.Background(), key)
	require.NoError(t, err)
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, "original bytes", string(contents))
	_, err = store.Open(context.Background(), "../outside")
	require.Error(t, err)
}
func TestFailedWriteLeavesNoPublishedFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocal(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	_, err = store.Put(context.Background(), brokenReader{})
	require.Error(t, err)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, files)
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("source disconnected") }

type partialThenBroken struct{ left int }

func (r *partialThenBroken) Read(p []byte) (int, error) {
	if r.left > 0 {
		p[0] = 'x'
		r.left--
		return 1, nil
	}
	return 0, errors.New("source disconnected")
}

func TestPartialWriteLeavesNoPublishedFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocal(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	_, err = store.Put(context.Background(), &partialThenBroken{left: 4})
	require.Error(t, err)
	_, _, err = store.PutHashed(context.Background(), &partialThenBroken{left: 4})
	require.Error(t, err)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestPutDoesNotPublishUntilWriteFinishes(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocal(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	pr, pw := io.Pipe()
	wrote := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		finish()
		_ = pw.Close()
	})
	go func() {
		_, _ = pw.Write([]byte("hel"))
		close(wrote)
		<-release
		_, _ = pw.Write([]byte("lo"))
		_ = pw.Close()
	}()
	done := make(chan struct{})
	var key string
	var putErr error
	go func() {
		key, putErr = store.Put(ctx, pr)
		close(done)
	}()
	select {
	case <-wrote:
	case <-time.After(3 * time.Second):
		t.Fatal("upload was not being written")
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	for _, entry := range entries {
		require.Truef(t, strings.HasSuffix(entry.Name(), ".partial"), "published %s before the write finished", entry.Name())
	}
	finish()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("put did not finish")
	}
	require.NoError(t, putErr)
	file, err := store.Open(context.Background(), key)
	require.NoError(t, err)
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, "hello", string(contents))
	entries, err = os.ReadDir(dir)
	require.NoError(t, err)
	require.Equal(t, []string{key}, names(entries))
}

func TestPutPreservesOriginalBytes(t *testing.T) {
	payload := pngBytes(t)
	payload = append(payload, 0x00, 0xff, 0xd8, 0xff)
	dir := t.TempDir()
	store, err := NewLocal(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	key, err := store.Put(context.Background(), bytes.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, payload, readKey(t, store, key))

	sum := sha256.Sum256(payload)
	want := hex.EncodeToString(sum[:])
	hashed, sha, err := store.PutHashed(context.Background(), bytes.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, want, hashed)
	require.Equal(t, want, sha)
	require.Equal(t, payload, readKey(t, store, hashed))
	info, err := os.Stat(filepath.Join(dir, hashed))
	require.NoError(t, err)

	again, _, err := store.PutHashed(context.Background(), bytes.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, hashed, again)
	after, err := os.Stat(filepath.Join(dir, hashed))
	require.NoError(t, err)
	require.Equal(t, info.Size(), after.Size())
	require.True(t, info.ModTime().Equal(after.ModTime()))
	require.Equal(t, payload, readKey(t, store, hashed))
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func readKey(t *testing.T, store *Local, key string) []byte {
	t.Helper()
	file, err := store.Open(context.Background(), key)
	require.NoError(t, err)
	defer file.Close()
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	return contents
}

func names(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name()
	}
	return out
}

func TestPutHashedDedupesIdenticalBytes(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocal(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	key1, sha1, err := store.PutHashed(context.Background(), strings.NewReader("same bytes"))
	require.NoError(t, err)
	key2, sha2, err := store.PutHashed(context.Background(), strings.NewReader("same bytes"))
	require.NoError(t, err)
	require.Equal(t, key1, key2)
	require.Equal(t, sha1, sha2)
	require.Len(t, key1, 64)
	key3, _, err := store.PutHashed(context.Background(), strings.NewReader("different bytes"))
	require.NoError(t, err)
	require.NotEqual(t, key1, key3)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 2) // identical content stored exactly once
	file, err := store.Open(context.Background(), key1)
	require.NoError(t, err)
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, "same bytes", string(contents))
}
