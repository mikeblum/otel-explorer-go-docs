package repo

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mikeblum/otel-explorer-go-docs/conf"
)

const (
	cwd          = ".repo"
	perms        = 0755
	shaLength    = 8
	manifestPath = "registry/registry_manifest.yaml"

	RepoGo      = "opentelemetry-go"
	RepoContrib = "opentelemetry-go-contrib"
	RepoSemconv = "semantic-conventions"
)

var repos = []string{
	"git@github.com:open-telemetry/opentelemetry-go-contrib.git",
}

type RepoInfo struct {
	Name    string
	Path    string
	Head    string
	SHA     string
	Message string
}

const semconvOTEL = "otel"
const semconvPath = "https://github.com/open-telemetry/semantic-conventions/archive/refs/tags/v1.38.0.zip[model]"
const semconvVersion = "0.1.0"

type RegistryManifest struct {
	Name          string               `yaml:"name"`
	Description   string               `yaml:"description"`
	Version       string               `yaml:"semconv_version"`
	SchemaBaseURL url.URL              `yaml:"schema_base_url"`
	Dependencies  []RegistryDependency `yaml:"dependencies"`
}

func NewRegistry() *RegistryManifest {
	return &RegistryManifest{
		Name:        "otel-explorer-go-docs",
		Description: "OTel Explorer Golang Instrumentation",
		Version:     semconvVersion,
		Dependencies: []RegistryDependency{
			{
				Name:         semconvOTEL,
				RegistryPath: semconvPath,
			},
		},
	}
}

func (r *RegistryManifest) SemConv() (*RegistryDependency, error) {
	var dep *RegistryDependency
	for _, d := range r.Dependencies {
		if d.Name == semconvOTEL {
			dep = &d
			break
		}
	}

	if dep == nil {
		return nil, fmt.Errorf("no semconv dependency found in manifest")
	}
	return dep, nil
}

type RegistryDependency struct {
	Name         string `yaml:"name"`
	RegistryPath string `yaml:"registry_path"`
}

// parseRegistryPath extracts URL and subdirectory from registry_path.
// Example: "https://github.com/.../v1.38.0.zip[model]" -> ("https://...zip", "model")
func (r *RegistryDependency) parseRegistryPath() (string, string) {
	if idx := strings.Index(r.RegistryPath, "["); idx != -1 {
		url := r.RegistryPath[:idx]
		subdir := strings.Trim(r.RegistryPath[idx:], "[]")
		return url, subdir
	}
	return r.RegistryPath, ""
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

func gitCommand(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func info(path string) (*RepoInfo, error) {
	head, err := gitCommand(path, "rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, err
	}

	sha, err := gitCommand(path, "log", "-1", "--format=%H")
	if err != nil {
		return nil, err
	}

	msg, err := gitCommand(path, "log", "-1", "--format=%s")
	if err != nil {
		return nil, err
	}

	return &RepoInfo{
		Head:    head,
		SHA:     sha[:shaLength],
		Message: strings.ReplaceAll(msg, "\n", " "),
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

// CheckoutSemconv downloads the semantic conventions registry from the manifest.
func CheckoutSemconv() (string, error) {
	log := conf.NewLog()
	env := conf.NewEnv()

	workDir, err := env.WorkDir()
	if err != nil {
		return "", err
	}

	manifest := NewRegistry()
	semconv, err := manifest.SemConv()
	if err != nil {
		return "", err
	}
	zipURL, subdir := semconv.parseRegistryPath()
	log.Info(RepoSemconv, "url", zipURL, "subdir", subdir)

	cloneDir := filepath.Join(workDir, cwd)
	semconvDir, err := downloadAndExtractZip(zipURL, subdir, cloneDir)
	if err != nil {
		return "", fmt.Errorf("failed to download semconv: %w", err)
	}

	return semconvDir, nil
}

// downloadAndExtractZip downloads a ZIP file and extracts a subdirectory.
func downloadAndExtractZip(zipURL, subdir, destDir string) (string, error) {
	resp, err := http.Get(zipURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download: %s", resp.Status)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", err
	}

	var extractedPath string
	for _, file := range zipReader.File {
		if !strings.Contains(file.Name, subdir+"/") {
			continue
		}

		targetPath := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, perms)
			if extractedPath == "" {
				extractedPath = targetPath
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), perms); err != nil {
			return "", err
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return "", err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return "", err
		}

		if extractedPath == "" {
			extractedPath = filepath.Dir(targetPath)
		}
	}

	return extractedPath, nil
}
