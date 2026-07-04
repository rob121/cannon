package templateengine

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/captcha"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
	"github.com/rob121/cannon/internal/user"
)

//go:embed default/*.html default/error/*.html default/mail/*.html default/partials/auth/*.html default/partials/blocks/*.html default/partials/content/*.html default/partials/error-page.html default/partials/meta-tags.html default/partials/offline-notice.html default/partials/page-header.html default/partials/menu-nav.html default/partials/route-iframe.html default/controllers/*/*.html default/*.css default/*.svg admin/*.html admin/error/*.html admin/admin.css admin/admin.js admin/cannon-icon.svg
var coreFS embed.FS

// BlockRenderer renders named template spaces via extensions.
type BlockRenderer func(name string) (template.HTML, error)

// BlockLenRenderer returns how many blocks are assigned to a template space.
type BlockLenRenderer func(name string) (int, error)

// Engine resolves templates with site precedence and block support.
type Engine struct {
	site      *config.SiteConfig
	themes    themes.Selection
	funcs     template.FuncMap
	blocks    BlockRenderer
	hookCtx   context.Context
	mu        sync.RWMutex
	parsed    map[string]*template.Template
	adminSet  *template.Template
	adminRoot string
}

// PageData is passed to layout templates.
type PageData struct {
	Title     string
	Subtitle  string
	ActiveNav string
	Main      template.HTML
}

// New creates a template engine for a site.
func New(site *config.SiteConfig, sel themes.Selection, blocks BlockRenderer, blockLen BlockLenRenderer, extra template.FuncMap) *Engine {
	funcs := FuncMap(blocks, blockLen)
	for k, v := range extra {
		funcs[k] = v
	}
	sel = sel.Normalize()
	eng := &Engine{
		site:   site,
		themes: sel,
		funcs:  funcs,
		blocks: blocks,
		parsed: make(map[string]*template.Template),
	}
	if !themes.IsBuiltinAdmin(sel.Admin) && site != nil {
		eng.adminRoot = themes.Root(site.TemplateDir, sel.Admin)
	}
	return eng
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

// HookContext returns the active render context, if any.
func (e *Engine) HookContext() context.Context {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hookCtx
}

// Render executes a template with layout wrapping.
func (e *Engine) Render(w io.Writer, layout, page string, data any) error {
	return e.render(w, layout, page, data, 0)
}

// RenderWithStatus renders a template and sets the HTTP status after render hooks run.
func (e *Engine) RenderWithStatus(w io.Writer, status int, layout, page string, data any) error {
	return e.render(w, layout, page, data, status)
}

func (e *Engine) render(w io.Writer, layout, page string, data any, status int) error {
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
	if _, ok := layoutData["User"]; !ok && e.hookCtx != nil {
		if userScope, ok := user.RequestUser(e.hookCtx); ok {
			layoutData["User"] = userScope
		}
	}
	if _, ok := layoutData["HomeURL"]; !ok && e.hookCtx != nil {
		layoutData["HomeURL"] = sites.DefaultRoutePath(e.hookCtx)
	}
	if _, ok := layoutData["IsOffline"]; !ok && e.hookCtx != nil {
		if offline, err := settings.SiteOffline(e.hookCtx); err == nil {
			layoutData["IsOffline"] = offline
		}
	}
	var rendered bytes.Buffer
	if err := e.renderLayout(&rendered, layout, layoutData); err != nil {
		return err
	}
	output := rendered.Bytes()

	if e.hookCtx != nil {
		if expanded, err := captcha.ExpandHTML(e.hookCtx, string(output)); err == nil {
			output = []byte(expanded)
		}
	}

	if rw, ok := w.(interface{ Header() http.Header }); ok {
		if rw.Header().Get("Content-Type") == "" {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
	}
	if e.hookCtx != nil {
		out, err := hooks.Fire(e.hookCtx, hooks.OnAfterRender, map[string]any{
			"layout":  layout,
			"page":    page,
			"headers": writerHeaders(w),
			"body":    string(output),
		})
		if err != nil {
			return err
		}
		applyHookHeaders(w, out)
		if encoding, _ := out["body_encoding"].(string); strings.EqualFold(encoding, "base64") {
			if encoded, _ := out["body_base64"].(string); encoded != "" {
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil {
					return fmt.Errorf("decode onAfterRender body_base64: %w", err)
				}
				output = decoded
			}
		} else if v, ok := out["body"].(string); ok {
			output = []byte(v)
		}
	}
	writeResponseStatus(w, status)
	_, err = w.Write(output)
	return err
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
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	out := buf.String()
	if e.hookCtx != nil {
		if expanded, err := captcha.ExpandHTML(e.hookCtx, out); err == nil {
			out = expanded
		}
	}
	_, err = io.WriteString(w, out)
	return err
}

// AdminAsset returns embedded admin static files.
func AdminAsset(name string) ([]byte, error) {
	return coreFS.ReadFile(filepath.Join("admin", name))
}

// SiteAsset returns embedded public site static files from default/.
func SiteAsset(name string) ([]byte, error) {
	return coreFS.ReadFile(filepath.Join("default", name))
}

// ThemeAsset reads a file from a theme assets directory, falling back to embedded default assets.
func ThemeAsset(templateDir, theme, name string) ([]byte, string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		return nil, "", fmt.Errorf("asset name is required")
	}
	if !themes.IsBuiltinFrontend(theme) && strings.TrimSpace(templateDir) != "" {
		path := filepath.Join(themes.AssetsDir(templateDir, theme), name)
		if raw, err := os.ReadFile(path); err == nil {
			return raw, path, nil
		}
	}
	raw, err := SiteAsset(name)
	if err != nil {
		return nil, "", err
	}
	return raw, "embed:default/" + name, nil
}

// AdminThemeAsset reads from the active admin theme assets directory or embedded admin files.
func AdminThemeAsset(templateDir, theme, name string) ([]byte, string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		return nil, "", fmt.Errorf("asset name is required")
	}
	if !themes.IsBuiltinAdmin(theme) && strings.TrimSpace(templateDir) != "" {
		path := filepath.Join(themes.AssetsDir(templateDir, theme), name)
		if raw, err := os.ReadFile(path); err == nil {
			return raw, path, nil
		}
	}
	raw, err := AdminAsset(name)
	if err != nil {
		return nil, "", err
	}
	return raw, "embed:admin/" + name, nil
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
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	out := buf.Bytes()
	if isLayoutTemplate(name) && !bytes.Contains(out, []byte("site-offline-notice")) && layoutIsOffline(e, data) {
		if notice, err := e.renderOfflineNotice(data); err == nil && notice != "" {
			out = injectAfterBodyOpen(out, notice)
		}
	}
	_, err = w.Write(out)
	return err
}

