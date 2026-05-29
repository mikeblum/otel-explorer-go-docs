package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mikeblum/otel-explorer-go-docs/conf"
	"github.com/mikeblum/otel-explorer-go-docs/instrumentation"
	"github.com/mikeblum/otel-explorer-go-docs/metadata"
)

func main() {
	log := conf.NewLog()
	ctx := context.Background()

	token := os.Getenv("GITHUB_TOKEN")
	resolver := metadata.NewTagResolver(token)

	entries, err := os.ReadDir("instrumentation")
	if err != nil {
		log.WithErrorMsg(err, "failed to read instrumentation directory")
		os.Exit(1)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join("instrumentation", entry.Name())
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			continue
		}

		requires, err := instrumentation.ParseContribRequires(goModPath)
		if err != nil {
			log.WithErrorMsg(err, "failed to parse go.mod", "dir", dir)
			continue
		}

		for _, req := range requires {
			m := instrumentation.DeriveMetadata(req)
			outPath := filepath.Join(dir, "metadata", m.Name+".yaml")
			if err := instrumentation.GenerateMetadataYAML(outPath, m); err != nil {
				log.WithErrorMsg(err, "failed to write metadata", "path", outPath)
				continue
			}
			slog.Info("generated", "path", outPath)
		}

		items, err := metadata.LoadDir(filepath.Join(dir, "metadata"))
		if err != nil {
			log.WithErrorMsg(err, "failed to load metadata", "dir", dir)
			continue
		}
		for _, m := range items {
			latest, err := resolver.LatestModuleVersion(ctx, m.Module.Path)
			if err != nil {
				slog.Warn("could not resolve version", "module", m.Module.Path, "error", err)
				latest = m.Module.Version
			}
			slog.Info("instrumentation",
				"name", m.Name,
				"display_name", m.DisplayName,
				"module", m.Module.Path,
				"version_pinned", m.Module.Version,
				"version_latest", latest,
				"type", m.InstrumentationType,
				"stability", m.Stability,
				"semconv", m.SemanticConventions,
			)
		}
	}
}
