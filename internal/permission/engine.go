package permission

import (
	"context"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/supcode/supcode/pkg"
)

// Engine implements the pkg.PermissionEngine interface.
type Engine struct {
	mu    sync.RWMutex
	rules []pkg.PermissionRule

	// Layer 3: Write confirmation
	allowedDirs    []string
	confirmedPaths map[string]bool
	confirmedMu    sync.Mutex

	// Layer 4: First use tracking
	tracker *FirstUseTracker

	// Layer 5: Audit logging
	auditor *AuditLogger
}

// New creates a PermissionEngine with built-in default rules.
func New() *Engine {
	e := &Engine{
		rules:          DefaultRules(),
		confirmedPaths: make(map[string]bool),
	}
	return e
}

// NewWithRules creates a PermissionEngine with a custom set of rules.
func NewWithRules(rules []pkg.PermissionRule) *Engine {
	e := &Engine{
		rules:          rules,
		confirmedPaths: make(map[string]bool),
	}
	e.sortRules()
	return e
}

// SetFirstUseTracker attaches a first-use tracker.
func (e *Engine) SetFirstUseTracker(t *FirstUseTracker) {
	e.tracker = t
}

// SetAuditLogger attaches an audit logger.
func (e *Engine) SetAuditLogger(a *AuditLogger) {
	e.auditor = a
}

// FirstUseTracker returns the attached first-use tracker, if any.
func (e *Engine) FirstUseTracker() *FirstUseTracker {
	return e.tracker
}

// AuditLogger returns the attached audit logger, if any.
func (e *Engine) AuditLogger() *AuditLogger {
	return e.auditor
}

// Check evaluates an action against the rule set and layered security policies.
// 1. Layer 1-2: Built-in rules (command blacklist, path protection)
// 2. Layer 3: Write confirmation (file_write / file_delete)
// 3. Layer 4: First-use tracking (tool / mcp)
// Returns DecisionAllow if no policy applies.
func (e *Engine) Check(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	if action.Target == "" {
		return pkg.DecisionAllow, nil
	}

	e.mu.RLock()
	// Layer 1-2: Rule matching
	for _, rule := range e.rules {
		if matches(rule, action) {
			e.mu.RUnlock()
			return rule.Decision, nil
		}
	}
	e.mu.RUnlock()

	// Layer 3: Write confirmation for file operations
	if action.Type == "file_write" || action.Type == "file_delete" {
		return e.checkWriteConfirm(action)
	}

	// Layer 4: First-use tracking for tools and MCP
	if action.ToolName != "" && e.tracker != nil {
		dec, err := e.tracker.CheckTool(ctx, action)
		if err != nil {
			return pkg.DecisionAllow, err
		}
		if dec != pkg.DecisionAllow {
			return dec, nil
		}
	}

	return pkg.DecisionAllow, nil
}

// checkWriteConfirm handles Layer 3 write confirmation logic.
func (e *Engine) checkWriteConfirm(action pkg.Action) (pkg.Decision, error) {
	path := action.Target

	// Check session-level confirmation cache
	e.confirmedMu.Lock()
	if e.confirmedPaths[path] {
		e.confirmedMu.Unlock()
		return pkg.DecisionAllow, nil
	}
	e.confirmedMu.Unlock()

	// Check project directory whitelist
	for _, dir := range e.allowedDirs {
		if strings.HasPrefix(path, dir) {
			return pkg.DecisionAllow, nil
		}
	}

	// Not in any of the exclusions, require confirmation
	return pkg.DecisionAsk, nil
}

// ConfirmPath marks a path as confirmed for the current session.
func (e *Engine) ConfirmPath(path string) {
	e.confirmedMu.Lock()
	defer e.confirmedMu.Unlock()
	e.confirmedPaths[path] = true
}

// AddAllowedDir adds a project directory to the write confirmation whitelist.
func (e *Engine) AddAllowedDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowedDirs = append(e.allowedDirs, dir)
}

// SetAllowedDirs sets the full list of allowed project directories.
func (e *Engine) SetAllowedDirs(dirs []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowedDirs = append([]string{}, dirs...)
}

// matches checks whether a rule applies to the given action.
func matches(rule pkg.PermissionRule, action pkg.Action) bool {
	switch rule.Scope {
	case "command":
		if action.Type != "command" {
			return false
		}
		return regexMatch(rule.Pattern, action.Target)
	case "path":
		if action.Type != "file_write" && action.Type != "file_delete" && action.Type != "file_read" {
			return false
		}
		return strings.HasPrefix(action.Target, rule.Pattern)
	case "tool":
		return action.ToolName != "" && stringMatch(rule.Pattern, action.ToolName)
	default:
		return regexMatch(rule.Pattern, action.Target)
	}
}

func regexMatch(pattern, target string) bool {
	matched, err := regexp.MatchString(pattern, target)
	if err != nil {
		return false
	}
	return matched
}

func stringMatch(pattern, target string) bool {
	return pattern == target || strings.HasPrefix(target, pattern)
}

// AddRule adds a rule and maintains priority ordering.
func (e *Engine) AddRule(rule pkg.PermissionRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
	e.sortRules()
	return nil
}

// RemoveRule removes a rule by its ID.
func (e *Engine) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	filtered := make([]pkg.PermissionRule, 0, len(e.rules))
	for _, r := range e.rules {
		if r.ID != ruleID {
			filtered = append(filtered, r)
		}
	}
	e.rules = filtered
	return nil
}

// ListRules returns a copy of the current rule list.
func (e *Engine) ListRules() []pkg.PermissionRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]pkg.PermissionRule, len(e.rules))
	copy(result, e.rules)
	return result
}

// LogAction records an audit log entry.
// Delegates to AuditLogger if attached, otherwise uses log.Printf.
func (e *Engine) LogAction(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error {
	if e.auditor != nil {
		return e.auditor.LogAction(ctx, "", action, decision, result)
	}
	log.Printf("[PERMISSION] action=%s target=%s decision=%s result=%s", action.Type, action.Target, decision, result)
	return nil
}

// sortRules sorts rules by Priority ascending.
func (e *Engine) sortRules() {
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority < e.rules[j].Priority
	})
}

// Compile-time interface check
var _ pkg.PermissionEngine = (*Engine)(nil)