func layoutIsOffline(e *Engine, data any) bool {
	if m, ok := data.(map[string]any); ok {
		if offline, ok := m["IsOffline"].(bool); ok && offline {
			return true
		}
	}
	if e.hookCtx == nil {
		return false
	}
	offline, err := settings.SiteOffline(e.hookCtx)
	return err == nil && offline
}

func isLayoutTemplate(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "layout.html"
}

func (e *Engine) renderOfflineNotice(data any) (string, error) {
	tmpl, err := e.parse("default/partials/offline-notice.html")
	if err != nil {
		return "", err
	}
	target := tmpl.Lookup("offlineNotice")
	if target == nil {
		target = tmpl
	}
	var buf bytes.Buffer
	if err := target.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func injectAfterBodyOpen(html []byte, insert string) []byte {
	lower := bytes.ToLower(html)
	idx := bytes.Index(lower, []byte("<body"))
	if idx < 0 {
		return append(append([]byte(nil), insert...), html...)
	}
	gt := bytes.IndexByte(html[idx:], '>')
	if gt < 0 {
		return html
	}
	pos := idx + gt + 1
	var out bytes.Buffer
	out.Write(html[:pos])
	out.WriteString(insert)
	out.Write(html[pos:])
	return out.Bytes()
}

func writerHeaders(w io.Writer) map[string][]string {
	rw, ok := w.(interface{ Header() http.Header })
	if !ok {
		return nil
	}
	out := map[string][]string{}
	for key, vals := range rw.Header() {
		out[key] = append([]string(nil), vals...)
	}
	return out
}

func writeResponseStatus(w io.Writer, status int) {
	if status <= 0 {
		return
	}
	rw, ok := w.(http.ResponseWriter)
	if !ok {
		return
	}
	rw.WriteHeader(status)
}

func applyHookHeaders(w io.Writer, args map[string]any) {
	rw, ok := w.(interface{ Header() http.Header })
	if !ok {
		return
	}
	for key, vals := range hookHeaderValues(args["headers"]) {
		rw.Header().Del(key)
		for _, val := range vals {
			rw.Header().Add(key, val)
		}
	}
}

func hookHeaderValues(raw any) map[string][]string {
	out := map[string][]string{}
	switch headers := raw.(type) {
	case map[string][]string:
		for key, vals := range headers {
			out[key] = append([]string(nil), vals...)
		}
	case map[string]any:
		for key, val := range headers {
			switch v := val.(type) {
			case []string:
				out[key] = append([]string(nil), v...)
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						out[key] = append(out[key], s)
					}
				}
			case string:
				out[key] = []string{v}
			}
		}
	}
	return out
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
	if err := e.attachPartials(base, name); err != nil {
		return nil, err
	}
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
	if !themes.IsBuiltinAdmin(e.themes.Admin) && e.adminRoot != "" {
		if err := e.parseAdminThemeDir(set, e.adminRoot); err != nil {
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
	return fs.WalkDir(coreFS, "admin", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "assets" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return nil
		}
		raw, err := coreFS.ReadFile(path)
		if err != nil {
			return err
		}
		return parseAdminFile(set, entry.Name(), string(raw))
	})
}

func (e *Engine) parseAdminThemeDir(set *template.Template, dir string) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "assets" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
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
	isAdmin := strings.HasPrefix(name, "admin/")
	if isAdmin {
		if themes.IsBuiltinAdmin(e.themes.Admin) {
			return e.readEmbedded(name)
		}
		if disk, src, ok := e.readThemeDisk(name, true); ok {
			return disk, src, nil
		}
		return e.readEmbedded(name)
	}

	if !themes.IsBuiltinFrontend(e.themes.Frontend) {
		if disk, src, ok := e.readThemeDisk(name, false); ok {
			return disk, src, nil
		}
	}
	return e.readEmbedded(name)
}

func (e *Engine) readThemeDisk(name string, admin bool) (string, string, bool) {
	if e.site == nil || strings.TrimSpace(e.site.TemplateDir) == "" {
		return "", "", false
	}
	theme := e.themes.Frontend
	themeRoot := themes.Root(e.site.TemplateDir, theme)
	if admin {
		theme = e.themes.Admin
		themeRoot = themes.Root(e.site.TemplateDir, theme)
	}
	if themeRoot == "" || !templatemeta.OverridesEnabled(themeRoot) {
		return "", "", false
	}
	diskPath := themes.DiskPathForLogical(e.site.TemplateDir, theme, name, admin)
	if diskPath == "" {
		return "", "", false
	}
	raw, err := os.ReadFile(diskPath)
	if err != nil {
		return "", "", false
	}
	return string(raw), diskPath, true
}

func (e *Engine) readEmbedded(name string) (string, string, error) {
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
