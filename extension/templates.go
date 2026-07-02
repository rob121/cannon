package extension

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/sprig/v3"
)

const templateOverridePrefix = "extension/"

// Templates resolves HTML templates embedded in an extension with optional
// site overrides from the Cannon template directory.
//
// A local template "contact/form.html" is overridden by
// "{template_dir}/extension/contact/form.html".
type Templates struct {
	embed fs.FS
	root  string
	dir   string
	funcs template.FuncMap

	mu     sync.RWMutex
	parsed map[string]*template.Template
}

// NewTemplates creates a template resolver rooted at root inside embed.
// Pass "." when HTML files live at the top level of embed.
func NewTemplates(embed fs.FS, root string) *Templates {
	return &Templates{
		embed:  embed,
		root:   strings.Trim(filepath.ToSlash(root), "/"),
		funcs:  sprig.FuncMap(),
		parsed: make(map[string]*template.Template),
	}
}

// WithFuncs returns a copy that uses the provided template functions.
func (t *Templates) WithFuncs(funcs template.FuncMap) *Templates {
	out := *t
	out.funcs = funcs
	out.parsed = make(map[string]*template.Template)
	return &out
}

// WithTemplateDir returns a copy pinned to a specific site template directory.
func (t *Templates) WithTemplateDir(dir string) *Templates {
	out := *t
	out.dir = strings.TrimSpace(dir)
	out.parsed = make(map[string]*template.Template)
	return &out
}

// TemplateOverridePath maps a local extension template path to the site override path.
func TemplateOverridePath(local string) (string, error) {
	local = normalizeTemplatePath(local)
	if local == "" {
		return "", fmt.Errorf("template path is required")
	}
	return templateOverridePrefix + local, nil
}

// List returns the embedded HTML templates available for site overrides.
func (t *Templates) List() ([]TemplateDefinition, error) {
	root := t.root
	if root == "" {
		root = "."
	}
	var out []TemplateDefinition
	err := fs.WalkDir(t.embed, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel := path
		if t.root != "" && t.root != "." {
			var relErr error
			rel, relErr = filepath.Rel(t.root, path)
			if relErr != nil {
				return relErr
			}
		}
		rel = normalizeTemplatePath(rel)
		if rel == "" {
			return nil
		}
		override, err := TemplateOverridePath(rel)
		if err != nil {
			return nil
		}
		def := TemplateDefinition{Path: rel, OverridePath: override}
		if info, err := entry.Info(); err == nil {
			def.Size = info.Size()
		}
		out = append(out, def)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// ReadEmbedded loads the embedded default template source, ignoring site overrides.
func (t *Templates) ReadEmbedded(name string) (content string, source string, err error) {
	name = normalizeTemplatePath(name)
	if name == "" {
		return "", "", fmt.Errorf("template path is required")
	}
	embedPath := t.embedPath(name)
	raw, err := fs.ReadFile(t.embed, embedPath)
	if err != nil {
		return "", "", fmt.Errorf("template not found: %s", name)
	}
	return string(raw), "embed:" + embedPath, nil
}

// Read loads template source, preferring a site override when present.
func (t *Templates) Read(name string) (content string, source string, err error) {
	name = normalizeTemplatePath(name)
	if name == "" {
		return "", "", fmt.Errorf("template path is required")
	}

	override, err := TemplateOverridePath(name)
	if err != nil {
		return "", "", err
	}
	if dir, err := t.templateDir(); err == nil && dir != "" {
		path := filepath.Join(dir, filepath.FromSlash(override))
		raw, err := os.ReadFile(path)
		if err == nil {
			return string(raw), path, nil
		}
		if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("read override template: %w", err)
		}
	}

	return t.ReadEmbedded(name)
}

// Execute renders a template with data.
func (t *Templates) Execute(name string, data any) (string, error) {
	tmpl, err := t.Parse(name)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

// Parse returns a parsed template, using a per-name cache.
func (t *Templates) Parse(name string) (*template.Template, error) {
	name = normalizeTemplatePath(name)
	if name == "" {
		return nil, fmt.Errorf("template path is required")
	}

	t.mu.RLock()
	if tmpl, ok := t.parsed[name]; ok {
		t.mu.RUnlock()
		return tmpl, nil
	}
	t.mu.RUnlock()

	content, _, err := t.Read(name)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if tmpl, ok := t.parsed[name]; ok {
		return tmpl, nil
	}

	base := template.New(filepath.Base(name)).Funcs(t.funcs)
	tmpl, err := base.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	t.parsed[name] = tmpl
	return tmpl, nil
}

func (t *Templates) templateDir() (string, error) {
	if t.dir != "" {
		return t.dir, nil
	}
	site, err := SiteConfig()
	if err != nil {
		return "", err
	}
	t.dir = strings.TrimSpace(site.TemplateDir)
	return t.dir, nil
}

func (t *Templates) embedPath(name string) string {
	name = normalizeTemplatePath(name)
	if t.root == "" || t.root == "." {
		return name
	}
	return t.root + "/" + name
}

func normalizeTemplatePath(name string) string {
	name = strings.TrimSpace(filepath.ToSlash(name))
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, templateOverridePrefix) {
		name = strings.TrimPrefix(name, templateOverridePrefix)
	}
	if !strings.HasSuffix(strings.ToLower(name), ".html") {
		return ""
	}
	if strings.Contains(name, "..") {
		return ""
	}
	return name
}
