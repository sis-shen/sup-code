package permission

import "github.com/supcode/supcode/pkg"

// Rule IDs for built-in security rules
const (
	RuleIDDenyForceDelete = "deny-force-delete"
	RuleIDDenyDD          = "deny-dd"
	RuleIDDenyMkfs        = "deny-mkfs"
	RuleIDDenyChmod777    = "deny-chmod-777"
	RuleIDDenyForkBomb    = "deny-fork-bomb"
	RuleIDDenyDevSD       = "deny-dev-sd"

	RuleIDAskEtc     = "ask-etc"
	RuleIDAskSSH     = "ask-ssh"
	RuleIDAskBoot    = "ask-boot"
	RuleIDAskSys     = "ask-sys"
	RuleIDAskProc    = "ask-proc"
	RuleIDAskWindows = "ask-windows"
)

// DefaultRules returns the built-in security rules.
// Priority starts at 1 for all rules.
func DefaultRules() []pkg.PermissionRule {
	return []pkg.PermissionRule{
		// --- Dangerous command blacklist (deny) ---
		{
			ID:       RuleIDDenyForceDelete,
			Scope:    "command",
			Pattern:  `rm\s+-rf`,
			Decision: pkg.DecisionDeny,
			Priority: 1,
		},
		{
			ID:       RuleIDDenyDD,
			Scope:    "command",
			Pattern:  `dd\s+if=`,
			Decision: pkg.DecisionDeny,
			Priority: 1,
		},
		{
			ID:       RuleIDDenyMkfs,
			Scope:    "command",
			Pattern:  `mkfs`,
			Decision: pkg.DecisionDeny,
			Priority: 1,
		},
		{
			ID:       RuleIDDenyChmod777,
			Scope:    "command",
			Pattern:  `chmod\s+777`,
			Decision: pkg.DecisionDeny,
			Priority: 1,
		},
		{
			ID:       RuleIDDenyForkBomb,
			Scope:    "command",
			Pattern:  `:\(\)\{ :\|:& \};:`,
			Decision: pkg.DecisionDeny,
			Priority: 1,
		},
		{
			ID:       RuleIDDenyDevSD,
			Scope:    "command",
			Pattern:  `>\s*/dev/sd`,
			Decision: pkg.DecisionDeny,
			Priority: 1,
		},

		// --- Sensitive directory protection (ask) ---
		{
			ID:       RuleIDAskEtc,
			Scope:    "path",
			Pattern:  `/etc/`,
			Decision: pkg.DecisionAsk,
			Priority: 1,
		},
		{
			ID:       RuleIDAskSSH,
			Scope:    "path",
			Pattern:  `~/.ssh/`,
			Decision: pkg.DecisionAsk,
			Priority: 1,
		},
		{
			ID:       RuleIDAskBoot,
			Scope:    "path",
			Pattern:  `/boot/`,
			Decision: pkg.DecisionAsk,
			Priority: 1,
		},
		{
			ID:       RuleIDAskSys,
			Scope:    "path",
			Pattern:  `/sys/`,
			Decision: pkg.DecisionAsk,
			Priority: 1,
		},
		{
			ID:       RuleIDAskProc,
			Scope:    "path",
			Pattern:  `/proc/`,
			Decision: pkg.DecisionAsk,
			Priority: 1,
		},
		{
			ID:       RuleIDAskWindows,
			Scope:    "path",
			Pattern:  `C:\Windows\`,
			Decision: pkg.DecisionAsk,
			Priority: 1,
		},
	}
}
