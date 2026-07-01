package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func newTestSessionManager(t *testing.T) *SessionManager {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_sessions.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sm.CloseAll()
		os.Remove(dbPath)
	})
	return sm
}

func TestCreate(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Test Session")
	require.NoError(t, err)

	assert.NotEmpty(t, session.ID, "session ID should not be empty")
	assert.Equal(t, "Test Session", session.Title)
	assert.Equal(t, pkg.StateIdle, session.State)
	assert.False(t, session.CreatedAt.IsZero(), "created_at should not be zero")
	assert.False(t, session.UpdatedAt.IsZero(), "updated_at should not be zero")
	assert.Empty(t, session.Messages)
	_, err = uuid.Parse(session.ID)
	assert.NoError(t, err, "session ID should be a valid UUID")
}

func TestCreateUniqueIDs(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	s1, err := sm.Create(ctx, "Session 1")
	require.NoError(t, err)
	s2, err := sm.Create(ctx, "Session 2")
	require.NoError(t, err)
	assert.NotEqual(t, s1.ID, s2.ID, "session IDs should be unique")
}

func TestGetAfterCreate(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	created, err := sm.Create(ctx, "Test Session")
	require.NoError(t, err)

	got, err := sm.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.Title, got.Title)
}

func TestGetNonexistent(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	_, err := sm.Get(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAppendAndGetMessages(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Test Session")
	require.NoError(t, err)

	msg1 := pkg.Message{
		Role:      pkg.RoleUser,
		Content:   "Hello",
		Timestamp: time.Now().UTC(),
	}
	err = sm.AppendMessage(ctx, session.ID, msg1)
	require.NoError(t, err)

	msg2 := pkg.Message{
		Role:      pkg.RoleAssistant,
		Content:   "Hi there!",
		Timestamp: time.Now().UTC(),
	}
	err = sm.AppendMessage(ctx, session.ID, msg2)
	require.NoError(t, err)

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Len(t, got.Messages, 2)
	assert.Equal(t, "Hello", got.Messages[0].Content)
	assert.Equal(t, "Hi there!", got.Messages[1].Content)
}

func TestSetState(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Test")
	require.NoError(t, err)

	err = sm.SetState(ctx, session.ID, pkg.StatePlanning)
	require.NoError(t, err)

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, pkg.StatePlanning, got.State)
}

func TestSetPlan(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Test")
	require.NoError(t, err)

	plan := pkg.Plan{
		Goal: "Test goal",
		Steps: []pkg.PlanItem{
			{ID: "1", Description: "Step 1", Status: "pending"},
		},
	}
	err = sm.SetPlan(ctx, session.ID, plan)
	require.NoError(t, err)

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Plan)
	assert.Equal(t, "Test goal", got.Plan.Goal)
	assert.Len(t, got.Plan.Steps, 1)
}

func TestDelete(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	session, err := sm.Create(ctx, "To Delete")
	require.NoError(t, err)

	err = sm.Delete(ctx, session.ID)
	require.NoError(t, err)

	_, err = sm.Get(ctx, session.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClosePersistsToDB(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Persist Test")
	require.NoError(t, err)

	msg := pkg.Message{Role: pkg.RoleUser, Content: "test", Timestamp: time.Now().UTC()}
	err = sm.AppendMessage(ctx, session.ID, msg)
	require.NoError(t, err)

	err = sm.SetState(ctx, session.ID, pkg.StateCompleted)
	require.NoError(t, err)

	err = sm.Close(ctx, session.ID)
	require.NoError(t, err)

	assert.Equal(t, 0, sm.CacheSize(), "cache should be empty after close")

	got, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, pkg.StateCompleted, got.State)
	assert.Len(t, got.Messages, 1)
}

func TestListOrder(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	s1, err := sm.Create(ctx, "First")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	s2, err := sm.Create(ctx, "Second")
	require.NoError(t, err)

	sessions, err := sm.List(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions, 2, "should list 2 sessions")
	if len(sessions) >= 2 {
		assert.Equal(t, s2.Title, sessions[0].Title, "newest should be first")
		assert.Equal(t, s1.Title, sessions[1].Title, "oldest should be last")
	}
}

func TestConcurrentCreate(t *testing.T) {
	sm := newTestSessionManager(t)
	ctx := context.Background()

	ids := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func() {
			s, err := sm.Create(ctx, "Concurrent")
			if err == nil {
				ids <- s.ID
			}
		}()
	}

	collected := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := <-ids
		assert.False(t, collected[id], "duplicate ID: %s", id)
		collected[id] = true
	}
	assert.Len(t, collected, 10)
}

func TestCacheEviction(t *testing.T) {
	sm := newTestSessionManager(t)
	sm.maxHot = 3
	ctx := context.Background()

	sessions := make([]*pkg.Session, 5)
	for i := 0; i < 5; i++ {
		s, err := sm.Create(ctx, "Session")
		require.NoError(t, err)
		sessions[i] = s
	}

	assert.LessOrEqual(t, sm.CacheSize(), 3, "cache should be limited to maxHot")
}
