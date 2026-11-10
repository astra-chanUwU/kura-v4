package board

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"strconv"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

const derivePostID = "11111111-1111-4111-8111-111111111111"

func TestDerivePostSecondRunDoesNotRewrite(t *testing.T) {
	original := pngBytes(64, 48)
	media := &memMedia{files: map[string][]byte{"orig": original}}
	svc, repo := newDeriveHarness(model.Post{
		ID: derivePostID, StorageKey: "orig", ContentType: "image/png",
	}, media)
	payload, err := json.Marshal(derivePayload{PostID: derivePostID})
	require.NoError(t, err)

	require.NoError(t, svc.DerivePost(context.Background(), payload))
	require.Equal(t, 2, media.puts)
	require.Equal(t, 1, repo.thumbSets)
	require.Equal(t, 1, repo.sampleSets)
	require.Equal(t, 1, repo.dimSets)
	require.NotEmpty(t, repo.post.ThumbnailKey)
	require.NotEmpty(t, repo.post.SampleKey)
	require.Equal(t, 64, repo.post.Width)
	require.Equal(t, 48, repo.post.Height)
	thumbKey, sampleKey := repo.post.ThumbnailKey, repo.post.SampleKey
	thumbBytes := append([]byte(nil), media.files[thumbKey]...)
	sampleBytes := append([]byte(nil), media.files[sampleKey]...)
	wantSample, err := sampleImageJPEG(bytes.NewReader(original))
	require.NoError(t, err)
	require.Equal(t, wantSample, sampleBytes)
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(sampleBytes))
	require.NoError(t, err)
	require.LessOrEqual(t, cfg.Width, sampleMaxEdge)
	require.LessOrEqual(t, cfg.Height, sampleMaxEdge)
	opens, puts := media.opens, media.puts

	require.NoError(t, svc.DerivePost(context.Background(), payload))
	require.Equal(t, puts, media.puts)
	require.Equal(t, opens, media.opens)
	require.Equal(t, 1, repo.thumbSets)
	require.Equal(t, 1, repo.sampleSets)
	require.Equal(t, 1, repo.dimSets)
	require.Equal(t, thumbKey, repo.post.ThumbnailKey)
	require.Equal(t, sampleKey, repo.post.SampleKey)
	require.Equal(t, thumbBytes, media.files[thumbKey])
	require.Equal(t, sampleBytes, media.files[sampleKey])
}

func TestDerivePostLeavesExistingThumbnailAndSample(t *testing.T) {
	media := &memMedia{files: map[string][]byte{
		"orig":            pngBytes(32, 32),
		"thumb-existing":  []byte("keep-thumb"),
		"sample-existing": []byte("keep-sample"),
	}}
	svc, repo := newDeriveHarness(model.Post{
		ID: derivePostID, StorageKey: "orig", ContentType: "image/png",
		ThumbnailKey: "thumb-existing", SampleKey: "sample-existing", Width: 32, Height: 32,
	}, media)
	payload, err := json.Marshal(derivePayload{PostID: derivePostID})
	require.NoError(t, err)

	require.NoError(t, svc.DerivePost(context.Background(), payload))
	require.NoError(t, svc.DerivePost(context.Background(), payload))
	require.Zero(t, media.puts)
	require.Zero(t, media.opens)
	require.Zero(t, repo.thumbSets)
	require.Zero(t, repo.sampleSets)
	require.Zero(t, repo.dimSets)
	require.Equal(t, []byte("keep-thumb"), media.files["thumb-existing"])
	require.Equal(t, []byte("keep-sample"), media.files["sample-existing"])
}

func TestDerivePostThumbnailStorageErrorRetries(t *testing.T) {
	media := &memMedia{files: map[string][]byte{"orig": pngBytes(16, 16)}, putErr: errors.New("disk full")}
	svc, repo := newDeriveHarness(model.Post{
		ID: derivePostID, StorageKey: "orig", ContentType: "image/png", Width: 16, Height: 16,
	}, media)
	payload, err := json.Marshal(derivePayload{PostID: derivePostID})
	require.NoError(t, err)
	require.Error(t, svc.DerivePost(context.Background(), payload))
	require.Empty(t, repo.post.ThumbnailKey)
	require.Empty(t, repo.post.SampleKey)
	require.Zero(t, repo.sampleSets)
}

func TestDerivePostSampleStorageErrorRetries(t *testing.T) {
	media := &memMedia{files: map[string][]byte{"orig": pngBytes(16, 16)}, putErr: errors.New("disk full")}
	svc, repo := newDeriveHarness(model.Post{
		ID: derivePostID, StorageKey: "orig", ContentType: "image/png",
		ThumbnailKey: "thumb-existing", Width: 16, Height: 16,
	}, media)
	payload, err := json.Marshal(derivePayload{PostID: derivePostID})
	require.NoError(t, err)
	require.Error(t, svc.DerivePost(context.Background(), payload))
	require.Equal(t, "thumb-existing", repo.post.ThumbnailKey)
	require.Empty(t, repo.post.SampleKey)
	require.Zero(t, media.puts)
}

