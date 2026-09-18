package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := NewRenderer()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestRenderCodeBlock_Go(t *testing.T) {
	r := newTestRenderer(t)
	code := `package main
import "fmt"
func main() { fmt.Println("hello") }`
	out, err := r.RenderCodeBlock("go", code)
	require.NoError(t, err)
	require.NotEmpty(t, out)
	assert.NotEqual(t, code, StripANSI(out))
}

func TestRenderCodeBlock_NoLanguage(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.RenderCodeBlock("", "plain text")
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestRenderCodeBlock_Empty(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.RenderCodeBlock("", "")
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestRenderCodeBlock_JSON(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.RenderCodeBlock("json", `{"key": "value"}`)
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestRendererClose(t *testing.T) {
	r, err := NewRenderer()
	require.NoError(t, err)
	err = r.Close()
	require.NoError(t, err)
}

func TestRendererRenderMarkdown(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.RenderMarkdown("# Hello\nThis is *markdown*")
	require.NoError(t, err)
	require.NotEmpty(t, out)
	assert.Contains(t, StripANSI(out), "Hello")
}

func TestRendererRenderMarkdown_Empty(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.RenderMarkdown("")
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestRendererRenderMarkdown_Table(t *testing.T) {
	r := newTestRenderer(t)
	table := `| A | B |
|---|---|
| 1 | 2 |`
	out, err := r.RenderMarkdown(table)
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestRendererRenderMarkdown_CodeFence(t *testing.T) {
	r := newTestRenderer(t)
	fenced := "```go\nfunc main() {}\n```"
	out, err := r.RenderMarkdown(fenced)
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestStripANSI_All(t *testing.T) {
	assert.Equal(t, "hello", StripANSI("hello"))
	assert.Equal(t, "", StripANSI(""))
	assert.Equal(t, "Red", StripANSI("\x1b[31mRed\x1b[0m"))
	assert.Equal(t, "BoldText", StripANSI("\x1b[1mBold\x1b[0mText"))
}
