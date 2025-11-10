package main

import (
	"os"

	"github.com/mikeblum/otel-explorer-go-docs/conf"
	"github.com/mikeblum/otel-explorer-go-docs/instrumentation"
	"github.com/mikeblum/otel-explorer-go-docs/repo"
)

func main() {
	log := conf.NewLog()
	log.Info("🔭OTel Ecosystem Explorer: Golang 🔭")

	semconvPath, err := repo.CheckoutSemconv()
	if err != nil {
		log.WithErrorMsg(err, "Error checking out semantic conventions")
	} else {
		if err := instrumentation.LoadSemconv(semconvPath); err != nil {
			log.WithErrorMsg(err, "Error loading semantic conventions")
		}
	}

	repoInfos, err := repo.Checkout()
	if err != nil {
		log.WithErrorMsg(err, "Error checking out otel repos, exiting...")
		os.Exit(1)
	}

	var groups []instrumentation.Group
	groupsByRepo := make(map[string][]instrumentation.Group)

	for _, repoInfo := range repoInfos {
		scannedGroups, err := instrumentation.Scan(repoInfo.Name, repoInfo.Path)
		if err != nil {
			log.WithErrorMsg(err, "Error scanning instrumentation packages", "repo", repoInfo.Name)
			continue
		}
		groups = append(groups, scannedGroups...)
		groupsByRepo[repoInfo.Name] = scannedGroups
	}

	if err := instrumentation.Generate(groups); err != nil {
		log.WithErrorMsg(err, "Error generating instrumentation list")
		os.Exit(1)
	}

	repoStats := instrumentation.CalculateStats(groupsByRepo)
	for repoName, stats := range repoStats {
		log.Info("Scan complete ✅",
			"repo", repoName,
			"instrumentation", stats)
	}
}