func TestDerivePostCompletesWhenSampleIsImpossible(t *testing.T) {
	prev := grabStill
	grabStill = func(context.Context, io.Reader) (image.Image, error) {
		return nil, errors.New("no frame")
	}
	t.Cleanup(func() { grabStill = prev })
	media := &memMedia{files: map[string][]byte{"orig": []byte("not a video")}}
	svc, repo := newDeriveHarness(model.Post{
		ID: derivePostID, StorageKey: "orig", ContentType: "video/mp4",
		ThumbnailKey: "thumb-existing", Width: 320, Height: 240,
	}, media)
	payload, err := json.Marshal(derivePayload{PostID: derivePostID})
	require.NoError(t, err)
	require.NoError(t, svc.DerivePost(context.Background(), payload))
	require.Empty(t, repo.post.SampleKey)
	require.Zero(t, media.puts)
	require.Equal(t, "thumb-existing", repo.post.ThumbnailKey)
}

func TestDerivePostHonorsCanceledContext(t *testing.T) {
	media := &memMedia{files: map[string][]byte{"orig": pngBytes(8, 8)}}
	svc, repo := newDeriveHarness(model.Post{
		ID: derivePostID, StorageKey: "orig", ContentType: "image/png",
	}, media)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	payload, err := json.Marshal(derivePayload{PostID: derivePostID})
	require.NoError(t, err)
	require.ErrorIs(t, svc.DerivePost(ctx, payload), context.Canceled)
	require.Zero(t, media.puts)
	require.Empty(t, repo.post.ThumbnailKey)
	require.Empty(t, repo.post.SampleKey)
}

func TestCreateEnqueuesDeriveOnce(t *testing.T) {
	repo := &fakeRepo{}
	q := &fakeQueue{}
	var logs bytes.Buffer
	svc := NewService(repo, nil)
	svc.SetJobQueue(q, slog.New(slog.NewTextHandler(&logs, nil)))

	created, err := svc.Create(context.Background(), model.Post{}, "sha")
	require.NoError(t, err)
	require.Len(t, q.kinds, 1)
	require.Equal(t, KindPostDerive, q.kinds[0])
	var body derivePayload
	require.NoError(t, json.Unmarshal(q.payloads[0], &body))
	require.Equal(t, created.ID, body.PostID)
	require.JSONEq(t, `{"post_id":"`+created.ID+`"}`, string(q.payloads[0]))

	_, err = svc.Create(context.Background(), model.Post{SourceURL: "javascript:alert(1)"}, "sha")
	require.Error(t, err)
	require.Len(t, q.kinds, 1)
}

func TestCreateEnqueueFailureDoesNotFailCreate(t *testing.T) {
	repo := &fakeRepo{}
	q := &fakeQueue{err: errors.New("db down")}
	var logs bytes.Buffer
	svc := NewService(repo, nil)
	svc.SetJobQueue(q, slog.New(slog.NewTextHandler(&logs, nil)))
	created, err := svc.Create(context.Background(), model.Post{}, "sha")
	require.NoError(t, err)
	require.Equal(t, "created", created.ID)
	require.Contains(t, logs.String(), "enqueue post.derive")
	require.Contains(t, logs.String(), "db down")
}

type deriveRepo struct {
	fakeRepo
	thumbSets  int
	sampleSets int
	dimSets    int
}

func (d *deriveRepo) SetPostThumbnail(_ context.Context, id, key string) error {
	if id != d.post.ID {
		return errors.New("missing post")
	}
	d.thumbSets++
	d.post.ThumbnailKey = key
	return nil
}

func (d *deriveRepo) SetPostSample(_ context.Context, id, key string) error {
	if id != d.post.ID {
		return errors.New("missing post")
	}
	d.sampleSets++
	d.post.SampleKey = key
	return nil
}

func (d *deriveRepo) SetPostDimensions(_ context.Context, id string, width, height int) error {
	if id != d.post.ID {
		return errors.New("missing post")
	}
	d.dimSets++
	d.post.Width = width
	d.post.Height = height
	return nil
}

func newDeriveHarness(post model.Post, media *memMedia) (*Service, *deriveRepo) {
	repo := &deriveRepo{fakeRepo: fakeRepo{post: post}}
	svc := NewService(repo, media)
	svc.SetJobQueue(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, repo
}

type memMedia struct {
	files  map[string][]byte
	puts   int
	opens  int
	putErr error
}

func (m *memMedia) Put(ctx context.Context, r io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if m.putErr != nil {
		return "", m.putErr
	}
	m.puts++
	key := "blob-" + strconv.Itoa(m.puts)
	stored := append([]byte(nil), buf...)
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[key] = stored
	return key, nil
}

func (m *memMedia) PutHashed(context.Context, io.Reader) (string, string, error) {
	return "", "", errors.New("put hashed not used")
}

func (m *memMedia) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.opens++
	b, ok := m.files[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), b...))), nil
}

func (m *memMedia) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(m.files, key)
	return nil
}

type fakeQueue struct {
	err      error
	kinds    []string
	payloads []json.RawMessage
}

func (f *fakeQueue) Enqueue(_ context.Context, kind string, payload json.RawMessage) (int64, error) {
	f.kinds = append(f.kinds, kind)
	f.payloads = append(f.payloads, append(json.RawMessage(nil), payload...))
	if f.err != nil {
		return 0, f.err
	}
	return int64(len(f.kinds)), nil
}
