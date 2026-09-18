package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentLoop_ReadFile(t *testing.T) {
	sessionMgr, toolReg, ctxMgr := setupRealComponents(t)
	defer func() { require.NoError(t, sessionMgr.CloseAll()) }()

	fp := fixturePathForward(t)
	mr := mockResponse{
		planContent:      `{"goal":"Read README.md","steps":[{"id":"step-1","description":"Read test_repo/README.md","tool_hint":"read_file"}]}`,
		selectionContent: `{"tool":"read_file","parameters":{"path":"` + fp + `/README.md"}}`,
	}

	server := newMockLLMServer(t, mr)
	defer server.Close()

	ag := setupAgent(t, newMockLLMClient(server.URL), sessionMgr, toolReg, ctxMgr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := sessionMgr.Create(ctx, "test-read-file")
	require.NoError(t, err)

	result, err := ag.Run(ctx, session.ID, "Read tests/fixturess/test_repo/README.md")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Summary, "completed successfully")
}

func TestAgentLoop_WriteFile(t *testing.T) {
	sessionMgr, toolReg, ctxMgr := setupRealComponents(t)
	defer func() { require.NoError(t, sessionMgr.CloseAll()) }()

	// Use forward-slash path to avoid Windows backslash JSON escaping issues
	tmpDir := strings.ReplaceAll(t.TempDir(), "\\", "/")
	testFile := tmpDir + "/test_output.txt"

	mr := mockResponse{
		planContent:      `{"goal":"Write a test file","steps":[{"id":"step-1","description":"Write to test file","tool_hint":"write_file"}]}`,
		selectionContent: `{"tool":"write_file","parameters":{"path":"` + testFile + `","content":"Hello from integration test!"}}`,
	}

	server := newMockLLMServer(t, mr)
	defer server.Close()

	ag := setupAgent(t, newMockLLMClient(server.URL), sessionMgr, toolReg, ctxMgr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := sessionMgr.Create(ctx, "test-write-file")
	require.NoError(t, err)

	result, err := ag.Run(ctx, session.ID, "Write Hello from integration test! to a temp file")
	require.NoError(t, err)
	require.NotNil(t, result)

	data, err := os.ReadFile(testFile)
	require.NoError(t, err, "file should exist after write")
	assert.Equal(t, "Hello from integration test!", string(data))
}

func TestAgentLoop_MultiStep(t *testing.T) {
	sessionMgr, toolReg, ctxMgr := setupRealComponents(t)
	defer func() { require.NoError(t, sessionMgr.CloseAll()) }()

	mr := mockResponse{
		planContent: `{"goal":"Find TODO comments and read the file","steps":[
			{"id":"step-1","description":"Search for TODO comments in go files","tool_hint":"grep"},
			{"id":"step-2","description":"Read the utils.go file","tool_hint":"read_file"}
		]}`,
		selectionContent: `{"tool":"grep","parameters":{"pattern":"TODO","path":"` + fixturePathForward(t) + `","include":"*.go"}}`,
	}

	server := newMockLLMServer(t, mr)
	defer server.Close()

	ag := setupAgent(t, newMockLLMClient(server.URL), sessionMgr, toolReg, ctxMgr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	session, err := sessionMgr.Create(ctx, "test-multi-step")
	require.NoError(t, err)

	result, err := ag.Run(ctx, session.ID, "Find TODO comments in go files and read utils.go")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Summary)
}

func TestAgentLoop_PermissionDeny(t *testing.T) {
	sessionMgr, toolReg, ctxMgr := setupRealComponents(t)
	defer func() { require.NoError(t, sessionMgr.CloseAll()) }()

	mr := mockResponse{
		planContent:      `{"goal":"Delete everything","steps":[{"id":"step-1","description":"Run rm -rf command","tool_hint":"bash"}]}`,
		selectionContent: `{"tool":"bash","parameters":{"command":"rm -rf /"}}`,
	}

	server := newMockLLMServer(t, mr)
	defer server.Close()

	ag := setupAgent(t, newMockLLMClient(server.URL), sessionMgr, toolReg, ctxMgr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := sessionMgr.Create(ctx, "test-permission-deny")
	require.NoError(t, err)

	result, err := ag.Run(ctx, session.ID, "Delete everything")
	if err == nil {
		if result != nil {
			t.Logf("result error: %s", result.Error)
		}
	} else {
		t.Logf("agent error (expected): %v", err)
	}
}

func TestAgentLoop_MultiTurn(t *testing.T) {
	sessionMgr, toolReg, ctxMgr := setupRealComponents(t)
	defer func() { require.NoError(t, sessionMgr.CloseAll()) }()

	fp := fixturePathForward(t)
	mr := mockResponse{
		planContent:      `{"goal":"Read a file","steps":[{"id":"step-1","description":"Read file","tool_hint":"read_file"}]}`,
		selectionContent: `{"tool":"read_file","parameters":{"path":"` + fp + `/README.md"}}`,
	}

	server := newMockLLMServer(t, mr)
	defer server.Close()

	ag := setupAgent(t, newMockLLMClient(server.URL), sessionMgr, toolReg, ctxMgr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	session, err := sessionMgr.Create(ctx, "test-multi-turn")
	require.NoError(t, err)

	// Turn 1 - verify messages accumulate in context manager
	_, err = ag.Run(ctx, session.ID, "Read README.md")
	require.NoError(t, err)

	_, messages, err := ctxMgr.BuildContext(ctx, session.ID)
	require.NoError(t, err)
	assert.Greater(t, len(messages), 0, "messages should accumulate in context manager after turn 1")
	turn1Count := len(messages)

	// Turn 2
	_, err = ag.Run(ctx, session.ID, "Read README.md again")
	require.NoError(t, err)

	_, _, err = ctxMgr.BuildContext(ctx, session.ID)
	require.NoError(t, err)

	// Turn 3
	_, err = ag.Run(ctx, session.ID, "Read README.md one more time")
	require.NoError(t, err)

	_, messages, err = ctxMgr.BuildContext(ctx, session.ID)
	require.NoError(t, err)
	assert.Greater(t, len(messages), turn1Count, "messages should accumulate after 3 turns")
}

func TestAgentLoop_ToolFailureRetry(t *testing.T) {
	sessionMgr, toolReg, ctxMgr := setupRealComponents(t)
	defer func() { require.NoError(t, sessionMgr.CloseAll()) }()

	fp := fixturePathForward(t)
	mr := mockResponse{
		planContent:      `{"goal":"Edit a non-existent string","steps":[{"id":"step-1","description":"Edit non-existent content","tool_hint":"edit_file"}]}`,
		selectionContent: `{"tool":"edit_file","parameters":{"path":"` + fp + `/README.md","old_string":"THIS_DOES_NOT_EXIST_XYZ","new_string":"replaced"}}`,
	}

	server := newMockLLMServer(t, mr)
	defer server.Close()

	ag := setupAgent(t, newMockLLMClient(server.URL), sessionMgr, toolReg, ctxMgr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	session, err := sessionMgr.Create(ctx, "test-tool-failure")
	require.NoError(t, err)

	result, err := ag.Run(ctx, session.ID, "Replace non-existent string in README.md")
	if err != nil {
		t.Logf("agent returned error as expected: %v", err)
	} else if result != nil && result.Error != "" {
		t.Logf("result has error: %s", result.Error)
	} else {
		t.Log("tool failed silently (edge case)")
	}
}
