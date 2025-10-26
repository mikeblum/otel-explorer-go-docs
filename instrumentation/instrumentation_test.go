package instrumentation

import (
	"os"
	"path/filepath"
	"testing"
)

func getRepoPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(filepath.Dir(wd), ".repo/opentelemetry-go-contrib")
}

func assertSpanHasAttribute(t *testing.T, attributes []Attribute, name string) {
	t.Helper()
	for _, attr := range attributes {
		if attr.Name == name {
			return
		}
	}
	t.Errorf("Span missing required attribute %s", name)
}

func TestAWSSDKInstrumentation(t *testing.T) {
	repoPath := getRepoPath(t)
	analysis, err := AnalyzePackage(filepath.Join(repoPath, "instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"))
	if err != nil {
		t.Fatalf("AnalyzePackage() error = %v", err)
	}

	if got := len(analysis.Telemetry); got != 1 {
		t.Fatalf("Telemetry sections = %d, want 1", got)
	}

	tel := analysis.Telemetry[0]

	if got := len(tel.Spans); got == 0 {
		t.Fatal("Expected spans but got 0")
	}

	var hasClientSpan bool
	for _, span := range tel.Spans {
		if span.Kind == "CLIENT" {
			hasClientSpan = true
			assertSpanHasAttribute(t, span.Attributes, "rpc.system")
			assertSpanHasAttribute(t, span.Attributes, "rpc.service")
			assertSpanHasAttribute(t, span.Attributes, "rpc.method")
			break
		}
	}
	if !hasClientSpan {
		t.Error("No CLIENT span found")
	}
}

func TestGinInstrumentation(t *testing.T) {
	repoPath := getRepoPath(t)
	analysis, err := AnalyzePackage(filepath.Join(repoPath, "instrumentation/github.com/gin-gonic/gin/otelgin"))
	if err != nil {
		t.Fatalf("AnalyzePackage() error = %v", err)
	}

	if got := len(analysis.Telemetry); got != 1 {
		t.Fatalf("Telemetry sections = %d, want 1", got)
	}

	tel := analysis.Telemetry[0]

	if got := len(tel.Spans); got != 1 {
		t.Fatalf("Spans count = %d, want 1", got)
	}

	span := tel.Spans[0]
	if span.Kind != "SERVER" {
		t.Errorf("Span kind = %v, want SERVER", span.Kind)
	}

	assertSpanHasAttribute(t, span.Attributes, "http.request.method")
	assertSpanHasAttribute(t, span.Attributes, "http.response.status_code")
	assertSpanHasAttribute(t, span.Attributes, "http.route")

	if got := len(tel.Metrics); got != 3 {
		t.Errorf("Metrics count = %d, want 3", got)
	}
}

func TestGRPCInstrumentation(t *testing.T) {
	repoPath := getRepoPath(t)
	analysis, err := AnalyzePackage(filepath.Join(repoPath, "instrumentation/google.golang.org/grpc/otelgrpc"))
	if err != nil {
		t.Fatalf("AnalyzePackage() error = %v", err)
	}

	if got := len(analysis.Telemetry); got != 1 {
		t.Fatalf("Telemetry sections = %d, want 1", got)
	}

	tel := analysis.Telemetry[0]

	if got := len(tel.Metrics); got != 3 {
		t.Errorf("Metrics count = %d, want 3", got)
	}
}

func TestMongoInstrumentation(t *testing.T) {
	repoPath := getRepoPath(t)
	analysis, err := AnalyzePackage(filepath.Join(repoPath, "instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"))
	if err != nil {
		t.Fatalf("AnalyzePackage() error = %v", err)
	}

	if got := len(analysis.Telemetry); got != 1 {
		t.Fatalf("Telemetry sections = %d, want 1", got)
	}

	tel := analysis.Telemetry[0]

	if got := len(tel.Spans); got != 1 {
		t.Fatalf("Spans count = %d, want 1", got)
	}

	span := tel.Spans[0]
	if span.Kind != "CLIENT" {
		t.Errorf("Span kind = %v, want CLIENT", span.Kind)
	}

	assertSpanHasAttribute(t, span.Attributes, "db.system")
	assertSpanHasAttribute(t, span.Attributes, "db.operation.name")
}

