package grep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/supcode/supcode/pkg"
)

const (
	defaultMaxMatches = 1000
	maxFileSize       = 100 * 1024 * 1024 // 100MB - skip binary-like large files
)

// GrepParams is the parsed parameters for Grep.
type GrepParams struct {
	Pattern        string  `json:"pattern"`
	Path           string  `json:"path"`
	PatternIsRegex *bool   `json:"pattern_is_regex,omitempty"`
	ContextLines   *int    `json:"context_lines,omitempty"`
	IncludePattern *string `json:"include_pattern,omitempty"`
	ExcludePattern *string `json:"exclude_pattern,omitempty"`
	MaxMatches     *int    `json:"max_matches,omitempty"`
}

var jsonSchema = json.RawMessage(`{
	"type": "object",
	"required": ["pattern", "path"],
	"additionalProperties": false,
	"properties": {
		"pattern": {
			"type": "string",
			"description": "The pattern to search for (regex or literal)"
		},
		"path": {
			"type": "string",
			"description": "File or directory path to search in"
		},
		"pattern_is_regex": {
			"type": "boolean",
			"description": "Whether pattern is a regex (default: true)"
		},
		"context_lines": {
			"type": "integer",
			"description": "Number of context lines before and after each match",
			"minimum": 0,
			"maximum": 100
		},
		"include_pattern": {
			"type": "string",
			"description": "Only search files matching this glob pattern"
		},
		"exclude_pattern": {
			"type": "string",
			"description": "Skip files matching this glob pattern"
		},
		"max_matches": {
			"type": "integer",
			"description": "Maximum number of matches (default: 1000)",
			"minimum": 1,
			"maximum": 100000
		}
	}
}`)

// MatchResult represents a single grep match.
type MatchResult struct {
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Content string   `json:"content"`
	Before  []string `json:"before,omitempty"`
	After   []string `json:"after,omitempty"`
}

// Tool implements pkg.Tool for searching file contents with regex.
type Tool struct{}

func (t *Tool) Name() string { return "grep" }
func (t *Tool) Description() string {
	return "Search file contents using regex patterns with concurrent scanning."
}
func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  jsonSchema,
	}
}

func (t *Tool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	var gp GrepParams
	if err := json.Unmarshal(params, &gp); err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}
	if gp.Pattern == "" {
		return pkg.ToolResult{
			Success: false,
			Error:   "pattern is required",
		}, nil
	}
	if gp.Path == "" {
		return pkg.ToolResult{
			Success: false,
			Error:   "path is required",
		}, nil
	}
	select {
	case <-ctx.Done():
		return pkg.ToolResult{Success: false, Error: "canceled"}, ctx.Err()
	default:
	}
	// Compile regex
	isRegex := true
	if gp.PatternIsRegex != nil {
		isRegex = *gp.PatternIsRegex
	}
	var re *regexp.Regexp
	var err error
	if isRegex {
		re, err = regexp.Compile(gp.Pattern)
	} else {
		re, err = regexp.Compile(regexp.QuoteMeta(gp.Pattern))
	}
	if err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid regex pattern: %v", err),
		}, nil
	}
	contextLines := 0
	if gp.ContextLines != nil {
		contextLines = *gp.ContextLines
	}
	maxMatches := defaultMaxMatches
	if gp.MaxMatches != nil && *gp.MaxMatches > 0 {
		maxMatches = *gp.MaxMatches
	}
	// Collect files to search
	var files []string
	info, err := os.Stat(gp.Path)
	if err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("path error: %v", err),
		}, nil
	}
	if !info.IsDir() {
		files = []string{gp.Path}
	} else {
		includePattern := ""
		if gp.IncludePattern != nil {
			includePattern = *gp.IncludePattern
		}
		excludePattern := ""
		if gp.ExcludePattern != nil {
			excludePattern = *gp.ExcludePattern
		}
		files, err = collectFiles(ctx, gp.Path, includePattern, excludePattern)
		if err != nil {
			return pkg.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("file collection failed: %v", err),
			}, nil
		}
	}
	if len(files) == 0 {
		return pkg.ToolResult{
			Success: true,
			Data:    json.RawMessage(`{"matches":[],"count":0}`),
		}, nil
	}
	// Search files concurrently
	results := grepFiles(ctx, files, re, contextLines, maxMatches)
	data, _ := json.Marshal(map[string]interface{}{
		"matches": results,
		"count":   len(results),
	})
	return pkg.ToolResult{
		Success: true,
		Data:    json.RawMessage(data),
	}, nil
}

