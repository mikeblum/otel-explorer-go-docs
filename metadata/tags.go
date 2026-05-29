package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	repoContrib = "open-telemetry/opentelemetry-go-contrib"

	githubAPI   = "https://api.github.com"
	moduleProxy = "https://proxy.golang.org"

	tagsPerPage = 100
)

type TagResolver struct {
	token  string
	client *http.Client
}

func NewTagResolver(githubToken string) *TagResolver {
	return &TagResolver{
		token:  githubToken,
		client: &http.Client{},
	}
}

// LatestModuleVersion returns the latest published version of a Go module
// using the Go module proxy, which is the canonical source for module metadata.
func (r *TagResolver) LatestModuleVersion(ctx context.Context, modulePath string) (string, error) {
	url := fmt.Sprintf("%s/%s/@latest", moduleProxy, modulePath)
	var result struct {
		Version string `json:"Version"`
	}
	if err := r.get(ctx, url, &result); err != nil {
		return "", err
	}
	return result.Version, nil
}

// ListContribModulePaths returns all instrumentation module paths from go-contrib
// by inspecting git tags. Useful for discovering modules without cloning the repo.
func (r *TagResolver) ListContribModulePaths(ctx context.Context) ([]string, error) {
	tags, err := r.listTags(ctx, repoContrib)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var paths []string
	for _, tag := range tags {
		// tags are like "instrumentation/net/http/otelhttp/v0.68.0"
		// strip the version suffix to get the module path
		if idx := strings.LastIndex(tag, "/v"); idx != -1 {
			path := tag[:idx]
			if !seen[path] {
				seen[path] = true
				paths = append(paths, path)
			}
		}
	}
	return paths, nil
}

func (r *TagResolver) listTags(ctx context.Context, repo string) ([]string, error) {
	var all []string
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/repos/%s/git/refs/tags?per_page=%d&page=%d",
			githubAPI, repo, tagsPerPage, page)
		var refs []struct {
			Ref string `json:"ref"`
		}
		if err := r.get(ctx, url, &refs); err != nil {
			return nil, err
		}
		if len(refs) == 0 {
			break
		}
		for _, ref := range refs {
			all = append(all, strings.TrimPrefix(ref.Ref, "refs/tags/"))
		}
		if len(refs) < tagsPerPage {
			break
		}
	}
	return all, nil
}

func (r *TagResolver) get(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