func TestRestfulInstrumentation(t *testing.T) {
	repoPath := getRepoPath(t)
	analysis, err := AnalyzePackage(filepath.Join(repoPath, "instrumentation/github.com/emicklei/go-restful/otelrestful"))
	if err != nil {
		t.Fatalf("AnalyzePackage() error = %v", err)
	}

	if got := len(analysis.Telemetry); got != 1 {
		t.Fatalf("Telemetry sections = %d, want 1", got)
	}

	tel := analysis.Telemetry[0]

	if got := len(tel.Spans); got != 1 {
		t.Fatalf("Spans count = %d, want 1", got)
	}

	span := tel.Spans[0]
	if span.Kind != "SERVER" {
		t.Errorf("Span kind = %v, want SERVER", span.Kind)
	}

	assertSpanHasAttribute(t, span.Attributes, "http.request.method")
	assertSpanHasAttribute(t, span.Attributes, "http.response.status_code")

	if got := len(tel.Metrics); got != 3 {
		t.Errorf("Metrics count = %d, want 3", got)
	}
}

func TestLambdaInstrumentation(t *testing.T) {
	repoPath := getRepoPath(t)
	analysis, err := AnalyzePackage(filepath.Join(repoPath, "instrumentation/github.com/aws/aws-lambda-go/otellambda"))
	if err != nil {
		t.Fatalf("AnalyzePackage() error = %v", err)
	}

	if got := len(analysis.Telemetry); got != 1 {
		t.Fatalf("Telemetry sections = %d, want 1", got)
	}

	tel := analysis.Telemetry[0]

	if got := len(tel.Spans); got == 0 {
		t.Fatal("Expected spans but got 0")
	}
}

