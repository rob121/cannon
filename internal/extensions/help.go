package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

// HelpArticle is a single help document from an extension.
type HelpArticle struct {
	Extension string
	Path      string
	Title     string
}

// HelpSection groups help articles for one extension.
type HelpSection struct {
	Extension string
	Label     string
	Articles  []HelpArticle
}

type helpEntry struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type helpListResponse struct {
	Help json.RawMessage `json:"help"`
}

// HelpIndex returns help articles from all active extensions with a help capability.
func (m *Manager) HelpIndex(ctx context.Context) ([]HelpSection, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	menuNames := map[string]string{}
	var rows []models.Extension
	if err := db.Select("name", "menu_name").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		menuNames[row.Name] = row.MenuName
	}

	m.mu.RLock()
	runtimes := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Help != "" && rt.Model.Status == models.StatusActive {
			runtimes = append(runtimes, rt)
		}
	}
	m.mu.RUnlock()
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].Model.Sort < runtimes[j].Model.Sort })

	sections := make([]HelpSection, 0, len(runtimes))
	for _, rt := range runtimes {
		articles, err := m.fetchHelpList(rt)
		if err != nil || len(articles) == 0 {
			continue
		}
		label := rt.Model.Name
		if v := strings.TrimSpace(menuNames[rt.Model.Name]); v != "" {
			label = v
		}
		sections = append(sections, HelpSection{
			Extension: rt.Model.Name,
			Label:     label,
			Articles:  articles,
		})
	}
	return sections, nil
}

// FetchHelpArticle loads markdown for an extension help path.
func (m *Manager) FetchHelpArticle(extensionName, articlePath string) (string, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Help == "" {
		return "", fmt.Errorf("extension %s has no help capability", extensionName)
	}
	path := normalizeHelpPath(articlePath)
	resp, err := m.do(rt.Model.Socket, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("help article %s: status %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (m *Manager) fetchHelpList(rt *Runtime) ([]HelpArticle, error) {
	path := "/" + strings.TrimPrefix(rt.Capabilities.Help, "/")
	resp, err := m.do(rt.Model.Socket, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("help index: status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload helpListResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	entries, err := parseHelpList(payload.Help)
	if err != nil {
		return nil, err
	}
	out := make([]HelpArticle, 0, len(entries))
	for _, entry := range entries {
		out = append(out, HelpArticle{
			Extension: rt.Model.Name,
			Path:      entry.Path,
			Title:     entry.Title,
		})
	}
	return out, nil
}

func parseHelpList(raw json.RawMessage) ([]helpEntry, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty help list")
	}
	var paths []string
	if err := json.Unmarshal(raw, &paths); err == nil {
		out := make([]helpEntry, 0, len(paths))
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			out = append(out, helpEntry{Path: p, Title: helpTitleFromPath(p)})
		}
		return out, nil
	}
	var entries []helpEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	out := make([]helpEntry, 0, len(entries))
	for _, entry := range entries {
		entry.Path = strings.TrimSpace(entry.Path)
		if entry.Path == "" {
			continue
		}
		if strings.TrimSpace(entry.Title) == "" {
			entry.Title = helpTitleFromPath(entry.Path)
		}
		out = append(out, entry)
	}
	return out, nil
}

func helpTitleFromPath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "Help"
	}
	parts := strings.Split(path, "/")
	last := parts[len(parts)-1]
	last = strings.ReplaceAll(last, "-", " ")
	last = strings.ReplaceAll(last, "_", " ")
	if last == "" {
		return "Help"
	}
	return strings.ToUpper(last[:1]) + last[1:]
}

func normalizeHelpPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// ExtensionFolderURL builds the admin URL for browsing an extension help folder.
func ExtensionFolderURL(extensionName string) string {
	return "/admin/help/extensions/" + extensionName
}

// HelpArticleURL builds the admin URL for an extension help article.
func HelpArticleURL(extensionName, articlePath string) string {
	path := normalizeHelpPath(articlePath)
	suffix := strings.TrimPrefix(path, "/")
	if suffix == "" {
		return "/admin/help/extensions/" + extensionName
	}
	return "/admin/help/extensions/" + extensionName + "/" + suffix
}

// HelpArticlePathFromParts reconstructs the extension help path from URL segments.
func HelpArticlePathFromParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return normalizeHelpPath(strings.Join(parts, "/"))
}
