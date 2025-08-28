// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package offline provides offline testing support for package management.
package offline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/janderssonse/karei/internal/domain"
)

// PackageDB provides a complete offline package database for testing.
type PackageDB struct {
	packages       map[string]PackageMetadata
	dependencies   map[string][]string
	repositories   map[string]RepositoryInfo
	githubReleases map[string]GitHubRelease
	flatpaks       map[string]FlatpakInfo
	customScripts  map[string]ScriptInfo
	verbose        bool
}

// PackageMetadata contains comprehensive package information.
type PackageMetadata struct {
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Description  string               `json:"description"`
	Size         int64                `json:"size"`
	Architecture string               `json:"architecture"`
	Section      string               `json:"section"`
	Priority     string               `json:"priority"`
	Maintainer   string               `json:"maintainer"`
	Dependencies []string             `json:"dependencies"`
	Available    bool                 `json:"available"`
	Method       domain.InstallMethod `json:"method"`
	Source       string               `json:"source"`
	ExtraData    map[string]any       `json:"extra_data,omitempty"`
}

// RepositoryInfo contains repository metadata.
type RepositoryInfo struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Components  []string `json:"components"`
	Enabled     bool     `json:"enabled"`
	GPGVerify   bool     `json:"gpg_verify"`
}

// GitHubRelease contains GitHub release information.
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	CreatedAt   string        `json:"created_at"`
	PublishedAt string        `json:"published_at"`
	Body        string        `json:"body"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset contains asset information.
type GitHubAsset struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	DownloadCount      int    `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// FlatpakInfo contains Flatpak application information.
type FlatpakInfo struct {
	Name           string   `json:"name"`
	ID             string   `json:"id"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Size           int64    `json:"size"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtime_version"`
	SDK            string   `json:"sdk"`
	Permissions    []string `json:"permissions"`
	Remote         string   `json:"remote"`
	Branch         string   `json:"branch"`
	Available      bool     `json:"available"`
}

// ScriptInfo contains custom script information.
type ScriptInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Commands    []string          `json:"commands"`
	PreReqs     []string          `json:"pre_reqs"`
	PostInstall []string          `json:"post_install"`
	Variables   map[string]string `json:"variables"`
}

// NewPackageDB creates a new offline package database.
func NewPackageDB(verbose bool) *PackageDB {
	return &PackageDB{
		packages:       make(map[string]PackageMetadata),
		dependencies:   make(map[string][]string),
		repositories:   make(map[string]RepositoryInfo),
		githubReleases: make(map[string]GitHubRelease),
		flatpaks:       make(map[string]FlatpakInfo),
		customScripts:  make(map[string]ScriptInfo),
		verbose:        verbose,
	}
}

// LoadFromFixtures loads package data from test fixtures.
func (db *PackageDB) LoadFromFixtures(fixtureDir string) error {
	// Load APT packages
	aptFile := filepath.Join(fixtureDir, "packages", "apt_cache", "available_packages.json")
	if err := db.loadAPTPackages(aptFile); err != nil {
		return fmt.Errorf("failed to load APT packages: %w", err)
	}

	// Load Flatpak packages
	flatpakFile := filepath.Join(fixtureDir, "packages", "flatpak_info", "available_flatpaks.json")
	if err := db.loadFlatpakPackages(flatpakFile); err != nil {
		return fmt.Errorf("failed to load Flatpak packages: %w", err)
	}

	// Load GitHub releases
	githubDir := filepath.Join(fixtureDir, "packages", "github_releases")
	if err := db.loadGitHubReleases(githubDir); err != nil {
		return fmt.Errorf("failed to load GitHub releases: %w", err)
	}

	// Load custom scripts
	if err := db.loadCustomScripts(); err != nil {
		return fmt.Errorf("failed to load custom scripts: %w", err)
	}

	if db.verbose {
		fmt.Printf("Loaded offline package database: %d packages, %d flatpaks, %d GitHub releases\n",
			len(db.packages), len(db.flatpaks), len(db.githubReleases))
	}

	return nil
}

// GetPackage returns package metadata.
func (db *PackageDB) GetPackage(name string) (PackageMetadata, bool) {
	pkg, exists := db.packages[name]

	return pkg, exists
}

