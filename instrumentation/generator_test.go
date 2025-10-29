package instrumentation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeblum/otel-explorer-go-docs/repo"
	"gopkg.in/yaml.v3"
)

const (
	perms = 0755
)

func TestGenerate(t *testing.T) {
	t.Run("generator - writes valid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.yaml")

		libraries := []Library{
			{
				Name:       "otelgin",
				SourcePath: "/test/path",
				Scope: Scope{
					Name: "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin",
				},
				TargetVersions: TargetVersions{
					Library: "v1.11.0",
				},
				LibraryLink: "https://pkg.go.dev/github.com/gin-gonic/gin",
			},
		}

		if err := Generate(libraries, outputPath); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var result []Library
		if err := yaml.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if got := len(result); got != 1 {
			t.Fatalf("Generate() wrote %d libraries, want 1", got)
		}

		if got := result[0].Name; got != "otelgin" {
			t.Errorf("Generated library name = %v, want otelgin", got)
		}
	})

	t.Run("generator - handles empty library list", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.yaml")

		libraries := []Library{}

		if err := Generate(libraries, outputPath); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		var result []Library
		if err := yaml.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if got := len(result); got > 0 {
			t.Errorf("Generate() wrote %d libraries, want 0", got)
		}
	})
}

func TestScan(t *testing.T) {
	t.Run("scanner - scans valid instrumentation directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		instDir := filepath.Join(tmpDir, "instrumentation")

		pkgDir := filepath.Join(instDir, "github.com/gin-gonic/gin/otelgin")
		if err := os.MkdirAll(pkgDir, perms); err != nil {
			t.Fatal(err)
		}

		goModContent := `module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin

go 1.24.0

require (
	github.com/gin-gonic/gin v1.11.0
	go.opentelemetry.io/otel v1.38.0
)
`
		goModPath := filepath.Join(pkgDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		libraries, err := Scan(repo.RepoContrib, tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if got := len(libraries); got != 1 {
			t.Fatalf("Scan() found %d libraries, want 1", got)
		}

		if got := libraries[0].Repository; got != repo.RepoContrib {
			t.Errorf("Library repository = %v, want %v", got, repo.RepoContrib)
		}

		if got := libraries[0].Name; got != "otelgin" {
			t.Errorf("Library name = %v, want otelgin", got)
		}
	})

	t.Run("scanner - handles missing instrumentation directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		libraries, err := Scan(repo.RepoContrib, tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		if got := len(libraries); got != 0 {
			t.Errorf("Scan() found %d libraries, want 0 for missing directory", got)
		}
	})
}
