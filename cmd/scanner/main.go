package main

import (
	"os"

	"github.com/mikeblum/otel-explorer-go-docs/conf"
	"github.com/mikeblum/otel-explorer-go-docs/repo"
)

func main() {
	log := conf.NewLog()
	if err := repo.Checkout(); err != nil {
		log.WithErrorMsg(err, "Error checking out otel repos, exiting...")
		os.Exit(1)
	}
}