func TestFullScanValidation(t *testing.T) {
	t.Run("scan all instrumentation packages", func(t *testing.T) {
		repoPath := getRepoPath(t)
		libs, err := Scan(repoPath)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if got := len(libs); got != 14 {
			t.Errorf("Total libraries = %d, want 14", got)
		}

		stats := CalculateStats(libs)

		// Validate overall stats
		if stats.LibrariesWithTelemetry < 10 {
			t.Errorf("Libraries with telemetry = %d, want at least 10", stats.LibrariesWithTelemetry)
		}

		if stats.TotalSpans < 10 {
			t.Errorf("Total spans = %d, want at least 10", stats.TotalSpans)
		}

		if stats.TotalMetrics < 15 {
			t.Errorf("Total metrics = %d, want at least 15 (3 per HTTP/gRPC package)", stats.TotalMetrics)
		}

		if stats.TotalAttributes < 150 {
			t.Errorf("Total attributes = %d, want at least 150", stats.TotalAttributes)
		}

		// Validate span kinds distribution
		if stats.SpansByKind["SERVER"] < 5 {
			t.Errorf("SERVER spans = %d, want at least 5 (HTTP frameworks)", stats.SpansByKind["SERVER"])
		}

		if stats.SpansByKind["CLIENT"] < 3 {
			t.Errorf("CLIENT spans = %d, want at least 3 (AWS, mongo, etc)", stats.SpansByKind["CLIENT"])
		}

		// Validate semantic conventions
		if stats.LibrariesWithSemanticConventions < 12 {
			t.Errorf("Libraries with semantic conventions = %d, want at least 12", stats.LibrariesWithSemanticConventions)
		}

		// Log detailed breakdown for debugging
		t.Logf("Stats breakdown:")
		t.Logf("  Total libraries: %d", len(libs))
		t.Logf("  With telemetry: %d", stats.LibrariesWithTelemetry)
		t.Logf("  Total spans: %d", stats.TotalSpans)
		t.Logf("  Total metrics: %d", stats.TotalMetrics)
		t.Logf("  Total attributes: %d", stats.TotalAttributes)
		t.Logf("  Spans by kind: SERVER=%d, CLIENT=%d, INTERNAL=%d",
			stats.SpansByKind["SERVER"],
			stats.SpansByKind["CLIENT"],
			stats.SpansByKind["INTERNAL"])
	})

	t.Run("validate no duplicate telemetry", func(t *testing.T) {
		repoPath := getRepoPath(t)
		libs, err := Scan(repoPath)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		for _, lib := range libs {
			for _, tel := range lib.Telemetry {
				// Check for duplicate spans
				spanKinds := make(map[string]int)
				for _, span := range tel.Spans {
					spanKinds[span.Kind]++
				}

				for kind, count := range spanKinds {
					if count > 1 {
						t.Errorf("Library %s has %d duplicate %s spans, expected 1 per kind", lib.Name, count, kind)
					}
				}

				// Check for duplicate metrics
				metricNames := make(map[string]int)
				for _, metric := range tel.Metrics {
					metricNames[metric.Name]++
				}

				for name, count := range metricNames {
					if count > 1 {
						t.Errorf("Library %s has %d duplicate %s metrics, expected 1 per name", lib.Name, count, name)
					}
				}

				// Validate each span has attributes (except INTERNAL which may not)
				for _, span := range tel.Spans {
					if len(span.Attributes) == 0 && span.Kind != "INTERNAL" {
						t.Errorf("Library %s has %s span with no attributes", lib.Name, span.Kind)
					}
				}

				// Validate each metric has attributes
				for _, metric := range tel.Metrics {
					if len(metric.Attributes) == 0 {
						t.Errorf("Library %s has metric %s with no attributes", lib.Name, metric.Name)
					}
					if metric.Unit == "" {
						t.Errorf("Library %s has metric %s with no unit", lib.Name, metric.Name)
					}
				}
			}
		}
	})

	t.Run("validate expected packages have telemetry", func(t *testing.T) {
		repoPath := getRepoPath(t)
		libs, err := Scan(repoPath)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		libMap := make(map[string]*Library)
		for i := range libs {
			libMap[libs[i].Name] = &libs[i]
		}

		// Excluded packages:
		excluded := map[string]string{
			"otelecho": "https://github.com/open-telemetry/opentelemetry-go-contrib/issues/8056",
		}

		httpPackages := []string{"otelgin", "otelmux", "otelrestful"}
		for _, name := range httpPackages {
			if reason, skip := excluded[name]; skip {
				t.Logf("Skipping %s: %s", name, reason)
				continue
			}

			lib, exists := libMap[name]
			if !exists {
				t.Errorf("Expected HTTP package %s not found", name)
				continue
			}

			if len(lib.Telemetry) == 0 {
				t.Errorf("HTTP package %s has no telemetry", name)
				continue
			}

			tel := lib.Telemetry[0]

			if len(tel.Spans) != 1 || tel.Spans[0].Kind != "SERVER" {
				t.Errorf("HTTP package %s should have 1 SERVER span, got %d spans", name, len(tel.Spans))
			}

			if len(tel.Metrics) != 3 {
				t.Errorf("HTTP package %s should have 3 metrics, got %d", name, len(tel.Metrics))
			}
		}

		if lib, exists := libMap["otelmongo"]; exists {
			if len(lib.Telemetry) == 0 {
				t.Error("otelmongo has no telemetry")
			} else {
				tel := lib.Telemetry[0]
				if len(tel.Spans) != 1 || tel.Spans[0].Kind != "CLIENT" {
					t.Error("otelmongo should have 1 CLIENT span")
				}
			}
		}

		if lib, exists := libMap["otelgrpc"]; exists {
			if len(lib.Telemetry) == 0 {
				t.Error("otelgrpc has no telemetry")
			} else {
				tel := lib.Telemetry[0]
				if len(tel.Metrics) != 3 {
					t.Errorf("otelgrpc should have 3 RPC metrics, got %d", len(tel.Metrics))
				}
			}
		}
	})
}
