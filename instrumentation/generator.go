package instrumentation

import (
	"os"
	"path/filepath"

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

func Scan(repoPath string) ([]Library, error) {
	instPath := filepath.Join(repoPath, "instrumentation")
	packages, err := Walk(instPath)
	if err != nil {
		return nil, err
	}

	var libraries []Library
	for _, pkg := range packages {
		lib, err := Parse(pkg.GoModPath, repoPath)
		if err != nil {
			continue
		}
		libraries = append(libraries, *lib)
	}

	return libraries, nil
}
