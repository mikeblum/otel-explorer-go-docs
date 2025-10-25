package repo

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mikeblum/otel-explorer-go-docs/conf"
)

const (
	cwd       = ".repo"
	perms     = 0755
	shaLength = 8
)

var repos = []string{
	"git@github.com:open-telemetry/opentelemetry-go.git",
	"git@github.com:open-telemetry/opentelemetry-go-contrib.git",
}

type RepoInfo struct {
	Head    string
	SHA     string
	Message string
}

func (r RepoInfo) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("head", r.Head),
		slog.String("sha", r.SHA),
		slog.String("message", r.Message),
	)
}

func name(url string) string {
	name := filepath.Base(url)
	return strings.TrimSuffix(name, ".git")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func clone(url, dir string) error {
	cmd := exec.Command("git", "clone", url)
	cmd.Dir = dir
	return cmd.Run()
}

func pull(path string) error {
	cmd := exec.Command("git", "pull", "--rebase")
	cmd.Dir = path
	return cmd.Run()
}

func info(path string) (*RepoInfo, error) {
	headCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	headCmd.Dir = path
	headOut, err := headCmd.Output()
	if err != nil {
		return nil, err
	}

	shaCmd := exec.Command("git", "log", "-1", "--format=%H")
	shaCmd.Dir = path
	shaOut, err := shaCmd.Output()
	if err != nil {
		return nil, err
	}

	msgCmd := exec.Command("git", "log", "-1", "--format=%s")
	msgCmd.Dir = path
	msgOut, err := msgCmd.Output()
	if err != nil {
		return nil, err
	}

	return &RepoInfo{
		Head:    strings.TrimSpace(string(headOut)),
		SHA:     strings.TrimSpace(string(shaOut))[:shaLength],
		Message: strings.TrimSpace(string(bytes.ReplaceAll(msgOut, []byte("\n"), []byte(" ")))),
	}, nil
}

func sync(url, dir string, log *conf.Log) error {
	name := name(url)
	path := filepath.Join(dir, name)

	if exists(path) {
		if err := pull(path); err != nil {
			log.WithErrorMsg(err, "Failed to pull repo", "repo", name, "action", "pull")
			return err
		}
	} else {
		if err := clone(url, dir); err != nil {
			log.WithErrorMsg(err, "Failed to clone repo", "repo", name, "action", "clone")
			return err
		}
	}

	info, err := info(path)
	if err != nil {
		log.WithErrorMsg(err, "Failed to resolve repo info", "repo", name)
		return err
	}
	log.Info(name, "info", *info)
	return nil
}

// Checkout clones the upstream opentelemetry-go repositories.
func Checkout() error {
	log := conf.NewLog()
	env := conf.NewEnv()

	workDir, err := env.WorkDir()
	if err != nil {
		log.WithErrorMsg(err, "Failed to resolve cwd")
		return err
	}

	cloneDir := filepath.Join(workDir, cwd)
	if err := os.MkdirAll(cloneDir, perms); err != nil {
		log.WithErrorMsg(err, "Failed to create clone directory", "dir", cloneDir, "perms", perms)
		return err
	}

	var errs error
	for _, repoURL := range repos {
		if err := sync(repoURL, cloneDir, log); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
