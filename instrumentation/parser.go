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

func Parse(goModPath string) (*Library, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, err
	}

	lib := &Library{
		Scope:      Scope{Name: modFile.Module.Mod.Path},
		Name:       path.Base(modFile.Module.Mod.Path),
		SourcePath: filepath.Dir(goModPath),
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
