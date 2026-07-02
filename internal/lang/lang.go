package lang

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/rob121/cannon/internal/config"
	"gopkg.in/ini.v1"
)

// LanguageFile describes a locale ini file on disk.
type LanguageFile struct {
	Locale   string
	Scope    string
	Filename string
	Label    string
	Count    int
}

// LanguageSection groups entries under an ini section.
type LanguageSection struct {
	Name    string
	Entries []LanguageEntry
}

// LanguageEntry is a single key/value within a section.
type LanguageEntry struct {
	Key   string
	Value string
}

func scopeFromFilename(locale, name string) string {
	if name == locale+".ini" {
		return "default"
	}
	if strings.HasPrefix(name, locale+"-") && strings.HasSuffix(name, ".ini") {
		return strings.TrimSuffix(strings.TrimPrefix(name, locale+"-"), ".ini")
	}
	return ""
}

func filenameForScope(locale, scope string) string {
	if scope == "" || scope == "default" {
		return locale + ".ini"
	}
	return locale + "-" + scope + ".ini"
}

func scopeLabel(scope string) string {
	switch scope {
	case "admin":
		return "Admin"
	case "site":
		return "Site"
	case "default":
		return "Default"
	default:
		if scope == "" {
			return "Default"
		}
		return strings.ToUpper(scope[:1]) + scope[1:]
	}
}

// Bundle holds parsed translations for a locale and scope.
type Bundle struct {
	Locale string
	Scope  string
	Values map[string]string
}

// Manager loads and formats locale strings.
type Manager struct {
	dir     string
	locale  string
	bundles map[string]*Bundle
	mu      sync.RWMutex
}

// NewManager creates a language manager for a site.
func NewManager(site *config.SiteConfig, locale string) (*Manager, error) {
	if locale == "" {
		locale = "en-US"
	}
	m := &Manager{
		dir:     site.LanguageDir,
		locale:  locale,
		bundles: make(map[string]*Bundle),
	}
	if err := m.Reload(); err != nil {
		return m, err
	}
	return m, nil
}

// Reload reads all matching ini files.
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return err
	}

	m.bundles = make(map[string]*Bundle)
	prefix := m.locale

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if name != prefix+".ini" && !strings.HasPrefix(name, prefix+"-") {
			continue
		}

		scope := scopeFromFilename(prefix, name)

		file, err := ini.Load(filepath.Join(m.dir, name))
		if err != nil {
			return err
		}

		b := &Bundle{Locale: m.locale, Scope: scope, Values: map[string]string{}}
		for _, section := range file.Sections() {
			sec := section.Name()
			for _, key := range section.Keys() {
				id := key.Name()
				if sec != "DEFAULT" {
					id = sec + "." + id
				}
				b.Values[id] = key.String()
			}
		}
		m.bundles[scope] = b
	}

	return nil
}

// Fmt formats a translation key with named placeholders.
func (m *Manager) Fmt(key string, pairs ...string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	text, ok := m.lookup(key)
	if !ok {
		return key
	}

	data := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		data[pairs[i]] = pairs[i+1]
	}

	tmpl, err := template.New(key).Parse(text)
	if err != nil {
		return text
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return text
	}
	return buf.String()
}

// Save writes or updates a key in the given scope file.
func (m *Manager) Save(scope, section, key, value string) error {
	m.mu.Lock()

	filename := filenameForScope(m.locale, scope)
	path := filepath.Join(m.dir, filename)

	file := ini.Empty()
	if raw, err := os.ReadFile(path); err == nil {
		file, err = ini.Load(raw)
		if err != nil {
			m.mu.Unlock()
			return err
		}
	}

	sec := file.Section(section)
	sec.Key(key).SetValue(value)
	if err := file.SaveTo(path); err != nil {
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	return m.Reload()
}

// ReplaceScope rewrites an entire scope file from section/key maps.
func (m *Manager) ReplaceScope(scope string, sections map[string]map[string]string) error {
	m.mu.Lock()

	file := ini.Empty()
	sectionNames := make([]string, 0, len(sections))
	for name := range sections {
		sectionNames = append(sectionNames, name)
	}
	sort.Strings(sectionNames)
	for _, sectionName := range sectionNames {
		keys := sections[sectionName]
		sec := file.Section(sectionName)
		keyNames := make([]string, 0, len(keys))
		for k := range keys {
			keyNames = append(keyNames, k)
		}
		sort.Strings(keyNames)
		for _, k := range keyNames {
			sec.Key(k).SetValue(keys[k])
		}
	}

	path := filepath.Join(m.dir, filenameForScope(m.locale, scope))
	if err := file.SaveTo(path); err != nil {
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	return m.Reload()
}

// Delete removes a translation key from a scope file.
func (m *Manager) Delete(scope, fullKey string) error {
	m.mu.Lock()

	section := "DEFAULT"
	key := fullKey
	if idx := strings.Index(fullKey, "."); idx >= 0 {
		section = fullKey[:idx]
		key = fullKey[idx+1:]
	}

	filename := filenameForScope(m.locale, scope)
	path := filepath.Join(m.dir, filename)

	file, err := ini.Load(path)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	sec := file.Section(section)
	if sec == nil {
		m.mu.Unlock()
		return fmt.Errorf("section not found")
	}
	sec.DeleteKey(key)
	if err := file.SaveTo(path); err != nil {
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	return m.Reload()
}

// Entries returns sorted key/value pairs for admin editing.
func (m *Manager) Entries(scope string) [][2]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, ok := m.bundles[scope]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(b.Values))
	for k := range b.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][2]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, [2]string{k, b.Values[k]})
	}
	return out
}

