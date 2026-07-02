package help

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed docs/**/*.md
var docsFS embed.FS

// Article is a built-in help document.
type Article struct {
	Folder string
	Slug   string
	Title  string
}

// Section groups built-in help articles by folder.
type Section struct {
	Folder   string
	Label    string
	Articles []Article
}

// Index returns built-in help sections sorted by folder then article slug.
func Index() ([]Section, error) {
	byFolder := map[string][]Article{}
	err := fs.WalkDir(docsFS, "docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel("docs", path)
		if err != nil {
			return err
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) != 2 {
			return nil
		}
		folder, slug := parts[0], strings.TrimSuffix(parts[1], ".md")
		raw, err := docsFS.ReadFile(path)
		if err != nil {
			return err
		}
		byFolder[folder] = append(byFolder[folder], Article{
			Folder: folder,
			Slug:   slug,
			Title:  titleFromMarkdown(slug, string(raw)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	folders := make([]string, 0, len(byFolder))
	for folder := range byFolder {
		folders = append(folders, folder)
	}
	sort.Strings(folders)

	sections := make([]Section, 0, len(folders))
	for _, folder := range folders {
		articles := byFolder[folder]
		sort.Slice(articles, func(i, j int) bool { return articles[i].Slug < articles[j].Slug })
		sections = append(sections, Section{
			Folder:   folder,
			Label:    folderLabel(folder),
			Articles: articles,
		})
	}
	return sections, nil
}

// Fetch returns markdown for a built-in help article.
func Fetch(folder, slug string) (string, error) {
	folder = strings.Trim(folder, "/")
	slug = strings.Trim(slug, "/")
	if folder == "" || slug == "" {
		return "", fmt.Errorf("help article not found")
	}
	path := filepath.Join("docs", folder, slug+".md")
	raw, err := docsFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("help article not found")
	}
	return string(raw), nil
}

// FolderURL builds the admin URL for browsing a built-in help folder.
func FolderURL(folder string) string {
	return "/admin/help/" + folder
}

// ExtensionsRootURL is the admin URL for the extension help folder listing.
func ExtensionsRootURL() string {
	return "/admin/help/extensions"
}

// FindSection locates a built-in help folder by id.
func FindSection(sections []Section, folder string) (*Section, bool) {
	for i := range sections {
		if sections[i].Folder == folder {
			return &sections[i], true
		}
	}
	return nil, false
}

// ArticleURL builds the admin URL for a built-in help article.
func ArticleURL(folder, slug string) string {
	return "/admin/help/" + folder + "/" + slug
}

// FirstArticle returns the first indexed article, if any.
func FirstArticle(sections []Section) *Article {
	for _, sec := range sections {
		if len(sec.Articles) > 0 {
			article := sec.Articles[0]
			return &article
		}
	}
	return nil
}

// FindArticle locates an article in the index.
func FindArticle(sections []Section, folder, slug string) (*Article, bool) {
	for _, sec := range sections {
		if sec.Folder != folder {
			continue
		}
		for _, article := range sec.Articles {
			if article.Slug == slug {
				return &article, true
			}
		}
	}
	return nil, false
}

func titleFromMarkdown(slug, content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
		if line != "" {
			break
		}
	}
	return slugLabel(slug)
}

func folderLabel(folder string) string {
	return slugLabel(folder)
}

func slugLabel(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return "Help"
	}
	parts := strings.Fields(value)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
