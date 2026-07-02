package templatemgr

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
)

const (
	gitListTimeout  = 45 * time.Second
	gitCloneTimeout = 3 * time.Minute
)

// GitImportOptions configures importing a theme from a git repository.
type GitImportOptions struct {
	TemplateDir string
	RepoURL     string
	Branch      string
	ThemeName   string
	Label       string
	Type        string
	Author      string
	Description string
}

// ListGitBranches returns remote branch names for a repository URL.
func ListGitBranches(ctx context.Context, repoURL string) ([]string, error) {
	repoURL, err := NormalizeGitURL(repoURL)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, gitListTimeout)
	defer cancel()

	type result struct {
		branches []string
		err      error
	}
	done := make(chan result, 1)
	go func() {
		branches, err := listGitBranches(repoURL)
		done <- result{branches: branches, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("listing branches timed out")
	case res := <-done:
		return res.branches, res.err
	}
}

func listGitBranches(repoURL string) ([]string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list remote branches: %w", err)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, ref := range refs {
		if !ref.Name().IsBranch() {
			continue
		}
		name := ref.Name().Short()
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no branches found in repository")
	}
	sort.Strings(out)
	return out, nil
}

// ImportGitTheme clones a branch into template_dir/{theme} without leaving a .git directory.
func ImportGitTheme(ctx context.Context, opts GitImportOptions) error {
	repoURL, err := NormalizeGitURL(opts.RepoURL)
	if err != nil {
		return err
	}
	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		return fmt.Errorf("branch is required")
	}
	themeName := content.Slugify(strings.TrimSpace(opts.ThemeName))
	if themeName == "" {
		themeName = ThemeNameFromRepoURL(repoURL)
	}
	if err := themes.ValidateName(themeName); err != nil {
		return err
	}
	templateDir := strings.TrimSpace(opts.TemplateDir)
	if templateDir == "" {
		return fmt.Errorf("template directory is not configured")
	}
	themeRoot := themes.Root(templateDir, themeName)
	if _, err := os.Stat(themeRoot); err == nil {
		return fmt.Errorf("theme %q already exists", themeName)
	} else if !os.IsNotExist(err) {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, gitCloneTimeout)
	defer cancel()

	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		done <- result{err: cloneBranchFlat(repoURL, branch, themeRoot)}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("git clone timed out")
	case res := <-done:
		if res.err != nil {
			return res.err
		}
	}

	meta, err := templatemeta.Load(themeRoot)
	if err != nil {
		return err
	}
	if !packMetaConfigured(meta) {
		meta = templatemeta.DefaultPackMeta()
	}
	if label := strings.TrimSpace(opts.Label); label != "" {
		meta.Name = label
	} else if strings.TrimSpace(meta.Name) == "" {
		meta.Name = themeName
	}
	if themeType := strings.TrimSpace(opts.Type); themeType != "" {
		meta.Type = themeType
	}
	if author := strings.TrimSpace(opts.Author); author != "" {
		meta.Author = author
	}
	if desc := strings.TrimSpace(opts.Description); desc != "" {
		meta.Description = desc
	}
	if strings.TrimSpace(meta.Status) == "" {
		meta.Status = templatemeta.StatusActive
	}
	return templatemeta.Save(themeRoot, meta)
}

func packMetaConfigured(meta templatemeta.PackMeta) bool {
	return strings.TrimSpace(meta.Name) != "" ||
		strings.TrimSpace(meta.Author) != "" ||
		strings.TrimSpace(meta.Description) != "" ||
		strings.TrimSpace(meta.Version) != "" ||
		len(meta.Groups) > 0
}

func cloneBranchFlat(repoURL, branch, themeRoot string) error {
	tempDir, err := os.MkdirTemp("", "cannon-git-import-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	cloneDir := filepath.Join(tempDir, "repo")
	if _, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Depth:         1,
	}); err != nil {
		return fmt.Errorf("clone repository: %w", err)
	}

	sourceRoot, err := resolveThemeSourceRoot(cloneDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(themeRoot, 0o755); err != nil {
		return err
	}
	return copyFlatClone(sourceRoot, themeRoot)
}

// resolveThemeSourceRoot returns the directory whose files should become the theme root.
// If the repository contains a single top-level directory, its contents are promoted.
func resolveThemeSourceRoot(cloneDir string) (string, error) {
	entries, err := os.ReadDir(cloneDir)
	if err != nil {
		return "", err
	}
	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		filtered = append(filtered, entry)
	}
	if len(filtered) == 1 && filtered[0].IsDir() {
		return filepath.Join(cloneDir, filtered[0].Name()), nil
	}
	return cloneDir, nil
}

func copyFlatClone(src, dest string) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target, entry.Type())
	})
}

func copyFile(src, dest string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// NormalizeGitURL validates and normalizes a git remote URL.
func NormalizeGitURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("repository URL is required")
	}
	if strings.HasPrefix(raw, "git@") {
		if !strings.Contains(raw, ":") {
			return "", fmt.Errorf("invalid git SSH URL")
		}
		return raw, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid repository URL")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		if strings.TrimSpace(parsed.Host) == "" {
			return "", fmt.Errorf("invalid repository URL")
		}
		return parsed.String(), nil
	default:
		return "", fmt.Errorf("repository URL must use http, https, or git@host:path")
	}
}

// ThemeNameFromRepoURL derives a theme folder name from a git remote URL.
func ThemeNameFromRepoURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if strings.Contains(raw, ":") && !strings.Contains(raw, "://") {
		if idx := strings.LastIndex(raw, ":"); idx >= 0 && idx < len(raw)-1 {
			raw = raw[idx+1:]
		}
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Path != "" {
		raw = parsed.Path
	}
	raw = strings.Trim(raw, "/")
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		raw = raw[idx+1:]
	}
	name := content.Slugify(raw)
	if name == "" {
		return "imported-theme"
	}
	return name
}
