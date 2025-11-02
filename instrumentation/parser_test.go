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

		if lib.TargetVersions == nil || lib.TargetVersions.Library != "v1.11.0" {
			t.Errorf("TargetVersions.Library = %v, want %v", lib.TargetVersions, "v1.11.0")
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

		if lib.TargetVersions != nil {
			t.Errorf("TargetVersions = %v, want nil (no external library)", lib.TargetVersions)
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

		if lib.TargetVersions == nil || lib.TargetVersions.Library != "v1.11.0" {
			t.Errorf("TargetVersions.Library = %v, want v1.11.0", lib.TargetVersions)
		}
	})

	t.Run("parser - skips test dependencies", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")

		content := `module go.opentelemetry.io/contrib/instrumentation/test

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.38.0
	google.golang.org/grpc v1.76.0
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		lib, err := Parse(goModPath, tmpDir, repo.RepoContrib)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if lib.TargetVersions == nil || lib.TargetVersions.Library != "v1.76.0" {
			t.Errorf("TargetVersions.Library = %v, want v1.76.0 (should skip testify)", lib.TargetVersions)
		}

		expectedLink := "https://pkg.go.dev/google.golang.org/grpc"
		if lib.LibraryLink != expectedLink {
			t.Errorf("LibraryLink = %v, want %v (should skip testify)", lib.LibraryLink, expectedLink)
		}
	})

	t.Run("parser - handles no valid library dependency", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")

		content := `module go.opentelemetry.io/contrib/instrumentation/runtime

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/metric v1.38.0
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		lib, err := Parse(goModPath, tmpDir, repo.RepoContrib)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if lib.TargetVersions != nil {
			t.Errorf("TargetVersions = %v, want nil (no valid external library)", lib.TargetVersions)
		}

		if lib.LibraryLink != "" {
			t.Errorf("LibraryLink = %v, want empty (no valid external library)", lib.LibraryLink)
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

func TestGenerateDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		pkgName string
		want    string
	}{
		{
			name:    "aws instrumentation",
			pkgName: "otelaws",
			want:    "AWS",
		},
		{
			name:    "grpc instrumentation",
			pkgName: "otelgrpc",
			want:    "gRPC",
		},
		{
			name:    "http instrumentation",
			pkgName: "otelhttp",
			want:    "HTTP",
		},
		{
			name:    "httptrace instrumentation",
			pkgName: "otelhttptrace",
			want:    "HTTP Trace",
		},
		{
			name:    "gin instrumentation",
			pkgName: "otelgin",
			want:    "Gin",
		},
		{
			name:    "echo instrumentation",
			pkgName: "otelecho",
			want:    "Echo",
		},
		{
			name:    "mux instrumentation",
			pkgName: "otelmux",
			want:    "Mux",
		},
		{
			name:    "mongo instrumentation",
			pkgName: "otelmongo",
			want:    "MongoDB",
		},
		{
			name:    "restful instrumentation",
			pkgName: "otelrestful",
			want:    "RESTful",
		},
		{
			name:    "lambda instrumentation",
			pkgName: "otellambda",
			want:    "Lambda",
		},
		{
			name:    "xrayconfig instrumentation",
			pkgName: "xrayconfig",
			want:    "X-Ray Config",
		},
		{
			name:    "host instrumentation",
			pkgName: "host",
			want:    "Host",
		},
		{
			name:    "runtime instrumentation",
			pkgName: "runtime",
			want:    "Runtime",
		},
		{
			name:    "unknown instrumentation",
			pkgName: "otelunknown",
			want:    "Unknown",
		},
		{
			name:    "empty after prefix strip",
			pkgName: "otel",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateDisplayName(tt.pkgName)
			if got != tt.want {
				t.Errorf("generateDisplayName(%q) = %v, want %v", tt.pkgName, got, tt.want)
			}
		})
	}
}
