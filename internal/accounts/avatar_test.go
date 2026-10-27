package accounts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fakeMedia is a content-addressed in-memory Media, mirroring storage.Local.
type fakeMedia struct{ blobs map[string][]byte }

func newFakeMedia() *fakeMedia { return &fakeMedia{blobs: map[string][]byte{}} }

func (f *fakeMedia) PutHashed(_ context.Context, r io.Reader) (string, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	key := hex.EncodeToString(sum[:])
	f.blobs[key] = data
	return key, key, nil
}

func (f *fakeMedia) Open(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := f.blobs[key]
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func noisyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7 % 251), G: uint8(y * 11 % 241), B: uint8((x + y) * 13 % 239), A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func decodeDims(t *testing.T, data []byte) (string, int, int) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	return format, cfg.Width, cfg.Height
}

func TestStoreAvatarDownscalesToSmallJPEG(t *testing.T) {
	ctx := context.Background()
	media := newFakeMedia()
	repo := NewMockRepository(t)
	var storedKey string
	repo.EXPECT().SetAvatar(ctx, "u1", mock.Anything, "image/jpeg").
		Run(func(_ context.Context, _ string, key string, _ string) { storedKey = key }).Return(nil)
	svc, err := NewService(repo, "http://localhost:5174", media)
	require.NoError(t, err)

	src := noisyPNG(t, 1024, 768)
	require.NoError(t, svc.StoreAvatar(ctx, "u1", bytes.NewReader(src), "image/png"))

	require.NotEmpty(t, storedKey)
	stored := media.blobs[storedKey]
	format, w, h := decodeDims(t, stored)
	require.Equal(t, "jpeg", format)
	require.Equal(t, avatarMaxDim, w)
	require.Equal(t, 192, h)
	require.Less(t, len(stored), len(src), "downscaled JPEG must be smaller than the oversized source")
}

func TestStoreAvatarRejectsUndecodableImage(t *testing.T) {
	ctx := context.Background()
	media := newFakeMedia()
	repo := NewMockRepository(t) // no SetAvatar expectation: storing must not happen
	svc, err := NewService(repo, "http://localhost:5174", media)
	require.NoError(t, err)

	err = svc.StoreAvatar(ctx, "u1", bytes.NewReader([]byte("definitely not an image")), "image/png")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Empty(t, media.blobs)
}

func TestAvatarConvertsLegacyOversizedPictureOnce(t *testing.T) {
	ctx := context.Background()
	media := newFakeMedia()
	legacy := noisyPNG(t, 1024, 768)
	sum := sha256.Sum256(legacy)
	legacyKey := hex.EncodeToString(sum[:])
	media.blobs[legacyKey] = legacy

	repo := NewMockRepository(t)
	legacyUser := model.User{ID: "u1", Username: "ana", AvatarKey: legacyKey, AvatarType: "image/png"}
	var convertedKey string
	repo.EXPECT().GetUserByID(ctx, "u1").Return(legacyUser, nil).Once()
	repo.EXPECT().SetAvatar(ctx, "u1", mock.Anything, "image/jpeg").
		Run(func(_ context.Context, _ string, key string, _ string) { convertedKey = key }).Return(nil).Once()
	svc, err := NewService(repo, "http://localhost:5174", media)
	require.NoError(t, err)

	user, file, err := svc.Avatar(ctx, "u1")
	require.NoError(t, err)
	served, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, "image/jpeg", user.AvatarType)
	format, w, _ := decodeDims(t, served)
	require.Equal(t, "jpeg", format)
	require.Equal(t, avatarMaxDim, w)
	require.NotEmpty(t, convertedKey)

	// Second open: the stored picture is already a small JPEG, so it is
	// served as-is with no further write.
	repo.EXPECT().GetUserByID(ctx, "u1").
		Return(model.User{ID: "u1", Username: "ana", AvatarKey: convertedKey, AvatarType: "image/jpeg"}, nil).Once()
	_, file, err = svc.Avatar(ctx, "u1")
	require.NoError(t, err)
	again, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, served, again)
}
