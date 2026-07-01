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
}

// New creates a PermissionEngine with the built-in default rules.
func New() *Engine {
	e := &Engine{}
	e.rules = DefaultRules()
	return e
}

// NewWithRules creates a PermissionEngine with a custom set of rules.
func NewWithRules(rules []pkg.PermissionRule) *Engine {
	e := &Engine{}
	e.rules = rules
	e.sortRules()
	return e
}

// Check evaluates an action against the rule set.
// Rules are evaluated in priority order (ascending).
// Returns the Decision of the first matching rule, or DecisionAllow if no rule matches.
func (e *Engine) Check(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	if action.Target == "" {
		return pkg.DecisionAllow, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if matches(rule, action) {
			return rule.Decision, nil
		}
	}

	return pkg.DecisionAllow, nil
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
		// Apply to file operations
		if action.Type != "file_write" && action.Type != "file_delete" && action.Type != "file_read" {
			return false
		}
		return strings.HasPrefix(action.Target, rule.Pattern)
	case "tool":
		return action.ToolName != "" && stringMatch(rule.Pattern, action.ToolName)
	default:
		// For generic or unknown scopes, try regex on Target
		return regexMatch(rule.Pattern, action.Target)
	}
}

// regexMatch compiles and matches a regex pattern safely.
func regexMatch(pattern, target string) bool {
	matched, err := regexp.MatchString(pattern, target)
	if err != nil {
		return false
	}
	return matched
}

// stringMatch checks for exact or prefix match.
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

// RemoveRule removes a rule by its ID. Returns nil if ID does not exist.
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

// LogAction records an audit log entry. MVP implementation uses log.Printf.
func (e *Engine) LogAction(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error {
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