// Files returns locale ini files sorted by locale then filename.
func (m *Manager) Files() []LanguageFile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil
	}

	prefix := m.locale
	out := make([]LanguageFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if name != prefix+".ini" && !strings.HasPrefix(name, prefix+"-") {
			continue
		}
		scope := scopeFromFilename(prefix, name)
		count := 0
		if b, ok := m.bundles[scope]; ok {
			count = len(b.Values)
		}
		out = append(out, LanguageFile{
			Locale:   m.locale,
			Scope:    scope,
			Filename: name,
			Label:    scopeLabel(scope),
			Count:    count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Locale != out[j].Locale {
			return out[i].Locale < out[j].Locale
		}
		return out[i].Filename < out[j].Filename
	})
	return out
}

// Sections returns entries grouped by ini section for a scope.
func (m *Manager) Sections(scope string) []LanguageSection {
	entries := m.Entries(scope)
	grouped := map[string][]LanguageEntry{}
	order := make([]string, 0)
	for _, e := range entries {
		secName := ""
		key := e[0]
		if idx := strings.Index(e[0], "."); idx >= 0 {
			secName = e[0][:idx]
			key = e[0][idx+1:]
		}
		if _, ok := grouped[secName]; !ok {
			order = append(order, secName)
		}
		grouped[secName] = append(grouped[secName], LanguageEntry{Key: key, Value: e[1]})
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i] == "" {
			return true
		}
		if order[j] == "" {
			return false
		}
		return order[i] < order[j]
	})
	sections := make([]LanguageSection, 0, len(order))
	for _, name := range order {
		rows := grouped[name]
		sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })
		sections = append(sections, LanguageSection{Name: name, Entries: rows})
	}
	return sections
}

// ScopeExists reports whether a scope file is loaded for the current locale.
func (m *Manager) ScopeExists(scope string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.bundles[scope]
	if ok {
		return true
	}
	path := filepath.Join(m.dir, filenameForScope(m.locale, scope))
	_, err := os.Stat(path)
	return err == nil
}

func (m *Manager) lookup(key string) (string, bool) {
	for _, scope := range []string{"admin", "site"} {
		if b, ok := m.bundles[scope]; ok {
			if v, ok := b.Values[key]; ok {
				return v, true
			}
		}
	}
	for _, b := range m.bundles {
		if v, ok := b.Values[key]; ok {
			return v, true
		}
	}
	return "", false
}

// ResolveLocale picks locale from cookie or Accept-Language.
func ResolveLocale(rCookie, acceptLanguage string) string {
	if rCookie != "" {
		return rCookie
	}
	if acceptLanguage == "" {
		return "en-US"
	}
	parts := strings.Split(acceptLanguage, ",")
	if len(parts) == 0 {
		return "en-US"
	}
	lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])
	if lang == "" {
		return "en-US"
	}
	return lang
}

// EnsureDefaults creates starter language files.
func EnsureDefaults(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	sitePath := filepath.Join(dir, "en-US-site.ini")
	if _, err := os.Stat(sitePath); os.IsNotExist(err) {
		content := "[user]\nregistered = \"You have successfully registered as {{.Username}}\"\n"
		if err := os.WriteFile(sitePath, []byte(content), 0644); err != nil {
			return err
		}
	}
	adminPath := filepath.Join(dir, "en-US-admin.ini")
	if _, err := os.Stat(adminPath); os.IsNotExist(err) {
		content := "[dashboard]\ntitle = \"Dashboard\"\n"
		if err := os.WriteFile(adminPath, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

// ScopeFromPath returns admin or site based on request path.
func ScopeFromPath(path string) string {
	if strings.HasPrefix(path, "/admin") {
		return "admin"
	}
	return "site"
}

// Format is a package-level helper when a manager is unavailable.
func Format(key string, pairs ...string) string {
	return key
}
