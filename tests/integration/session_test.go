package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

func TestSession_CreateAndGet(t *testing.T) {
	sm, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	defer sm.CloseAll()

	ctx := context.Background()

	session, err := sm.Create(ctx, "test-session")
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "test-session", session.Title)
	assert.Equal(t, pkg.StateIdle, session.State)

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, session.Title, got.Title)
}

func TestSession_AppendMessage(t *testing.T) {
	sm, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	defer sm.CloseAll()

	ctx := context.Background()
	session, err := sm.Create(ctx, "test-msg")
	require.NoError(t, err)

	msg1 := pkg.Message{Role: pkg.RoleUser, Content: "Hello", Timestamp: time.Now()}
	msg2 := pkg.Message{Role: pkg.RoleAssistant, Content: "Hi there!", Timestamp: time.Now()}

	err = sm.AppendMessage(ctx, session.ID, msg1)
	require.NoError(t, err)
	err = sm.AppendMessage(ctx, session.ID, msg2)
	require.NoError(t, err)

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(got.Messages))
	assert.Equal(t, "Hello", got.Messages[0].Content)
	assert.Equal(t, "Hi there!", got.Messages[1].Content)
}

func TestSession_PersistAndRestore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.json")

	sm1, err := tui.NewSessionManager(dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	session, err := sm1.Create(ctx, "persist-test")
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		err := sm1.AppendMessage(ctx, session.ID, pkg.Message{
			Role:      pkg.RoleUser,
			Content:   "msg-" + string(rune('0'+i)),
			Timestamp: time.Now(),
		})
		require.NoError(t, err)
	}

	err = sm1.Close(ctx, session.ID)
	require.NoError(t, err)

	sm2, err := tui.NewSessionManager(dbPath)
	require.NoError(t, err)
	defer sm2.CloseAll()

	restored, err := sm2.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, restored.ID)
	assert.Equal(t, session.Title, restored.Title)
	assert.Equal(t, 3, len(restored.Messages))
}

func TestSession_ConcurrentSessions(t *testing.T) {
	sm, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	defer sm.CloseAll()

	ctx := context.Background()
	s1, err := sm.Create(ctx, "session-1")
	require.NoError(t, err)

	s2, err := sm.Create(ctx, "session-2")
	require.NoError(t, err)

	err = sm.AppendMessage(ctx, s1.ID, pkg.Message{Role: pkg.RoleUser, Content: "only in s1", Timestamp: time.Now()})
	require.NoError(t, err)

	err = sm.AppendMessage(ctx, s2.ID, pkg.Message{Role: pkg.RoleUser, Content: "only in s2", Timestamp: time.Now()})
	require.NoError(t, err)

	got1, err := sm.Get(ctx, s1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(got1.Messages))
	assert.Equal(t, "only in s1", got1.Messages[0].Content)

	got2, err := sm.Get(ctx, s2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(got2.Messages))
	assert.Equal(t, "only in s2", got2.Messages[0].Content)
}

func TestSession_List(t *testing.T) {
	sm, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	defer sm.CloseAll()

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := sm.Create(ctx, "list-session")
		require.NoError(t, err)
	}

	sessions, err := sm.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, len(sessions))
}

func TestSession_Delete(t *testing.T) {
	sm, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	defer sm.CloseAll()

	ctx := context.Background()
	session, err := sm.Create(ctx, "delete-me")
	require.NoError(t, err)

	err = sm.Delete(ctx, session.ID)
	require.NoError(t, err)

	_, err = sm.Get(ctx, session.ID)
	assert.Error(t, err)
}

func TestSession_SetStateAndPlan(t *testing.T) {
	sm, err := tui.NewSessionManager(filepath.Join(t.TempDir(), "sessions.json"))
	require.NoError(t, err)
	defer sm.CloseAll()

	ctx := context.Background()
	session, err := sm.Create(ctx, "state-plan-test")
	require.NoError(t, err)

	err = sm.SetState(ctx, session.ID, pkg.StatePlanning)
	require.NoError(t, err)

	plan := pkg.Plan{
		Goal: "test goal",
		Steps: []pkg.PlanItem{
			{ID: "step-1", Description: "Test step", Status: "pending"},
		},
	}
	err = sm.SetPlan(ctx, session.ID, plan)
	require.NoError(t, err)

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, pkg.StatePlanning, got.State)
	require.NotNil(t, got.Plan)
	assert.Equal(t, "test goal", got.Plan.Goal)
	assert.Equal(t, 1, len(got.Plan.Steps))
}
