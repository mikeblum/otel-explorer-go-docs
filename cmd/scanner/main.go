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
	for _, repoInfo := range repoInfos {
		if repoInfo.Name == repo.RepoContrib {
			scannedLibs, err := instrumentation.Scan(repoInfo.Path)
			if err != nil {
				log.WithErrorMsg(err, "Error scanning instrumentation packages", "repo", repoInfo.Name)
				os.Exit(1)
			}
			libs = append(libs, scannedLibs...)
		}
	}

	if err := instrumentation.Generate(libs, "instrumentation-list.yaml"); err != nil {
		log.WithErrorMsg(err, "Error generating instrumentation list")
		os.Exit(1)
	}

	stats := instrumentation.CalculateStats(libs)
	log.Info("Scan complete",
		"instrumentation", stats)
}
