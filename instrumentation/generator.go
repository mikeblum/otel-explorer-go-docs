package instrumentation

import (
	"os"
	"path/filepath"

	"github.com/mikeblum/otel-explorer-go-docs/repo"
	"gopkg.in/yaml.v3"
)

func Generate(libraries []Library, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	defer func() {
		_ = encoder.Close()
	}()

	return encoder.Encode(libraries)
}

func Scan(repoName, repoPath string) ([]Library, error) {
	var scanPaths []string

	switch repoName {
	case repo.RepoContrib:
		scanPaths = []string{filepath.Join(repoPath, "instrumentation")}
	case repo.RepoGo:
		scanPaths = []string{repoPath}
	default:
		scanPaths = []string{filepath.Join(repoPath, "instrumentation")}
	}

	var libraries []Library
	for _, scanPath := range scanPaths {
		packages, err := Walk(scanPath)
		if err != nil {
			continue
		}

		for _, pkg := range packages {
			lib, err := Parse(pkg.GoModPath, repoPath, repoName)
			if err != nil {
				continue
			}
			libraries = append(libraries, *lib)
		}
	}

	return libraries, nil
}
