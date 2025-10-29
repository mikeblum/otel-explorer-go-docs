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

		lib.TargetVersions.Library = req.Mod.Version
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
	return strings.ToUpper(name[:1]) + name[1:]
}

func extractGoVersion(modFile *modfile.File) string {
	if modFile.Go == nil {
		return ""
	}
	return modFile.Go.Version
}
