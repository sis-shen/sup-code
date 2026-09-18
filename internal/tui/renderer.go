package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// Renderer 负责 Markdown 文本的终端渲染
type Renderer struct {
	renderer *glamour.TermRenderer
}

// NewRenderer 创建终端 Markdown 渲染器
func NewRenderer() (*Renderer, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return nil, fmt.Errorf("create glamour renderer: %w", err)
	}

	return &Renderer{renderer: r}, nil
}

// RenderMarkdown 将 Markdown 文本渲染为终端 ANSI 字符串
func (r *Renderer) RenderMarkdown(input string) (string, error) {
	out, err := r.renderer.Render(input)
	if err != nil {
		return input, fmt.Errorf("render markdown: %w", err)
	}
	return out, nil
}

func (r *Renderer) RenderCodeBlock(language, code string) (string, error) {
	// Use glamour for code block rendering (it uses chroma internally)
	fenced := "```" + language + "\n" + code + "\n```"
	return r.RenderMarkdown(fenced)
}

// StripANSI 移除 ANSI 转义序列
func StripANSI(input string) string {
	var result strings.Builder
	inEscape := false
	for _, ch := range input {
		if ch == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(ch)
	}
	return result.String()
}

// Close 释放渲染器资源
func (r *Renderer) Close() error {
	return r.renderer.Close()
}
