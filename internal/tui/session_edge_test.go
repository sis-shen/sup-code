package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func TestSessionSaveLoad(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_save.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Save Test")
	require.NoError(t, err)
	err = sm.AppendMessage(ctx, session.ID, pkg.Message{Role: pkg.RoleUser, Content: "Test message"})
	require.NoError(t, err)
	err = sm.CloseAll()
	require.NoError(t, err)

	sm2, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	got, err := sm2.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Save Test", got.Title)
	assert.Len(t, got.Messages, 1)
	_ = os.Remove(dbPath)
}

func TestSessionDBPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "dbpath_test.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	assert.Equal(t, dbPath, sm.DBPath())
	_ = os.Remove(dbPath)
}

func TestSessionAppendMessageToNonexistent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "append_err.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()
	err = sm.AppendMessage(ctx, "nonexistent-id", pkg.Message{Role: pkg.RoleUser, Content: "test"})
	assert.Error(t, err)
	_ = os.Remove(dbPath)
}

func TestSessionSetStateNonexistent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state_err.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()
	err = sm.SetState(ctx, "nonexistent", pkg.StatePlanning)
	assert.Error(t, err)
	_ = os.Remove(dbPath)
}

func TestSessionSetPlanNonexistent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plan_err.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()
	err = sm.SetPlan(ctx, "nonexistent", pkg.Plan{Goal: "test"})
	assert.Error(t, err)
	_ = os.Remove(dbPath)
}

func TestSessionCloseNonexistent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "close_err.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()
	err = sm.Close(ctx, "nonexistent")
	assert.Error(t, err)
	_ = os.Remove(dbPath)
}

func TestListEmpty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()
	sessions, err := sm.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, sessions)
	_ = os.Remove(dbPath)
}

func TestSQLitePersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")
	sm, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	ctx := context.Background()

	session, err := sm.Create(ctx, "Persist Test")
	require.NoError(t, err)
	err = sm.AppendMessage(ctx, session.ID, pkg.Message{Role: pkg.RoleUser, Content: "hello"})
	require.NoError(t, err)
	err = sm.Close(ctx, session.ID)
	require.NoError(t, err)

	sm2, err := NewSessionManager(dbPath)
	require.NoError(t, err)
	got, err := sm2.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Persist Test", got.Title)
	assert.Len(t, got.Messages, 1)
	_ = os.Remove(dbPath)
}
