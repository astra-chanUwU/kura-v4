package app

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/stretchr/testify/require"
)

func TestAgentSchemaSkipsDatabase(t *testing.T) {
	var buf bytes.Buffer
	code := Agent(context.Background(), config.Config{}, []string{"schema"}, &buf)
	require.Equal(t, 0, code)
	require.Contains(t, buf.String(), "posts.upload")
	require.Contains(t, buf.String(), `"ok":true`)
}

func TestAgentUsageSkipsDatabase(t *testing.T) {
	var buf bytes.Buffer
	code := Agent(context.Background(), config.Config{}, nil, &buf)
	require.Equal(t, 2, code)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &body))
	require.Equal(t, "usage", body.Error.Code)
}
