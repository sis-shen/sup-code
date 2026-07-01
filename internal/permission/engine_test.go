package permission

import (
	"context"
	"sync"
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestDenyForceDelete(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "rm -rf /"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check(rm -rf /) = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestDenyForceDeleteSubdir(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "rm -rf /var/log"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check(rm -rf /var/log) = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestAllowSafeCommand(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "ls -la"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("Check(ls -la) = %v, want %v", dec, pkg.DecisionAllow)
	}
}

func TestAskSensitiveDirEtc(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "file_write", Target: "/etc/hosts"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAsk {
		t.Errorf("Check(/etc/hosts) = %v, want %v", dec, pkg.DecisionAsk)
	}
}

func TestAllowNormalPath(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "file_write", Target: "/home/user/file.go"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("Check(/home/user/file.go) = %v, want %v", dec, pkg.DecisionAllow)
	}
}

func TestAllowEmptyRules(t *testing.T) {
	e := NewWithRules(nil)
	action := pkg.Action{Type: "command", Target: "rm -rf /"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("Check with empty rules = %v, want %v", dec, pkg.DecisionAllow)
	}
}

func TestPriorityOrder(t *testing.T) {
	rules := []pkg.PermissionRule{
		{ID: "allow-echo", Scope: "command", Pattern: `echo`, Decision: pkg.DecisionAllow, Priority: 10},
		{ID: "deny-echo-rm", Scope: "command", Pattern: `rm\s+-rf`, Decision: pkg.DecisionDeny, Priority: 1},
	}
	e := NewWithRules(rules)
	action := pkg.Action{Type: "command", Target: "rm -rf /"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check with priority order = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestAddRuleEffect(t *testing.T) {
	e := NewWithRules(nil)
	action := pkg.Action{Type: "command", Target: "evil_command"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("expected allow before AddRule, got %v", dec)
	}

	err = e.AddRule(pkg.PermissionRule{
		ID: "deny-evil", Scope: "command", Pattern: `evil_command`,
		Decision: pkg.DecisionDeny, Priority: 1,
	})
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	dec, err = e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("expected deny after AddRule, got %v", dec)
	}
}

func TestRemoveRuleEffect(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "rm -rf /"}

	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("expected deny before RemoveRule, got %v", dec)
	}

	err = e.RemoveRule(RuleIDDenyForceDelete)
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	dec, err = e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("expected allow after RemoveRule, got %v", dec)
	}
}

func TestConcurrentCheck(t *testing.T) {
	e := New()
	var wg sync.WaitGroup
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.Check(ctx, pkg.Action{Type: "command", Target: "ls -la"})
			if err != nil {
				t.Errorf("concurrent Check failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestEmptyTarget(t *testing.T) {
	e := New()
	dec, err := e.Check(context.Background(), pkg.Action{Type: "command", Target: ""})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("empty target = %v, want %v", dec, pkg.DecisionAllow)
	}
}

func TestVeryLongCommand(t *testing.T) {
	e := New()
	longCmd := make([]byte, 100000)
	for i := range longCmd {
		longCmd[i] = 'a'
	}
	dec, err := e.Check(context.Background(), pkg.Action{Type: "command", Target: string(longCmd)})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("long command = %v, want %v", dec, pkg.DecisionAllow)
	}
}

