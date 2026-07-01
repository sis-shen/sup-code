package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/internal"
	"github.com/supcode/supcode/internal/cli"
)

func TestSmoke_Build(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping build test in short mode")
	}
	buildDir := t.TempDir()
	output := filepath.Join(buildDir, "supcode")

	cmd := exec.Command("go", "build", "-o", output, "./cmd/supcode")
	cmd.Dir = findProjectRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\noutput: %s", err, string(out))
	}
	_, err = os.Stat(output)
	require.NoError(t, err, "binary should exist after build")
}

func TestSmoke_Help(t *testing.T) {
	rootCmd := createRootCmd(t)
	require.NotNil(t, rootCmd)
	assert.Contains(t, rootCmd.Use, "supcode")
	assert.Contains(t, rootCmd.Short, "SupCode")
}

func TestSmoke_Version(t *testing.T) {
	assert.Equal(t, "0.1.0", cli.Version())
}

func createRootCmd(t *testing.T) *cobra.Command {
	t.Helper()
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-smoke-test")
	sc, err := internal.NewSupCode("")
	require.NoError(t, err)
	t.Cleanup(func() { sc.Close() })
	cli.SetApp(sc)
	return cli.RootCmd()
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
