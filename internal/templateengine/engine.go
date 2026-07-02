package templateengine

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/hooks"
)

//go:embed default/*.html default/*.css default/*.svg default/controllers/*/*.html admin/*.html admin/admin.css admin/admin.js admin/cannon-icon.svg
var coreFS embed.FS

// BlockRenderer renders named template spaces via extensions.
type BlockRenderer func(name string) (template.HTML, error)

// BlockLenRenderer returns how many blocks are assigned to a template space.
type BlockLenRenderer func(name string) (int, error)

// Engine resolves templates with site precedence and block support.
type Engine struct {
	site     *config.SiteConfig
	funcs    template.FuncMap
	blocks   BlockRenderer
	hookCtx  context.Context
	mu       sync.RWMutex
	parsed   map[string]*template.Template
	adminSet *template.Template
}

// PageData is passed to layout templates.
type PageData struct {
	Title     string
	Subtitle  string
	ActiveNav string
	Main      template.HTML
}

// New creates a template engine for a site.
func New(site *config.SiteConfig, blocks BlockRenderer, blockLen BlockLenRenderer, extra template.FuncMap) *Engine {
	funcs := FuncMap(blocks, blockLen)
	for k, v := range extra {
		funcs[k] = v
	}
	return &Engine{
		site:   site,
		funcs:  funcs,
		blocks: blocks,
		parsed: make(map[string]*template.Template),
	}
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case int32:
		return int(n)
	default:
		return 0
	}
}

// SetHookContext attaches request context used when firing render hooks.
func (e *Engine) SetHookContext(ctx context.Context) {
	e.hookCtx = ctx
}

// Render executes a template with layout wrapping.
func (e *Engine) Render(w io.Writer, layout, page string, data any) error {
	if e.hookCtx != nil {
		out, err := hooks.Fire(e.hookCtx, hooks.OnBeforeRender, map[string]any{
			"layout": layout,
			"page":   page,
		})
		if err != nil {
			return err
		}
		if v, ok := out["layout"].(string); ok && v != "" {
			layout = v
		}
		if v, ok := out["page"].(string); ok && v != "" {
			page = v
		}
	}

	body, err := e.renderPage(page, data)
	if err != nil {
		return err
	}

	if e.hookCtx != nil {
		out, err := hooks.Fire(e.hookCtx, hooks.OnContentAfterDisplay, map[string]any{
			"layout": layout,
			"page":   page,
			"body":   string(body),
		})
		if err != nil {
			return err
		}
		if v, ok := out["body"].(string); ok {
			body = template.HTML(v)
		}
	}

	layoutData := map[string]any{"Main": body}
	if m, ok := data.(map[string]any); ok {
		for k, v := range m {
			layoutData[k] = v
		}
	}
	if err := e.renderLayout(w, layout, layoutData); err != nil {
		return err
	}
	if e.hookCtx != nil {
		_, _ = hooks.Fire(e.hookCtx, hooks.OnAfterRender, map[string]any{
			"layout": layout,
			"page":   page,
		})
	}
	return nil
}

// RenderController renders default/controllers/{controller}/{action}.html inside the layout.
func (e *Engine) RenderController(w io.Writer, layout, controllerID, actionID string, data any) error {
	page := ControllerTemplatePath(controllerID, actionID)
	return e.Render(w, layout, page, data)
}

// ControllerTemplatePath returns the conventional controller template path.
func ControllerTemplatePath(controllerID, actionID string) string {
	return "default/controllers/" + controllerID + "/" + actionID + ".html"
}

// RenderFragment renders a template without layout.
func (e *Engine) RenderFragment(w io.Writer, name string, data any) error {
	tmpl, err := e.parse(name)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// AdminAsset returns embedded admin static files.
func AdminAsset(name string) ([]byte, error) {
	return coreFS.ReadFile(filepath.Join("admin", name))
}

// SiteAsset returns embedded public site static files from default/.
func SiteAsset(name string) ([]byte, error) {
	return coreFS.ReadFile(filepath.Join("default", name))
}

// ReadBuiltin returns embedded HTML template source for admin/ or default/ paths.
func ReadBuiltin(name string) (string, error) {
	name = filepath.ToSlash(strings.TrimPrefix(name, "/"))
	if !strings.HasSuffix(strings.ToLower(name), ".html") {
		return "", fmt.Errorf("template not found: %s", name)
	}
	raw, err := coreFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("template not found: %s", name)
	}
	return string(raw), nil
}