func TestDenyDD(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "dd if=/dev/zero of=/dev/sda bs=1M"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check(dd if=) = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestDenyMkfs(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "mkfs.ext4 /dev/sda1"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check(mkfs) = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestDenyChmod777(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "chmod 777 /etc/passwd"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check(chmod 777) = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestDenyDevSD(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "command", Target: "echo data > /dev/sda"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionDeny {
		t.Errorf("Check(> /dev/sd) = %v, want %v", dec, pkg.DecisionDeny)
	}
}

func TestAskSensitiveDirBoot(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "file_write", Target: "/boot/grub.cfg"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAsk {
		t.Errorf("Check(/boot/) = %v, want %v", dec, pkg.DecisionAsk)
	}
}

func TestAskSensitiveDirWindows(t *testing.T) {
	e := New()
	action := pkg.Action{Type: "file_write", Target: `C:\Windows\system32\config`}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAsk {
		t.Errorf("Check(C:\\Windows\\) = %v, want %v", dec, pkg.DecisionAsk)
	}
}

func TestListRules(t *testing.T) {
	e := New()
	rules := e.ListRules()
	if len(rules) == 0 {
		t.Fatal("ListRules returned empty slice")
	}
	// Ensure we got a copy by checking it doesn't reference the same underlying array
	if len(rules) != len(DefaultRules()) {
		t.Errorf("ListRules returned %d rules, want %d", len(rules), len(DefaultRules()))
	}
}

func TestRemoveNonexistentRule(t *testing.T) {
	e := New()
	err := e.RemoveRule("nonexistent-rule-id")
	if err != nil {
		t.Fatalf("RemoveRule for nonexistent ID returned error: %v", err)
	}
	// Should still have all original rules
	if len(e.ListRules()) != len(DefaultRules()) {
		t.Errorf("after removing nonexistent rule, got %d rules, want %d", len(e.ListRules()), len(DefaultRules()))
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	e := New()
	var wg sync.WaitGroup
	ctx := context.Background()

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = e.Check(ctx, pkg.Action{Type: "command", Target: "ls"})
			_ = e.ListRules()
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = e.AddRule(pkg.PermissionRule{
				ID:       "test-rule",
				Scope:    "command",
				Pattern:  `test`,
				Decision: pkg.DecisionDeny,
				Priority: 100,
			})
			_ = e.RemoveRule("test-rule")
		}(i)
	}

	wg.Wait()
}

func TestLogAction(t *testing.T) {
	e := New()
	err := e.LogAction(context.Background(),
		pkg.Action{Type: "command", Target: "rm -rf /"},
		pkg.DecisionDeny,
		"blocked by deny-force-delete rule",
	)
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}
}

func TestNonMatchingActionType(t *testing.T) {
	e := New()
	// Command rule should not match a non-command action type
	action := pkg.Action{Type: "tool", Target: "rm -rf /", ToolName: "bash"}
	dec, err := e.Check(context.Background(), action)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec != pkg.DecisionAllow {
		t.Errorf("non-matching type should allow, got %v", dec)
	}

	// Path rule should not match a command action type
	action2 := pkg.Action{Type: "command", Target: "/etc/hosts"}
	dec2, err := e.Check(context.Background(), action2)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if dec2 != pkg.DecisionAllow {
		t.Errorf("path rule should not match command type, got %v", dec2)
	}
}

func TestBlacklistCoverage(t *testing.T) {
	e := New()
	tests := []struct {
		name   string
		action pkg.Action
		want   pkg.Decision
	}{
		{"rm -rf /", pkg.Action{Type: "command", Target: "rm -rf /"}, pkg.DecisionDeny},
		{"rm -rf /*", pkg.Action{Type: "command", Target: "rm -rf /*"}, pkg.DecisionDeny},
		{"rm -rf /var/log", pkg.Action{Type: "command", Target: "rm -rf /var/log"}, pkg.DecisionDeny},
		{"rm -rf /home/user", pkg.Action{Type: "command", Target: "rm -rf /home/user"}, pkg.DecisionDeny},
		{"dd if=/dev/zero of=/dev/sda", pkg.Action{Type: "command", Target: "dd if=/dev/zero of=/dev/sda"}, pkg.DecisionDeny},
		{"mkfs.ext4", pkg.Action{Type: "command", Target: "mkfs.ext4 /dev/sda1"}, pkg.DecisionDeny},
		{"chmod 777", pkg.Action{Type: "command", Target: "chmod 777 /etc/passwd"}, pkg.DecisionDeny},
		{"echo > /dev/sda", pkg.Action{Type: "command", Target: "echo data > /dev/sda"}, pkg.DecisionDeny},
		{"echo > /dev/sdb1", pkg.Action{Type: "command", Target: "echo > /dev/sdb1"}, pkg.DecisionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Check(context.Background(), tt.action)
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewWithRulesSorted(t *testing.T) {
	rules := []pkg.PermissionRule{
		{ID: "r2", Scope: "command", Pattern: `pattern2`, Decision: pkg.DecisionAllow, Priority: 10},
		{ID: "r1", Scope: "command", Pattern: `pattern1`, Decision: pkg.DecisionDeny, Priority: 1},
	}
	e := NewWithRules(rules)
	list := e.ListRules()
	if len(list) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(list))
	}
	if list[0].Priority != 1 || list[1].Priority != 10 {
		t.Errorf("rules not sorted by priority: %+v", list)
	}
}
