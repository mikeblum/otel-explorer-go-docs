package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func setupGitRepo(t *testing.T, tmpDir string, cmds [][]string) {
	t.Helper()
	for _, cmd := range cmds {
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Dir = tmpDir
		if err := c.Run(); err != nil {
			t.Fatalf("failed to run %v: %v", cmd, err)
		}
	}
}

func TestName(t *testing.T) {
	tests := []struct {
		testName string
		url      string
		want     string
	}{
		{
			testName: "name - github ssh url",
			url:      "git@github.com:open-telemetry/opentelemetry-go.git",
			want:     "opentelemetry-go",
		},
		{
			testName: "name - github https url",
			url:      "https://github.com/open-telemetry/opentelemetry-go.git",
			want:     "opentelemetry-go",
		},
		{
			testName: "name - no git extension",
			url:      "git@github.com:open-telemetry/opentelemetry-go",
			want:     "opentelemetry-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got := name(tt.url)
			if got != tt.want {
				t.Errorf("name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExists(t *testing.T) {
	t.Run("exists - returns true for existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		if !exists(tmpDir) {
			t.Error("exists() = false, want true for existing directory")
		}
	})

	t.Run("exists - returns false for non-existing directory", func(t *testing.T) {
		nonExistent := filepath.Join(t.TempDir(), "does-not-exist")

		if exists(nonExistent) {
			t.Error("exists() = true, want false for non-existing directory")
		}
	})

	t.Run("exists - returns true for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		if !exists(file) {
			t.Error("exists() = false, want true for existing file")
		}
	})
}

func TestInfo(t *testing.T) {
	t.Run("info - returns commit info from git repo", func(t *testing.T) {
		requireGit(t)

		tmpDir := t.TempDir()

		cmds := [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@example.com"},
			{"git", "config", "user.name", "Test User"},
			{"git", "commit", "--allow-empty", "-m", "Initial commit"},
		}

		setupGitRepo(t, tmpDir, cmds)

		repoInfo, err := info(tmpDir)
		if err != nil {
			t.Fatalf("info() error = %v", err)
		}

		if repoInfo.Head == "" {
			t.Error("info() Head is empty")
		}

		if repoInfo.SHA == "" {
			t.Error("info() SHA is empty")
		}

		if repoInfo.Message != "Initial commit" {
			t.Errorf("info() Message = %v, want 'Initial commit'", repoInfo.Message)
		}
	})

	t.Run("info - returns error for non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := info(tmpDir)
		if err == nil {
			t.Error("info() expected error for non-git directory, got nil")
		}
	})

	t.Run("info - handles multiline commit messages", func(t *testing.T) {
		requireGit(t)

		tmpDir := t.TempDir()

		cmds := [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@example.com"},
			{"git", "config", "user.name", "Test User"},
			{"git", "commit", "--allow-empty", "-m", "First line\nSecond line"},
		}

		setupGitRepo(t, tmpDir, cmds)

		repoInfo, err := info(tmpDir)
		if err != nil {
			t.Fatalf("info() error = %v", err)
		}

		if repoInfo.Message != "First line Second line" {
			t.Errorf("info() Message = %v, want 'First line Second line'", repoInfo.Message)
		}
	})
}

func TestClone(t *testing.T) {
	t.Run("clone - returns error for invalid url", func(t *testing.T) {
		requireGit(t)

		tmpDir := t.TempDir()
		if err := clone("invalid-url", tmpDir); err == nil {
			t.Error("clone() expected error for invalid url, got nil")
		}
	})
}

func TestPull(t *testing.T) {
	t.Run("pull - returns error for non-git directory", func(t *testing.T) {
		requireGit(t)

		tmpDir := t.TempDir()
		if err := pull(tmpDir); err == nil {
			t.Error("pull() expected error for non-git directory, got nil")
		}
	})

	t.Run("pull - requires upstream branch", func(t *testing.T) {
		requireGit(t)

		tmpDir := t.TempDir()

		cmds := [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@example.com"},
			{"git", "config", "user.name", "Test User"},
			{"git", "commit", "--allow-empty", "-m", "initial commit"},
		}

		setupGitRepo(t, tmpDir, cmds)

		if err := pull(tmpDir); err == nil {
			t.Error("pull() expected error for repo without upstream, got nil")
		}
	})
}

func TestRepoInfoLogValue(t *testing.T) {
	t.Run("LogValue - returns slog.Value", func(t *testing.T) {
		info := RepoInfo{
			Head:    "abc1234",
			SHA:     "abc12345",
			Message: "test commit",
		}

		val := info.LogValue()
		if val.Kind().String() != "Group" {
			t.Errorf("LogValue() kind = %v, want Group", val.Kind())
		}
	})
}