// BuiltinTemplates lists embedded HTML templates that can be overridden.
func BuiltinTemplates() ([]string, error) {
	var names []string
	err := fs.WalkDir(coreFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return nil
		}
		names = append(names, filepath.ToSlash(path))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func (e *Engine) renderPage(name string, data any) (template.HTML, error) {
	tmpl, err := e.parse(name)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func (e *Engine) renderLayout(w io.Writer, name string, data any) error {
	tmpl, err := e.parse(name)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

func (e *Engine) parse(name string) (*template.Template, error) {
	if strings.HasPrefix(name, "admin/") {
		return e.parseAdmin(name)
	}

	e.mu.RLock()
	if tmpl, ok := e.parsed[name]; ok {
		e.mu.RUnlock()
		return tmpl, nil
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	if tmpl, ok := e.parsed[name]; ok {
		return tmpl, nil
	}

	content, src, err := e.readTemplate(name)
	if err != nil {
		return nil, err
	}

	base := template.New(filepath.Base(name)).Funcs(e.funcs)
	tmpl, err := base.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse template %s (%s): %w", name, src, err)
	}
	e.parsed[name] = tmpl
	return tmpl, nil
}

func (e *Engine) parseAdmin(name string) (*template.Template, error) {
	e.mu.RLock()
	if e.adminSet != nil {
		set := e.adminSet
		e.mu.RUnlock()
		if t := set.Lookup(filepath.Base(name)); t != nil {
			return t, nil
		}
		return nil, fmt.Errorf("admin template not found: %s", name)
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.adminSet != nil {
		if t := e.adminSet.Lookup(filepath.Base(name)); t != nil {
			return t, nil
		}
		return nil, fmt.Errorf("admin template not found: %s", name)
	}

	set := template.New("admin").Funcs(e.funcs)
	if err := e.parseAdminEmbed(set); err != nil {
		return nil, err
	}
	if e.site.TemplateDir != "" {
		if err := e.parseAdminDir(set, e.site.TemplateDir); err != nil {
			return nil, err
		}
	}
	e.adminSet = set

	t := set.Lookup(filepath.Base(name))
	if t == nil {
		return nil, fmt.Errorf("admin template not found: %s", name)
	}
	return t, nil
}

func (e *Engine) parseAdminEmbed(set *template.Template) error {
	entries, err := coreFS.ReadDir("admin")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		raw, err := coreFS.ReadFile("admin/" + entry.Name())
		if err != nil {
			return err
		}
		if err := parseAdminFile(set, entry.Name(), string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) parseAdminDir(set *template.Template, dir string) error {
	adminRoot := filepath.Join(dir, "admin")
	if _, err := os.Stat(adminRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(adminRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return parseAdminFile(set, entry.Name(), string(raw))
	})
}

// parseAdminFile registers a single admin HTML file in the template set.
// Fragment files (prefixed with _) and files containing {{define}} are merged
// into the set; page templates are registered under their filename so Lookup works.
func parseAdminFile(set *template.Template, name, content string) error {
	if strings.HasPrefix(name, "_") || strings.Contains(content, "{{define") {
		var err error
		set, err = set.Parse(content)
		if err != nil {
			return fmt.Errorf("parse admin/%s: %w", name, err)
		}
		return nil
	}
	if existing := set.Lookup(name); existing != nil {
		if _, err := existing.Parse(content); err != nil {
			return fmt.Errorf("parse admin/%s: %w", name, err)
		}
		return nil
	}
	if _, err := set.New(name).Parse(content); err != nil {
		return fmt.Errorf("parse admin/%s: %w", name, err)
	}
	return nil
}

func (e *Engine) readTemplate(name string) (string, string, error) {
	if e.site.TemplateDir != "" {
		path := filepath.Join(e.site.TemplateDir, name)
		if raw, err := os.ReadFile(path); err == nil {
			return string(raw), path, nil
		}
	}

	corePath := name
	if strings.HasPrefix(name, "admin/") || strings.HasPrefix(name, "default/") {
		if raw, err := coreFS.ReadFile(corePath); err == nil {
			return string(raw), "embed:" + corePath, nil
		}
	}

	if !strings.Contains(name, "/") {
		for _, prefix := range []string{"default/", "admin/"} {
			corePath = prefix + name
			if raw, err := coreFS.ReadFile(corePath); err == nil {
				return string(raw), "embed:" + corePath, nil
			}
		}
	}

	return "", "", fmt.Errorf("template not found: %s", name)
}

// ParseNested resolves {{template "path"}} definitions from site then default.
func (e *Engine) ParseNested(name string) (*template.Template, error) {
	return e.parse(name)
}

// CoreFS exposes embedded templates for copying at install.
func CoreFS() fs.FS {
	sub, _ := fs.Sub(coreFS, ".")
	return sub
}
