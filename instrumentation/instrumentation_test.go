package instrumentation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeblum/otel-explorer-go-docs/repo"
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
	if span.Kind != SpanKindServer {
		t.Errorf("Span kind = %v, want %s", span.Kind, SpanKindServer)
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
	if span.Kind != SpanKindServer {
		t.Errorf("Span kind = %v, want %s", span.Kind, SpanKindServer)
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
		groups, err := Scan(repo.RepoContrib, repoPath)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if got := len(groups); got < 10 {
			t.Errorf("Total groups = %d, want at least 10", got)
		}

		groupsByRepo := map[string][]Group{
			repo.RepoContrib: groups,
		}
		repoStats := CalculateStats(groupsByRepo)
		stats := repoStats[repo.RepoContrib]

		if stats.TotalSpans < 10 {
			t.Errorf("Total spans = %d, want at least 10", stats.TotalSpans)
		}

		if stats.TotalMetrics < 15 {
			t.Errorf("Total metrics = %d, want at least 15", stats.TotalMetrics)
		}

		t.Logf("Stats breakdown:")
		t.Logf("  Total groups: %d", len(groups))
		t.Logf("  Total spans: %d", stats.TotalSpans)
		t.Logf("  Total metrics: %d", stats.TotalMetrics)
		t.Logf("  Total attributes: %d", stats.TotalAttributes)
	})

	t.Run("validate no duplicate groups", func(t *testing.T) {
		repoPath := getRepoPath(t)
		groups, err := Scan(repo.RepoContrib, repoPath)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		groupIDs := make(map[string]int)
		for _, group := range groups {
			groupIDs[group.ID]++
		}

		for id, count := range groupIDs {
			if count > 1 {
				t.Errorf("Duplicate group ID %s appears %d times", id, count)
			}
		}
	})

	t.Run("validate groups have required fields", func(t *testing.T) {
		repoPath := getRepoPath(t)
		groups, err := Scan(repo.RepoContrib, repoPath)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		for _, group := range groups {
			if group.ID == "" {
				t.Errorf("Group has empty ID")
			}
			if group.Type == "" {
				t.Errorf("Group %s has empty type", group.ID)
			}
			if group.Type != "span" && group.Type != "metric" {
				t.Errorf("Group %s has invalid type %s", group.ID, group.Type)
			}
		}
	})
}