// collectFiles walks a directory and returns matching files.
func collectFiles(ctx context.Context, root, includePattern, excludePattern string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip permission errors
		}
		if info.IsDir() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Skip binary files (check extension and size)
		if isBinaryFile(info) {
			return nil
		}
		if includePattern != "" {
			matched, err := filepath.Match(includePattern, filepath.Base(path))
			if err != nil || !matched {
				matched, err = filepath.Match(includePattern, path)
				if err != nil || !matched {
					return nil
				}
			}
		}
		if excludePattern != "" {
			matched, err := filepath.Match(excludePattern, filepath.Base(path))
			if err == nil && matched {
				return nil
			}
		}
		if info.Size() > maxFileSize {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// isBinaryFile checks if a file looks like a binary.
func isBinaryFile(info os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(info.Name()))
	switch ext {
	case ".exe", ".dll", ".so", ".dylib", ".bin", ".o", ".a", ".lib",
		".png", ".jpg", ".jpeg", ".gif", ".ico", ".bmp", ".webp",
		".mp3", ".mp4", ".avi", ".mov", ".wav", ".flac", ".ogg",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".ttf", ".otf", ".woff", ".woff2",
		".pyc", ".pyo", ".class", ".jar":
		return true
	}
	return false
}

// grepFiles searches files concurrently using a goroutine pool.
func grepFiles(ctx context.Context, files []string, re *regexp.Regexp, contextLines, maxMatches int) []MatchResult {
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > len(files) {
		numWorkers = len(files)
	}
	if numWorkers == 0 {
		return nil
	}
	type job struct {
		file string
	}
	type result struct {
		matches []MatchResult
		err     error
	}
	jobCh := make(chan job, len(files))
	resultCh := make(chan result, numWorkers)
	// Send jobs
	for _, f := range files {
		jobCh <- job{file: f}
	}
	close(jobCh)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				select {
				case <-ctx.Done():
					return
				default:
				}
				matches, err := grepFile(j.file, re, contextLines, maxMatches)
				if err != nil {
					resultCh <- result{err: err}
					continue
				}
				if len(matches) > 0 {
					resultCh <- result{matches: matches}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()
	var allResults []MatchResult
	totalMatches := 0
	for r := range resultCh {
		if r.err != nil {
			continue
		}
		allResults = append(allResults, r.matches...)
		totalMatches += len(r.matches)
		if totalMatches >= maxMatches {
			break
		}
	}
	if len(allResults) > maxMatches {
		allResults = allResults[:maxMatches]
	}
	return allResults
}

// grepFile searches a single file for matches.
func grepFile(path string, re *regexp.Regexp, contextLines, maxMatches int) ([]MatchResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Quick binary content check
	if bytes.Contains(data, []byte{0x00}) {
		return nil, nil
	}
	// Check if it looks like text
	nullCount := bytes.Count(data, []byte{0x00})
	if len(data) > 0 && float64(nullCount)/float64(len(data)) > 0.01 {
		return nil, nil
	}
	lines := strings.Split(string(data), "\n")
	var results []MatchResult
	for i, line := range lines {
		if re.MatchString(line) {
			m := MatchResult{
				File:    path,
				Line:    i + 1,
				Content: line,
			}
			if contextLines > 0 {
				beforeStart := i - contextLines
				if beforeStart < 0 {
					beforeStart = 0
				}
				for j := beforeStart; j < i; j++ {
					m.Before = append(m.Before, lines[j])
				}
				afterEnd := i + contextLines + 1
				if afterEnd > len(lines) {
					afterEnd = len(lines)
				}
				for j := i + 1; j < afterEnd; j++ {
					m.After = append(m.After, lines[j])
				}
			}
			results = append(results, m)
			if len(results) >= maxMatches {
				break
			}
		}
	}
	return results, nil
}
