package instrumentation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mikeblum/otel-explorer-go-docs/metadata"
	"golang.org/x/mod/modfile"
)

const otelContribPrefix = "go.opentelemetry.io/contrib/"

var bridgeTargetMap = map[string]string{
	"otelslog":   "log/slog",
	"otellogr":   "github.com/go-logr/logr",
	"otelzap":    "go.uber.org/zap",
	"otellogrus": "github.com/sirupsen/logrus",
}

var bridgeDisplayNames = map[string]string{
	"slog":   "slog",
	"logr":   "logr",
	"zap":    "zap",
	"logrus": "logrus",
}

type ContribRequire struct {
	Path      string
	Version   string
	GoVersion string
}

func IsOTelContribRequire(path string) bool {
	return strings.HasPrefix(path, otelContribPrefix)
}

func ParseContribRequires(goModPath string) ([]ContribRequire, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	f, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, err
	}
	var goVer string
	if f.Go != nil {
		goVer = f.Go.Version
	}
	var results []ContribRequire
	for _, req := range f.Require {
		if req.Indirect || !IsOTelContribRequire(req.Mod.Path) {
			continue
		}
		results = append(results, ContribRequire{
			Path:      req.Mod.Path,
			Version:   req.Mod.Version,
			GoVersion: goVer,
		})
	}
	return results, nil
}

func DeriveMetadata(r ContribRequire) *metadata.Metadata {
	name := filepath.Base(r.Path)
	instrType := inferInstrType(r.Path)
	return &metadata.Metadata{
		Name:                name,
		DisplayName:         inferDisplayName(name),
		SourcePath:          strings.TrimPrefix(r.Path, "go.opentelemetry.io/contrib/"),
		Scope:               metadata.Scope{Name: r.Path},
		Module:              metadata.Module{Path: r.Path, Version: r.Version},
		TargetModule:        inferTarget(r.Path, name),
		GoMinVersion:        r.GoVersion,
		LibraryLink:         "https://pkg.go.dev/" + r.Path,
		InstrumentationType: instrType,
		Installation:        metadata.Installation{Type: inferInstallType(instrType)},
		Stability:           metadata.StabilityExperimental,
	}
}

func GenerateMetadataYAML(path string, m *metadata.Metadata) error {
	if existing, err := metadata.Load(path); err == nil && existing.Description != "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return encodeYAMLFile(path, m)
}

func inferInstrType(path string) metadata.InstrType {
	suffix := strings.TrimPrefix(path, otelContribPrefix)
	switch {
	case strings.HasPrefix(suffix, "bridges/"):
		return metadata.InstrTypeBridge
	case strings.HasPrefix(suffix, "exporters/"):
		return metadata.InstrTypeExporter
	case strings.HasPrefix(suffix, "propagators/"):
		return metadata.InstrTypePropagator
	case strings.HasPrefix(suffix, "samplers/"):
		return metadata.InstrTypeSDKComponent
	default:
		return metadata.InstrTypeWrapper
	}
}

func inferInstallType(t metadata.InstrType) metadata.InstallType {
	if t == metadata.InstrTypeWrapper {
		return metadata.InstallTypeWrapper
	}
	return metadata.InstallTypeImport
}

func inferTarget(path, name string) string {
	if target, ok := bridgeTargetMap[name]; ok {
		return target
	}
	suffix := strings.TrimPrefix(path, otelContribPrefix)
	if strings.HasPrefix(suffix, "instrumentation/") {
		parts := strings.Split(strings.TrimPrefix(suffix, "instrumentation/"), "/")
		if len(parts) > 1 {
			return strings.Join(parts[:len(parts)-1], "/")
		}
	}
	return ""
}

func inferDisplayName(name string) string {
	stripped := strings.TrimPrefix(name, "otel")
	if d, ok := displayNameMap[stripped]; ok {
		return d
	}
	if d, ok := bridgeDisplayNames[stripped]; ok {
		return d
	}
	if d, ok := displayNameMap[name]; ok {
		return d
	}
	return name
}
