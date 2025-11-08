package instrumentation

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func Parse(goModPath string, repoRoot string, repoName string) (*Library, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, err
	}

	pkgPath := filepath.Dir(goModPath)
	relPath, err := filepath.Rel(repoRoot, pkgPath)
	if err != nil {
		return nil, err
	}

	pkgName := path.Base(modFile.Module.Mod.Path)

	lib := &Library{
		Repository:       repoName,
		Scope:            Scope{Name: modFile.Module.Mod.Path},
		Name:             pkgName,
		DisplayName:      generateDisplayName(pkgName),
		SourcePath:       relPath,
		MinimumGoVersion: extractGoVersion(modFile),
	}

	for _, req := range modFile.Require {
		if req.Indirect {
			continue
		}
		if strings.HasPrefix(req.Mod.Path, otelPrefix) {
			continue
		}
		if testDependencies[req.Mod.Path] {
			continue
		}

		lib.TargetVersions = &TargetVersions{
			Library: req.Mod.Version,
		}
		lib.LibraryLink = buildLibraryLink(req.Mod.Path)
		break
	}

	// Perform AST analysis to extract additional metadata
	analysis, err := AnalyzePackage(pkgPath)
	if err == nil && analysis != nil {
		lib.Description = analysis.Description
		lib.SemanticConventions = analysis.SemanticConventions
		lib.Telemetry = analysis.Telemetry
	}

	return lib, nil
}

func buildLibraryLink(pkg string) string {
	u := &url.URL{
		Scheme: httpsScheme,
		Host:   pkgGoDevHost,
		Path:   "/" + pkg,
	}
	return u.String()
}

func generateDisplayName(pkgName string) string {
	name := strings.TrimPrefix(pkgName, "otel")
	if len(name) == 0 {
		return ""
	}

	if displayName, ok := displayNameMap[name]; ok {
		return displayName
	}

	return strings.ToUpper(name[:1]) + name[1:]
}

func extractGoVersion(modFile *modfile.File) string {
	if modFile.Go == nil {
		return ""
	}
	return modFile.Go.Version
}
