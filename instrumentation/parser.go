package instrumentation

import (
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

const (
	otelPrefix   = "go.opentelemetry.io"
	httpsScheme  = "https"
	pkgGoDevHost = "pkg.go.dev"
)

var testDependencies = map[string]bool{
	"github.com/stretchr/testify":   true,
	"github.com/google/go-cmp":      true,
	"github.com/davecgh/go-spew":    true,
	"github.com/pmezard/go-difflib": true,
}

var displayNameMap = map[string]string{
	"aws":        "AWS",
	"grpc":       "gRPC",
	"http":       "HTTP",
	"httptrace":  "HTTP Trace",
	"gin":        "Gin",
	"echo":       "Echo",
	"mux":        "Mux",
	"mongo":      "MongoDB",
	"restful":    "RESTful",
	"lambda":     "Lambda",
	"xrayconfig": "X-Ray Config",
	"host":       "Host",
	"runtime":    "Runtime",
}

func Parse(goModPath string, repoRoot string, repoName string) ([]Group, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	_, err = modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, err
	}

	pkgPath := filepath.Dir(goModPath)

	analysis, err := AnalyzePackage(pkgPath)
	if err != nil {
		return nil, err
	}

	if analysis == nil || len(analysis.Groups) == 0 {
		return nil, nil
	}

	return analysis.Groups, nil
}
