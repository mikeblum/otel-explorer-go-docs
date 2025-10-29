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

		lib, err := Parse(goModPath, tmpDir, repo.RepoContrib)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if lib.Repository != repo.RepoContrib {
			t.Errorf("Repository = %v, want %v", lib.Repository, repo.RepoContrib)
		}

		if lib.Name != "otelgin" {
			t.Errorf("Name = %v, want %v", lib.Name, "otelgin")
		}

		if lib.Scope.Name != "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin" {
			t.Errorf("Scope.Name = %v, want %v", lib.Scope.Name, "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin")
		}

		if lib.TargetVersions.Library != "v1.11.0" {
			t.Errorf("TargetVersions.Library = %v, want %v", lib.TargetVersions.Library, "v1.11.0")
		}

		expectedLink := "https://pkg.go.dev/github.com/gin-gonic/gin"
		if lib.LibraryLink != expectedLink {
			t.Errorf("LibraryLink = %v, want %v", lib.LibraryLink, expectedLink)
		}

		if lib.SourcePath != "." {
			t.Errorf("SourcePath = %v, want %v", lib.SourcePath, ".")
		}
	})

	t.Run("parser - skips indirect dependencies", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")

		content := `module go.opentelemetry.io/contrib/instrumentation/test

go 1.24.0

require (
	go.opentelemetry.io/otel v1.38.0
)

require (
	github.com/some/package v1.0.0 // indirect
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		lib, err := Parse(goModPath, tmpDir, repo.RepoContrib)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if lib.TargetVersions.Library != "" {
			t.Errorf("TargetVersions.Library = %v, want empty", lib.TargetVersions.Library)
		}
	})

	t.Run("parser - skips otel dependencies", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")

		content := `module go.opentelemetry.io/contrib/instrumentation/test

go 1.24.0

require (
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	github.com/gin-gonic/gin v1.11.0
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		lib, err := Parse(goModPath, tmpDir, repo.RepoContrib)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if lib.TargetVersions.Library != "v1.11.0" {
			t.Errorf("TargetVersions.Library = %v, want v1.11.0", lib.TargetVersions.Library)
		}
	})
}

func TestBuildLibraryLink(t *testing.T) {
	tests := []struct {
		name string
		pkg  string
		want string
	}{
		{
			name: "buildLibraryLink - github package",
			pkg:  "github.com/gin-gonic/gin",
			want: "https://pkg.go.dev/github.com/gin-gonic/gin",
		},
		{
			name: "buildLibraryLink - google golang package",
			pkg:  "google.golang.org/grpc",
			want: "https://pkg.go.dev/google.golang.org/grpc",
		},
		{
			name: "buildLibraryLink - mongodb package",
			pkg:  "go.mongodb.org/mongo-driver",
			want: "https://pkg.go.dev/go.mongodb.org/mongo-driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLibraryLink(tt.pkg)
			if got != tt.want {
				t.Errorf("buildLibraryLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
