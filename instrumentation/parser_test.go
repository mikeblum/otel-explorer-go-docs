package instrumentation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeblum/otel-explorer-go-docs/repo"
)

func TestParse(t *testing.T) {
	t.Run("parser - parses valid go.mod file", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")

		content := `module go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin

go 1.24.0

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		groups, err := Parse(goModPath, tmpDir, repo.RepoContrib)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if groups == nil {
			t.Log("Parse returned nil groups (expected for packages without telemetry)")
		}
	})

}