// GetAllPackages returns all packages.
func (db *PackageDB) GetAllPackages() map[string]PackageMetadata {
	result := make(map[string]PackageMetadata)
	for k, v := range db.packages {
		result[k] = v
	}

	return result
}

// GetPackagesByMethod returns packages by installation method.
func (db *PackageDB) GetPackagesByMethod(method domain.InstallMethod) []PackageMetadata {
	var packages []PackageMetadata
	for _, pkg := range db.packages {
		if pkg.Method == method {
			packages = append(packages, pkg)
		}
	}

	return packages
}

// GetDependencies returns package dependencies.
func (db *PackageDB) GetDependencies(packageName string) []string {
	deps, exists := db.dependencies[packageName]
	if !exists {
		return []string{}
	}

	return deps
}

// GetGitHubRelease returns GitHub release information.
func (db *PackageDB) GetGitHubRelease(repo string) (GitHubRelease, bool) {
	release, exists := db.githubReleases[repo]

	return release, exists
}

// GetFlatpak returns Flatpak information.
func (db *PackageDB) GetFlatpak(id string) (FlatpakInfo, bool) {
	flatpak, exists := db.flatpaks[id]

	return flatpak, exists
}

// GetCustomScript returns custom script information.
func (db *PackageDB) GetCustomScript(name string) (ScriptInfo, bool) {
	script, exists := db.customScripts[name]

	return script, exists
}

// IsPackageAvailable checks if a package is available.
func (db *PackageDB) IsPackageAvailable(name string) bool {
	pkg, exists := db.packages[name]

	return exists && pkg.Available
}

// SearchPackages searches for packages by name or description.
func (db *PackageDB) SearchPackages(query string) []PackageMetadata {
	var results []PackageMetadata

	query = strings.ToLower(query)

	for _, pkg := range db.packages {
		if strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.Description), query) {
			results = append(results, pkg)
		}
	}

	return results
}

// GetStatistics returns database statistics.
func (db *PackageDB) GetStatistics() map[string]any {
	stats := make(map[string]any)

	stats["total_packages"] = len(db.packages)
	stats["github_releases"] = len(db.githubReleases)
	stats["flatpaks"] = len(db.flatpaks)
	stats["custom_scripts"] = len(db.customScripts)
	stats["repositories"] = len(db.repositories)

	// Count by method
	methodCounts := make(map[string]int)
	for _, pkg := range db.packages {
		methodCounts[string(pkg.Method)]++
	}

	stats["by_method"] = methodCounts

	return stats
}

// ValidateDatabase performs consistency checks.
func (db *PackageDB) ValidateDatabase() []string {
	var errors []string

	// Check for missing dependencies
	for pkgName, deps := range db.dependencies {
		for _, dep := range deps {
			if !db.IsPackageAvailable(dep) {
				errors = append(errors, fmt.Sprintf("Package %s depends on missing package %s", pkgName, dep))
			}
		}
	}

	// Check GitHub releases have Linux assets
	for repo, release := range db.githubReleases {
		hasLinuxAsset := false

		for _, asset := range release.Assets {
			if strings.Contains(strings.ToLower(asset.Name), "linux") {
				hasLinuxAsset = true

				break
			}
		}

		if !hasLinuxAsset {
			errors = append(errors, fmt.Sprintf("GitHub release %s has no Linux assets", repo))
		}
	}

	return errors
}

// ExportToJSON exports the database to JSON format.
func (db *PackageDB) ExportToJSON(filename string) error {
	data := map[string]any{
		"packages":        db.packages,
		"dependencies":    db.dependencies,
		"repositories":    db.repositories,
		"github_releases": db.githubReleases,
		"flatpaks":        db.flatpaks,
		"custom_scripts":  db.customScripts,
		"statistics":      db.GetStatistics(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644) //nolint:gosec
}

// loadAPTPackages loads APT package information.
func (db *PackageDB) loadAPTPackages(filename string) error {
	data, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		return err
	}

	var aptData struct {
		Packages     map[string]PackageMetadata `json:"packages"`
		Repositories map[string]RepositoryInfo  `json:"repositories"`
	}

	if err := json.Unmarshal(data, &aptData); err != nil {
		return err
	}

	for name, pkg := range aptData.Packages {
		pkg.Method = domain.MethodAPT
		pkg.Source = name
		db.packages[name] = pkg
		db.dependencies[name] = pkg.Dependencies
	}

	for name, repo := range aptData.Repositories {
		db.repositories[name] = repo
	}

	return nil
}

