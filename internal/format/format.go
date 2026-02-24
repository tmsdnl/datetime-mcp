package format

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	repoOwner    = "tmsdnl"
	repoName     = "datetime-mcp"
	formatsPath  = "formats"
	githubAPIBase = "https://api.github.com"
)

// Status describes the outcome of a format file operation.
type Status int

const (
	StatusInstalled Status = iota
	StatusUpdated
	StatusSkipped // already exists, not overwritten (install only)
	StatusRemoved
	StatusNotFound
	StatusError
)

// Result holds the outcome of one format file operation.
type Result struct {
	Name   string
	Path   string
	Status Status
	DryRun bool
	Err    error
}

// Config holds options for format operations.
type Config struct {
	Dir     string // override XDG dir; empty = default
	Version string // binary version for ref selection
	DryRun  bool
}

func defaultDir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "datetime-mcp", "formats")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "datetime-mcp", "formats")
}

func refFromVersion(v string) string {
	if v == "" || v == "dev" || v == "unknown" {
		return "main"
	}
	return "v" + v
}

type remoteFile struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func fetchFileList(version string) ([]remoteFile, error) {
	ref := refFromVersion(version)
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		githubAPIBase, repoOwner, repoName, formatsPath, ref)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "datetime-mcp/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching file list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// If the versioned ref fails, fall back to main.
		if ref != "main" {
			return fetchFileListRef("main", version)
		}
		return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, url)
	}

	var all []remoteFile
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("parsing file list: %w", err)
	}

	var yaml []remoteFile
	for _, f := range all {
		if f.Type == "file" && (strings.HasSuffix(f.Name, ".yaml") || strings.HasSuffix(f.Name, ".yml")) {
			yaml = append(yaml, f)
		}
	}
	return yaml, nil
}

func fetchFileListRef(ref, version string) ([]remoteFile, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		githubAPIBase, repoOwner, repoName, formatsPath, ref)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "datetime-mcp/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching file list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, url)
	}

	var all []remoteFile
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("parsing file list: %w", err)
	}

	var yaml []remoteFile
	for _, f := range all {
		if f.Type == "file" && (strings.HasSuffix(f.Name, ".yaml") || strings.HasSuffix(f.Name, ".yml")) {
			yaml = append(yaml, f)
		}
	}
	return yaml, nil
}

func download(url, version string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "datetime-mcp/"+version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func nameWithoutExt(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// Install downloads format files that do not already exist locally.
func Install(cfg Config) []Result {
	dir := cfg.Dir
	if dir == "" {
		dir = defaultDir()
	}

	files, err := fetchFileList(cfg.Version)
	if err != nil {
		return []Result{{Name: "fetch", Status: StatusError, Err: err}}
	}

	if !cfg.DryRun {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return []Result{{Name: "mkdir", Status: StatusError, Err: err}}
		}
	}

	var results []Result
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		name := nameWithoutExt(f.Name)

		if _, err := os.Stat(path); err == nil {
			results = append(results, Result{Name: name, Path: path, Status: StatusSkipped})
			continue
		}

		if cfg.DryRun {
			results = append(results, Result{Name: name, Path: path, Status: StatusInstalled, DryRun: true})
			continue
		}

		data, err := download(f.DownloadURL, cfg.Version)
		if err != nil {
			results = append(results, Result{Name: name, Path: path, Status: StatusError, Err: err})
			continue
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			results = append(results, Result{Name: name, Path: path, Status: StatusError, Err: err})
			continue
		}

		results = append(results, Result{Name: name, Path: path, Status: StatusInstalled})
	}
	return results
}

// Update downloads and overwrites all managed format files.
func Update(cfg Config) []Result {
	dir := cfg.Dir
	if dir == "" {
		dir = defaultDir()
	}

	files, err := fetchFileList(cfg.Version)
	if err != nil {
		return []Result{{Name: "fetch", Status: StatusError, Err: err}}
	}

	if !cfg.DryRun {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return []Result{{Name: "mkdir", Status: StatusError, Err: err}}
		}
	}

	var results []Result
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		name := nameWithoutExt(f.Name)

		if cfg.DryRun {
			results = append(results, Result{Name: name, Path: path, Status: StatusUpdated, DryRun: true})
			continue
		}

		data, err := download(f.DownloadURL, cfg.Version)
		if err != nil {
			results = append(results, Result{Name: name, Path: path, Status: StatusError, Err: err})
			continue
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			results = append(results, Result{Name: name, Path: path, Status: StatusError, Err: err})
			continue
		}

		results = append(results, Result{Name: name, Path: path, Status: StatusUpdated})
	}
	return results
}

// Uninstall removes local format files whose names match the upstream repo.
// User-created files with different names are not touched.
func Uninstall(cfg Config) []Result {
	dir := cfg.Dir
	if dir == "" {
		dir = defaultDir()
	}

	files, err := fetchFileList(cfg.Version)
	if err != nil {
		return []Result{{Name: "fetch", Status: StatusError, Err: err}}
	}

	var results []Result
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		name := nameWithoutExt(f.Name)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			results = append(results, Result{Name: name, Path: path, Status: StatusNotFound})
			continue
		}

		if cfg.DryRun {
			results = append(results, Result{Name: name, Path: path, Status: StatusRemoved, DryRun: true})
			continue
		}

		if err := os.Remove(path); err != nil {
			results = append(results, Result{Name: name, Path: path, Status: StatusError, Err: err})
			continue
		}

		results = append(results, Result{Name: name, Path: path, Status: StatusRemoved})
	}
	return results
}
