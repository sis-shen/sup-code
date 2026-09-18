package pkg

import "encoding/json"

// Tool pipeline event names, aligned with dsh/Cordis. The canonical order is:
//
//	tools/pre-execute  (waterfall)  permission short-circuit / param rewrite
//	tools/execute      (waterfall)  actual execution wrapper
//	tools/post-execute (waterfall)  result rewrite / git auto-commit
//	tools/result       (emit)       audit / telemetry
const (
	EventToolsPreExecute  = "tools/pre-execute"
	EventToolsExecute     = "tools/execute"
	EventToolsPostExecute = "tools/post-execute"
	EventToolsResult      = "tools/result"
)

// ToolInvocation is the payload carried through the tool pipeline events. The
// same value flows through pre-execute, execute, post-execute and result so
// listeners can observe and rewrite the call in place.
type ToolInvocation struct {
	// Name is the tool being invoked.
	Name string
	// Params is the raw arguments supplied by the caller.
	Params json.RawMessage
	// Result is populated after execution (post-execute / result).
	Result ToolResult
	// Decision is filled by the permission listener on pre-execute.
	Decision Decision
}
