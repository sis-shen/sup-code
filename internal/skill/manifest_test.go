package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidManifest(t *testing.T) {
	data := []byte(`{
		"name": "code-review",
		"version": "1.0.0",
		"description": "Review code changes"
	}`)
	m, err := ParseManifest(data)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "code-review", m.Name)
	assert.Equal(t, "1.0.0", m.Version)
	assert.Equal(t, "Review code changes", m.Description)
}

func TestParseInvalidName(t *testing.T) {
	data := []byte(`{"name": "Code Review!", "version": "1.0.0", "description": "test"}`)
	_, err := ParseManifest(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestParseEmptyName(t *testing.T) {
	data := []byte(`{"name": "", "version": "1.0.0", "description": "test"}`)
	_, err := ParseManifest(data)
	assert.Error(t, err)
}

func TestParseMissingVersion(t *testing.T) {
	data := []byte(`{"name": "test", "version": "", "description": "test"}`)
	_, err := ParseManifest(data)
	assert.Error(t, err)
}

func TestParseInvalidVersion(t *testing.T) {
	data := []byte(`{"name": "test", "version": "abc", "description": "test"}`)
	_, err := ParseManifest(data)
	assert.Error(t, err)
}

func TestParseMissingDescription(t *testing.T) {
	data := []byte(`{"name": "test", "version": "1.0.0", "description": ""}`)
	_, err := ParseManifest(data)
	assert.Error(t, err)
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := ParseManifest([]byte(`{invalid`))
	assert.Error(t, err)
}

func TestParseManifestWithTools(t *testing.T) {
	data := []byte(`{"name": "test", "version": "1.0.0", "description": "test", "tools": ["bash", "readfile"]}`)
	m, err := ParseManifest(data)
	require.NoError(t, err)
	assert.Equal(t, []string{"bash", "readfile"}, m.Tools)
}

func TestParseManifestWithAuthor(t *testing.T) {
	data := []byte(`{"name": "test", "version": "1.0.0", "description": "test", "author": "me"}`)
	m, err := ParseManifest(data)
	require.NoError(t, err)
	assert.Equal(t, "me", m.Author)
}

func TestLoadManifestFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.json")
	os.WriteFile(path, []byte(`{"name": "test", "version": "1.0.0", "description": "test"}`), 0644)
	m, err := LoadManifestFile(path)
	require.NoError(t, err)
	assert.Equal(t, "test", m.Name)
}

func TestLoadManifestFileNotFound(t *testing.T) {
	_, err := LoadManifestFile("/nonexistent/skill.json")
	assert.Error(t, err)
}

func TestIsValidSemver(t *testing.T) {
	assert.True(t, isValidSemver("1.0.0"))
	assert.True(t, isValidSemver("0.1.0"))
	assert.True(t, isValidSemver("10.20.30"))
	assert.False(t, isValidSemver(""))
	assert.False(t, isValidSemver("abc"))
	assert.False(t, isValidSemver("1"))
}

func TestParseManifestWithRequires(t *testing.T) {
	data := []byte(`{"name": "test", "version": "1.0.0", "description": "test", "requires": ["node"]}`)
	m, err := ParseManifest(data)
	require.NoError(t, err)
	assert.Equal(t, []string{"node"}, m.Requires)
}
