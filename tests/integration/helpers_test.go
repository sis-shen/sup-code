package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supcode/supcode/internal/agent"
	"github.com/supcode/supcode/internal/contextmgr"
	"github.com/supcode/supcode/internal/llm/openai"
	"github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/internal/tools/editfile"
	"github.com/supcode/supcode/internal/tools/glob"
	"github.com/supcode/supcode/internal/tools/grep"
	"github.com/supcode/supcode/internal/tools/readfile"
	"github.com/supcode/supcode/internal/tools/writefile"
	"github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

func fixturePath(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "tests")); err == nil {
			return filepath.Join(dir, "tests", "fixturess", "test_repo")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find project root from %s", cwd)
		}
		dir = parent
	}
}

func fixturePathForward(t *testing.T) string {
	return strings.ReplaceAll(fixturePath(t), "\\", "/")
}

type mockResponse struct {
	planContent      string
	selectionContent string
}

func newMockLLMServer(t *testing.T, mr mockResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		bodyStr := string(body)

		var response string
		if strings.Contains(bodyStr, "task planner") || strings.Contains(bodyStr, "break down") {
			response = buildSSE(mr.planContent)
		} else {
			response = buildSSE(mr.selectionContent)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, response)
	}))
}

func buildSSE(content string) string {
	contentEscaped, _ := json.Marshal(content)
	body := fmt.Sprintf(`data: {"id":"test-id","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":%s},"finish_reason":null}]}`+"\n\n", string(contentEscaped))
	body += `data: {"id":"test-id","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n"
	body += "data: [DONE]\n\n"
	return body
}

func setupRealComponents(t *testing.T) (*tui.SessionManager, *tools.Registry, *contextmgr.Manager) {
	t.Helper()
	permEng := permission.New()
	ctxMgr := contextmgr.NewManager()
	toolReg := tools.NewRegistry(permEng)
	for _, tool := range []pkg.Tool{
		&readfile.Tool{}, &writefile.Tool{}, &editfile.Tool{},
		&bash.Tool{}, &glob.Tool{}, &grep.Tool{},
	} {
		if err := toolReg.Register(tool); err != nil {
			t.Fatalf("register tool %s: %v", tool.Name(), err)
		}
	}
	dbDir := t.TempDir()
	sessionMgr, err := tui.NewSessionManager(filepath.Join(dbDir, "sessions.json"))
	if err != nil {
		t.Fatalf("create session manager: %v", err)
	}
	return sessionMgr, toolReg, ctxMgr
}

func newMockLLMClient(serverURL string) *openai.LLMClient {
	return openai.NewLLMClient(openai.Config{
		APIKey:    "sk-test",
		BaseURL:   serverURL,
		Model:     "gpt-4o-test",
		MaxTokens: 100,
	})
}

func setupAgent(t *testing.T, llmClient pkg.LLMClient, sessionMgr *tui.SessionManager, toolReg *tools.Registry, ctxMgr *contextmgr.Manager) *agent.Agent {
	t.Helper()
	ag, err := agent.NewAgent(agent.AgentConfig{
		LLMClient:      llmClient,
		ToolRegistry:   toolReg,
		ContextManager: ctxMgr,
		SessionManager: sessionMgr,
		MaxIterations:  5,
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return ag
}
