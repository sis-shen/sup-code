package pkg

// Service keys exposed by Sup Harness plugins on the kernel Context. Keys are
// stable and aligned with dsh/Cordis (ctx.llm, ctx.tools, ...). Consumers must
// declare the key they need via core.Plugin.Inject.
const (
	ServiceConfig     = "config"
	ServiceLLM        = "llm"
	ServiceTools      = "tools"
	ServicePermission = "permission"
	ServiceSkills     = "skills"
	ServiceMCP        = "mcp"
	ServiceMemory     = "memory"
	ServiceWorktree   = "worktree"
)
