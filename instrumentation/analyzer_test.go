package instrumentation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzePackage(t *testing.T) {
	t.Run("analyzer - analyzes valid package with doc", func(t *testing.T) {
		tmpDir := t.TempDir()

		docContent := `// Package testpkg provides test instrumentation.
//
// This package instruments test operations.
package testpkg
`
		docPath := filepath.Join(tmpDir, "doc.go")
		if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		if analysis == nil {
			t.Fatal("AnalyzePackage() returned nil")
		}

		if got := analysis.Name; got != "testpkg" {
			t.Errorf("Name = %v, want testpkg", got)
		}

		if got := analysis.Description; got == "" {
			t.Error("Description is empty")
		}
	})

	t.Run("analyzer - extracts telemetry with span kinds", func(t *testing.T) {
		tmpDir := t.TempDir()

		instrumentContent := `package testpkg

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

func InstrumentRequest(ctx context.Context, tracer trace.Tracer) {
	ctx, span := tracer.Start(ctx, "http.request", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
}
`
		instrumentPath := filepath.Join(tmpDir, "instrument.go")
		if err := os.WriteFile(instrumentPath, []byte(instrumentContent), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24

require go.opentelemetry.io/otel v1.38.0
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		if got := len(analysis.Telemetry); got != 1 {
			t.Fatalf("Telemetry count = %d, want 1", got)
		}

		tel := analysis.Telemetry[0]
		if tel.When != "default" {
			t.Errorf("When = %v, want default", tel.When)
		}

		if got := len(tel.Spans); got != 1 {
			t.Fatalf("Spans count = %d, want 1", got)
		}

		span := tel.Spans[0]
		if span.Kind != "SERVER" {
			t.Errorf("Span kind = %v, want SERVER", span.Kind)
		}
	})

	t.Run("analyzer - extracts semantic conventions from imports", func(t *testing.T) {
		tmpDir := t.TempDir()

		mainContent := `package testpkg

import (
	"go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func doSomething() {
	// Use semconv
	_ = semconv.HTTPMethodKey
}
`
		mainPath := filepath.Join(tmpDir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24

require (
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
)
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		if got := len(analysis.SemanticConventions); got == 0 {
			t.Error("SemanticConventions is empty, expected at least one")
		}

		foundSemconv := false
		for _, conv := range analysis.SemanticConventions {
			if conv == "go.opentelemetry.io/otel/semconv/v1.20.0" {
				foundSemconv = true
				break
			}
		}

		if !foundSemconv {
			t.Errorf("SemanticConventions = %v, want to contain semconv import", analysis.SemanticConventions)
		}
	})

	t.Run("analyzer - handles package without telemetry", func(t *testing.T) {
		tmpDir := t.TempDir()

		mainContent := `package testpkg

func DoSomething() {}
`
		mainPath := filepath.Join(tmpDir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		if got := len(analysis.Telemetry); got != 0 {
			t.Errorf("Telemetry count = %d, want 0", got)
		}
	})

	t.Run("analyzer - handles non-existent package", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist")

		_, err := AnalyzePackage(nonExistent)
		if err == nil {
			t.Error("AnalyzePackage() expected error for non-existent package, got nil")
		}
	})
}

func TestExtractTelemetry(t *testing.T) {
	t.Run("extractTelemetry - finds span and metric creation", func(t *testing.T) {
		tmpDir := t.TempDir()

		content := `package testpkg

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func instrument(ctx context.Context, tracer trace.Tracer, meter metric.Meter) {
	ctx, span := tracer.Start(ctx, "operation.name",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("http.method", "GET")))
	defer span.End()

	counter, _ := meter.Int64Counter("request.count")
	histogram, _ := meter.Float64Histogram("request.duration")
}
`
		filePath := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24

require go.opentelemetry.io/otel v1.38.0
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		if got := len(analysis.Telemetry); got != 1 {
			t.Fatalf("Telemetry count = %d, want 1", got)
		}

		tel := analysis.Telemetry[0]

		if got := len(tel.Spans); got != 1 {
			t.Fatalf("Spans count = %d, want 1", got)
		}

		span := tel.Spans[0]
		if span.Kind != "CLIENT" {
			t.Errorf("Span kind = %v, want CLIENT", span.Kind)
		}

		if got := len(span.Attributes); got < 1 {
			t.Errorf("Attributes count = %d, want at least 1", got)
		}

		foundHTTPMethod := false
		for _, attr := range span.Attributes {
			if attr.Name == "http.method" && attr.Type == "STRING" {
				foundHTTPMethod = true
			}
		}
		if !foundHTTPMethod {
			t.Error("Expected http.method STRING attribute not found")
		}

		if got := len(tel.Metrics); got != 2 {
			t.Fatalf("Metrics count = %d, want 2", got)
		}

		foundCounter := false
		foundHistogram := false
		for _, metric := range tel.Metrics {
			if metric.Name == "request.count" && metric.Type == "COUNTER" {
				foundCounter = true
			}
			if metric.Name == "request.duration" && metric.Type == "HISTOGRAM" {
				foundHistogram = true
			}
		}

		if !foundCounter {
			t.Error("Expected request.count COUNTER metric not found")
		}
		if !foundHistogram {
			t.Error("Expected request.duration HISTOGRAM metric not found")
		}
	})
}

func TestExtractSemanticConventions(t *testing.T) {
	t.Run("extractSemanticConventions - finds semconv imports", func(t *testing.T) {
		tmpDir := t.TempDir()

		content := `package testpkg

import (
	"go.opentelemetry.io/otel/semconv/v1.20.0"
	"fmt"
)

func main() {
	fmt.Println(semconv.HTTPMethodKey)
}
`
		filePath := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24

require go.opentelemetry.io/otel v1.38.0
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		conventions := analysis.SemanticConventions
		if got := len(conventions); got == 0 {
			t.Error("extractSemanticConventions() found no conventions, want at least 1")
		}
	})

	t.Run("extractSemanticConventions - ignores non-semconv imports", func(t *testing.T) {
		tmpDir := t.TempDir()

		content := `package testpkg

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("hello")
	http.Get("url")
}
`
		filePath := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		goModContent := `module example.com/testpkg

go 1.24
`
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}

		analysis, err := AnalyzePackage(tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}

		conventions := analysis.SemanticConventions
		if got := len(conventions); got != 0 {
			t.Errorf("extractSemanticConventions() found %d conventions, want 0", got)
		}
	})
}
