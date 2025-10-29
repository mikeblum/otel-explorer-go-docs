package main

import (
	"os"

	"github.com/mikeblum/otel-explorer-go-docs/conf"
	"github.com/mikeblum/otel-explorer-go-docs/instrumentation"
	"github.com/mikeblum/otel-explorer-go-docs/repo"
)

func main() {
	log := conf.NewLog()
	repoInfos, err := repo.Checkout()
	if err != nil {
		log.WithErrorMsg(err, "Error checking out otel repos, exiting...")
		os.Exit(1)
	}

	var libs []instrumentation.Library
	libsByRepo := make(map[string][]instrumentation.Library)

	for _, repoInfo := range repoInfos {
		scannedLibs, err := instrumentation.Scan(repoInfo.Name, repoInfo.Path)
		if err != nil {
			log.WithErrorMsg(err, "Error scanning instrumentation packages", "repo", repoInfo.Name)
			continue
		}
		libs = append(libs, scannedLibs...)
		libsByRepo[repoInfo.Name] = scannedLibs
	}

	if err := instrumentation.Generate(libs, "instrumentation-list.yaml"); err != nil {
		log.WithErrorMsg(err, "Error generating instrumentation list")
		os.Exit(1)
	}

	statsByRepo := instrumentation.CalculateStats(libsByRepo)
	for repoName, stats := range statsByRepo {
		log.Info("Scan complete",
			"repo", repoName,
			"instrumentation", stats)
	}
}
