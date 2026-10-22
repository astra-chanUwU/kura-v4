package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/site"
	"github.com/stretchr/testify/require"
)

type siteStub struct{ err error }

func (r siteStub) Get(context.Context) (model.Site, error) {
	return model.Site{Name: "Library", Description: "Personal archive"}, r.err
}
func TestHTTPContractAndSPAFallback(t *testing.T) {
	assets := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Kura</title>")}}
	app := NewHTTP(site.NewService(siteStub{}), func(context.Context) error { return nil }, assets, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/site", nil))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, map[string]any{"name": "Library", "description": "Personal archive"}, body)
	for _, test := range []struct {
		path   string
		status int
	}{{"/about", 200}, {"/api/v1/missing", 404}, {"/assets/missing.js", 404}, {"/media/private", 404}} {
		t.Run(test.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest("GET", test.path, nil))
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, test.status, resp.StatusCode)
		})
	}
}
func TestInternalErrorsDoNotExposeDatabaseDetails(t *testing.T) {
	app := NewHTTP(site.NewService(siteStub{err: errors.New("private connection details")}), func(context.Context) error { return nil }, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/site", nil))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, 500, response.StatusCode)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NotContains(t, string(data), "private connection details")
	var body map[string]string
	require.NoError(t, json.Unmarshal(data, &body))
	require.NotEmpty(t, body["request_id"])
	require.Equal(t, "internal_error", body["code"])
}
func TestReadinessFailsWhileLivenessRemainsAvailable(t *testing.T) {
	app := NewHTTP(site.NewService(siteStub{}), func(context.Context) error { return errors.New("offline") }, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, test := range []struct {
		path   string
		status int
	}{{"/health/ready", 503}, {"/health/live", 200}} {
		resp, err := app.Test(httptest.NewRequest("GET", test.path, nil))
		require.NoError(t, err)
		require.Equal(t, test.status, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	}
}