// loadFlatpakPackages loads Flatpak package information.
func (db *PackageDB) loadFlatpakPackages(filename string) error {
	data, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		return err
	}

	var flatpakData struct {
		Flatpaks map[string]FlatpakInfo    `json:"flatpaks"`
		Remotes  map[string]RepositoryInfo `json:"remotes"`
	}

	if err := json.Unmarshal(data, &flatpakData); err != nil {
		return err
	}

	for flatpakID, flatpak := range flatpakData.Flatpaks {
		db.flatpaks[flatpakID] = flatpak

		// Also add as package
		pkg := PackageMetadata{
			Name:        flatpak.Name,
			Version:     flatpak.Version,
			Description: flatpak.Description,
			Size:        flatpak.Size,
			Available:   flatpak.Available,
			Method:      domain.MethodFlatpak,
			Source:      flatpak.ID,
		}
		db.packages[flatpakID] = pkg
	}

	for name, remote := range flatpakData.Remotes {
		db.repositories["flatpak-"+name] = remote
	}

	return nil
}

// loadGitHubReleases loads GitHub release information.
func (db *PackageDB) loadGitHubReleases(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Extract repo name from filename
		repoName := strings.TrimSuffix(file.Name(), "_latest.json")
		repoName = strings.ReplaceAll(repoName, "_", "/")

		filePath := filepath.Join(dir, file.Name())

		data, err := os.ReadFile(filePath) //nolint:gosec
		if err != nil {
			continue
		}

		var release GitHubRelease
		if err := json.Unmarshal(data, &release); err != nil {
			continue
		}

		db.githubReleases[repoName] = release

		// Extract package name from repo
		parts := strings.Split(repoName, "/")
		if len(parts) >= 2 {
			packageName := parts[1]
			pkg := PackageMetadata{
				Name:        packageName,
				Version:     strings.TrimPrefix(release.TagName, "v"),
				Description: "GitHub release: " + release.Name,
				Available:   true,
				Method:      domain.MethodGitHub,
				Source:      repoName,
			}

			// Find Linux asset for size
			for _, asset := range release.Assets {
				if strings.Contains(asset.Name, "Linux") || strings.Contains(asset.Name, "linux") {
					pkg.Size = asset.Size

					break
				}
			}

			db.packages[packageName] = pkg
		}
	}

	return nil
}

// loadCustomScripts loads custom installation scripts.
func (db *PackageDB) loadCustomScripts() error { //nolint:unparam
	scripts := map[string]ScriptInfo{
		"fastfetch-install": {
			Name:        "fastfetch",
			Description: "System information display tool installation",
			Commands: []string{
				"sudo add-apt-repository -y ppa:zhangsongcui3371/fastfetch",
				"sudo apt update -y",
				"sudo apt install -y fastfetch",
			},
			PreReqs: []string{"add-apt-repository", "apt"},
		},
		"typora-install": {
			Name:        "typora",
			Description: "Markdown editor installation",
			Commands: []string{
				"wget -qO - https://typora.io/linux/public-key.asc | sudo tee /etc/apt/trusted.gpg.d/typora.asc",
				"sudo add-apt-repository -y 'deb https://typora.io/linux ./'",
				"sudo apt update -y",
				"sudo apt install -y typora",
			},
			PreReqs: []string{"wget", "apt"},
			PostInstall: []string{
				"mkdir -p ~/.config/Typora/themes",
			},
		},
		"mise-install": {
			Name:        "mise",
			Description: "Fast polyglot tool version manager installation",
			Commands: []string{
				"curl https://mise.jdx.dev/install.sh | sh",
				"echo '~/.local/bin/mise activate fish | source' >> ~/.config/fish/config.fish",
			},
			PreReqs: []string{"curl"},
		},
	}

	for name, script := range scripts {
		db.customScripts[name] = script

		// Add as package
		pkg := PackageMetadata{
			Name:        script.Name,
			Version:     "latest",
			Description: script.Description,
			Available:   true,
			Method:      domain.MethodScript,
			Source:      name,
		}
		db.packages[script.Name] = pkg
	}

	return nil
}
