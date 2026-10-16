package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dangnmh/core-kit/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJSONConfig struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestLoadJSON_ValidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"test","value":42}`), 0o600))

	cfg, err := config.LoadJSON[testJSONConfig](path)
	require.NoError(t, err)
	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, 42, cfg.Value)
}

func TestLoadJSON_JSONC(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.jsonc")
	data := `{
		// This is a comment
		"name": "jsonc",
		"value": 99,
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))

	cfg, err := config.LoadJSON[testJSONConfig](path)
	require.NoError(t, err)
	assert.Equal(t, "jsonc", cfg.Name)
	assert.Equal(t, 99, cfg.Value)
}

func TestLoadJSON_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := config.LoadJSON[testJSONConfig]("/nonexistent/file.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoadJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{not valid`), 0o600))

	_, err := config.LoadJSON[testJSONConfig](path)
	assert.Error(t, err)
}

func TestLoadJSON_EmptyJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o600))

	cfg, err := config.LoadJSON[testJSONConfig](path)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Name)
	assert.Equal(t, 0, cfg.Value)
}

func TestLoadJSON_InvalidSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad_schema.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"test","value":"not_an_int"}`), 0o600))

	_, err := config.LoadJSON[testJSONConfig](path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}
