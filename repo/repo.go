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

	RepoGo      = "opentelemetry-go"
	RepoContrib = "opentelemetry-go-contrib"
)

var repos = []string{
	"git@github.com:open-telemetry/opentelemetry-go.git",
	"git@github.com:open-telemetry/opentelemetry-go-contrib.git",
}

type RepoInfo struct {
	Name    string
	Path    string
	Head    string
	SHA     string
	Message string
}

func (r RepoInfo) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", r.Name),
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

func sync(url, dir string, log *conf.Log) (*RepoInfo, error) {
	repoName := name(url)
	repoPath := filepath.Join(dir, repoName)

	if exists(repoPath) {
		if err := pull(repoPath); err != nil {
			log.WithErrorMsg(err, "Failed to pull repo", "repo", repoName, "action", "pull")
			return nil, err
		}
	} else {
		if err := clone(url, dir); err != nil {
			log.WithErrorMsg(err, "Failed to clone repo", "repo", repoName, "action", "clone")
			return nil, err
		}
	}

	commitInfo, err := info(repoPath)
	if err != nil {
		log.WithErrorMsg(err, "Failed to resolve repo info", "repo", repoName)
		return nil, err
	}

	repoInfo := &RepoInfo{
		Name:    repoName,
		Path:    repoPath,
		Head:    commitInfo.Head,
		SHA:     commitInfo.SHA,
		Message: commitInfo.Message,
	}

	log.Info(repoName, "info", *repoInfo)
	return repoInfo, nil
}

// Checkout clones the upstream opentelemetry-go repositories.
func Checkout() ([]RepoInfo, error) {
	log := conf.NewLog()
	env := conf.NewEnv()

	workDir, err := env.WorkDir()
	if err != nil {
		log.WithErrorMsg(err, "Failed to resolve cwd")
		return nil, err
	}

	cloneDir := filepath.Join(workDir, cwd)
	if err := os.MkdirAll(cloneDir, perms); err != nil {
		log.WithErrorMsg(err, "Failed to create clone directory", "dir", cloneDir, "perms", perms)
		return nil, err
	}

	var repoInfos []RepoInfo
	var errs error
	for _, repoURL := range repos {
		repoInfo, err := sync(repoURL, cloneDir, log)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		repoInfos = append(repoInfos, *repoInfo)
	}
	return repoInfos, errs
}
