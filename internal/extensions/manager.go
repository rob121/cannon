package extensions

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

type Capabilities struct {
	Request       string `json:"request"`
	Page          string `json:"page"`
	Block         string `json:"block"`
	Endpoint      string `json:"endpoint"`
	Data          string `json:"data"`
	Admin         string `json:"admin"`
	Help          string `json:"help"`
	Configuration string `json:"configuration"`
	Hooks         string `json:"hooks"`
	Templates     string `json:"templates"`
	Captcha       string `json:"captcha"`
}

type CapabilitiesResponse struct {
	Capabilities Capabilities              `json:"capabilities"`
	Permissions  []extension.PermissionDef `json:"permissions,omitempty"`
	Defaults     struct {
		Admin struct {
			MenuName string `json:"menu_name"`
		} `json:"admin"`
	} `json:"defaults"`
}

type WireRequest = extension.WireRequest
type WireResponse = extension.WireResponse

type Runtime struct {
	Model                models.Extension
	Capabilities         Capabilities
	Permissions          []extension.PermissionDef
	Blocks               []extension.BlockDefinition
	Pages                []extension.PageDefinition
	Endpoints            []extension.EndpointDefinition
	Templates            []extension.TemplateDefinition
	Hooks                []string
	DefaultAdminMenuName string
	Meta                 Meta
	MetaAvailable        bool
	cmd                  *exec.Cmd
}

type Manager struct {
	app          *config.App
	site         *config.SiteConfig
	runtimes     map[string]*Runtime
	suppressed   map[string]bool
	updateClient *http.Client
	updateOnce   sync.Once
	mu           sync.RWMutex
}

func NewManager(app *config.App, site *config.SiteConfig) *Manager {
	return &Manager{
		app:          app,
		site:         site,
		runtimes:     make(map[string]*Runtime),
		suppressed:   make(map[string]bool),
		updateClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *Manager) Bootstrap(ctx context.Context) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	if err := m.syncDirectory(db); err != nil {
		return err
	}
	var rows []models.Extension
	if err := db.Where("status = ?", models.StatusActive).Order("sort asc").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if m.isSuppressed(row.Name) {
			continue
		}
		if err := m.start(ctx, row); err != nil {
			return err
		}
	}
	m.startUpdateChecker()
	return nil
}

func (m *Manager) syncDirectory(db *gorm.DB) error {
	dir := m.app.Extensions.Dir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(m.app.Extensions.SocketsDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&0111 == 0 {
			continue
		}
		socket := m.socketPath(name)
		var row models.Extension
		err = db.Where("name = ?", name).First(&row).Error
		if err == gorm.ErrRecordNotFound {
			row = models.Extension{Name: name, Socket: socket, Status: models.StatusInactive}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if row.Socket == "" {
			row.Socket = socket
			if err := db.Save(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) socketPath(name string) string {
	sum := sha256.Sum256([]byte(name + ":" + m.site.ID))
	hash := hex.EncodeToString(sum[:16])
	return filepath.Join(m.app.Extensions.SocketsDir, hash+".sock")
}

func (m *Manager) binaryPath(name string) string {
	return filepath.Join(m.app.Extensions.Dir, name)
}

func (m *Manager) start(ctx context.Context, row models.Extension) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rt, ok := m.runtimes[row.Name]; ok && rt.cmd != nil && rt.cmd.Process != nil && rt.cmd.ProcessState == nil {
		rt.Model = row
		_ = m.refreshRuntimeMetadata(ctx, rt, row.Socket)
		return nil
	}
	_ = os.Remove(row.Socket)
	cmd := exec.Command(m.binaryPath(row.Name), "--site="+m.site.ID, "--socket="+row.Socket)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if cfgPath := config.ConfigPath(); cfgPath != "" {
		cmd.Env = append(os.Environ(), "CANNON_CONFIG="+cfgPath)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start extension %s: %w", row.Name, err)
	}
	rt := &Runtime{Model: row, cmd: cmd}
	m.runtimes[row.Name] = rt
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(row.Socket); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := m.refreshRuntimeMetadata(ctx, rt, row.Socket); err != nil {
		return err
	}
	if !row.Installed {
		if err := m.runInstall(ctx, row.Socket); err != nil {
			return fmt.Errorf("install extension %s: %w", row.Name, err)
		}
		if err := m.markInstalled(ctx, row.Name); err != nil {
			return err
		}
		rt.Model.Installed = true
	}
	return nil
}

func (m *Manager) refreshRuntimeMetadata(ctx context.Context, rt *Runtime, socketPath string) error {
	payload, err := m.fetchCapabilities(socketPath)
	if err != nil {
		return err
	}
	rt.Capabilities = payload.Capabilities
	rt.DefaultAdminMenuName = strings.TrimSpace(payload.Defaults.Admin.MenuName)
	rt.Permissions = append([]extension.PermissionDef(nil), payload.Permissions...)
	if len(payload.Permissions) > 0 {
		security.RegisterExtensionPermissions(rt.Model.Name, payload.Permissions)
		if db, err := sites.DB(ctx); err == nil {
			_ = security.SyncToDB(db)
		}
	}
	rt.Blocks = nil
	rt.Pages = nil
	rt.Endpoints = nil
	rt.Templates = nil
	rt.Hooks = nil
	if payload.Capabilities.Block != "" {
		if blocks, err := m.fetchBlocks(socketPath, payload.Capabilities.Block); err == nil {
			rt.Blocks = blocks
		}
	}
	if payload.Capabilities.Page != "" {
		rt.Pages = m.loadPages(socketPath, payload.Capabilities.Page)
	}
	if payload.Capabilities.Endpoint != "" {
		rt.Endpoints = m.loadEndpoints(socketPath, payload.Capabilities.Endpoint)
	}
	if payload.Capabilities.Hooks != "" {
		if hookNames, err := m.fetchHooks(socketPath, payload.Capabilities.Hooks); err == nil {
			rt.Hooks = hookNames
		}
	}
	if payload.Capabilities.Templates != "" {
		if templates, err := m.fetchTemplates(socketPath, payload.Capabilities.Templates); err == nil {
			rt.Templates = templates
		}
	}
	if err := m.applyDefaultMenuName(ctx, &rt.Model, payload.Defaults.Admin.MenuName); err != nil {
		return err
	}
	rt.Meta = Meta{}
	rt.MetaAvailable = false
	if meta, err := m.fetchMeta(socketPath); err == nil {
		rt.Meta = meta
		rt.MetaAvailable = true
		if err := m.applyCachedMeta(ctx, &rt.Model, meta); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyCachedMeta(ctx context.Context, row *models.Extension, meta Meta) error {
	updates := map[string]any{}
	if v := strings.TrimSpace(meta.Title); v != "" && v != strings.TrimSpace(row.Title) {
		updates["title"] = v
		row.Title = v
	}
	if v := strings.TrimSpace(meta.Description); v != "" && v != strings.TrimSpace(row.Description) {
		updates["description"] = v
		row.Description = v
	}
	if v := strings.TrimSpace(meta.Version); v != "" && v != strings.TrimSpace(row.Version) {
		updates["version"] = v
		row.Version = v
		if row.LatestVersion != "" && !newerVersion(row.LatestVersion, v) {
			updates["update_available"] = false
			updates["update_asset_url"] = ""
			updates["update_asset_sha256"] = ""
		}
	}
	if v := strings.TrimSpace(meta.UpdateURLBase); v != strings.TrimSpace(row.UpdateURLBase) {
		updates["update_url_base"] = v
		row.UpdateURLBase = v
		if v == "" {
			updates["update_available"] = false
			updates["latest_version"] = ""
			updates["update_asset_url"] = ""
			updates["update_asset_sha256"] = ""
			updates["update_error"] = ""
		}
	}
	if len(updates) == 0 {
		return nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	return db.Model(&models.Extension{}).Where("name = ?", row.Name).Updates(updates).Error
}

func (m *Manager) applyDefaultMenuName(ctx context.Context, row *models.Extension, menuName string) error {
	menuName = strings.TrimSpace(menuName)
	if menuName == "" || strings.TrimSpace(row.MenuName) != "" {
		return nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	if err := db.Model(&models.Extension{}).Where("name = ?", row.Name).Update("menu_name", menuName).Error; err != nil {
		return err
	}
	row.MenuName = menuName
	return nil
}

func (m *Manager) fetchCapabilities(socketPath string) (CapabilitiesResponse, error) {
	resp, err := m.do(socketPath, http.MethodGet, "/capabilities", nil)
	if err != nil {
		return CapabilitiesResponse{}, err
	}
	defer resp.Body.Close()
	var payload CapabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return CapabilitiesResponse{}, err
	}
	return payload, nil
}

func (m *Manager) HandleRequest(ctx context.Context, r *http.Request, userCtx map[string]any) (*http.Request, *http.Response, bool, error) {
	m.mu.RLock()
	runtimes := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Request != "" {
			runtimes = append(runtimes, rt)
		}
	}
	m.mu.RUnlock()
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].Model.Sort < runtimes[j].Model.Sort })

	current := r
	for _, rt := range runtimes {
		wire, err := m.buildWireRequest(current, userCtx)
		if err != nil {
			return current, nil, false, err
		}
		raw, _ := json.Marshal(wire)
		resp, err := m.do(rt.Model.Socket, http.MethodPost, "/"+strings.TrimPrefix(rt.Capabilities.Request, "/"), bytes.NewReader(raw))
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var out WireResponse
		if err := json.Unmarshal(body, &out); err != nil {
			continue
		}
		if out.Updated != nil {
			current = m.applyWireRequest(current, out.Updated)
		}
		if out.Stop || out.StatusCode >= 300 {
			return current, m.buildHTTPResponse(out), true, nil
		}
	}
	return current, nil, false, nil
}

func (m *Manager) RenderPage(ctx context.Context, extensionName, item string, r *http.Request, userCtx, pageData map[string]any) (string, error) {
	return m.InvokePage(ctx, extensionName, item, r, userCtx, pageData)
}

// InvokePage POSTs to an extension page handler with optional route metadata.
func (m *Manager) InvokePage(ctx context.Context, extensionName, item string, r *http.Request, userCtx, pageData map[string]any) (string, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Page == "" {
		return "", fmt.Errorf("extension %s has no page capability", extensionName)
	}
	item = strings.TrimSpace(item)
	if item == "" {
		item = "default"
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return "", err
	}
	wire.PageItem = item
	if len(pageData) > 0 {
		wire.PageData = pageData
	}
	raw, _ := json.Marshal(wire)
	path := capabilityPath(rt.Capabilities.Page, item)
	resp, err := m.do(rt.Model.Socket, http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out WireResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return string(body), nil
	}
	return out.Body, nil
}

// InvokeBlock POSTs to an extension block handler with optional placement metadata.
func (m *Manager) InvokeBlock(ctx context.Context, extensionName, item, space string, r *http.Request, userCtx, blockData map[string]any) (string, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Block == "" {
		return "", fmt.Errorf("extension %s has no block capability", extensionName)
	}
	item = strings.TrimSpace(item)
	if item == "" {
		return "", fmt.Errorf("block item is required")
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return "", err
	}
	wire.BlockSpace = space
	wire.BlockItem = item
	if len(blockData) > 0 {
		wire.BlockData = blockData
	}
	raw, _ := json.Marshal(wire)
	path := capabilityPath(rt.Capabilities.Block, item)
	resp, err := m.do(rt.Model.Socket, http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out WireResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return string(body), nil
	}
	return out.Body, nil
}

func (m *Manager) RenderBlock(ctx context.Context, space string, r *http.Request, userCtx map[string]any) (string, error) {
	m.mu.RLock()
	runtimes := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Block != "" {
			runtimes = append(runtimes, rt)
		}
	}
	m.mu.RUnlock()
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].Model.Sort < runtimes[j].Model.Sort })

	for _, rt := range runtimes {
		item, ok := MatchBlock(rt.Blocks, space)
		if !ok && len(rt.Blocks) == 0 {
			item, ok = space, true
		}
		if !ok {
			continue
		}
		wire, err := m.buildWireRequest(r, userCtx)
		if err != nil {
			return "", err
		}
		wire.BlockSpace = space
		wire.BlockItem = item
		raw, _ := json.Marshal(wire)
		path := capabilityPath(rt.Capabilities.Block, item)
		resp, err := m.do(rt.Model.Socket, http.MethodPost, path, bytes.NewReader(raw))
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var out WireResponse
		if err := json.Unmarshal(body, &out); err == nil && out.Body != "" {
			return out.Body, nil
		}
	}
	return "", nil
}

// Runtime returns a loaded extension runtime by name.
func (m *Manager) Runtime(name string) (*Runtime, bool) {
	return m.runtime(name)
}

func (m *Manager) runtime(name string) (*Runtime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rt, ok := m.runtimes[name]
	return rt, ok
}

// IsRunning reports whether an extension process is currently running.
func (m *Manager) IsRunning(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rt, ok := m.runtimes[name]
	if !ok || rt.cmd == nil || rt.cmd.Process == nil {
		return false
	}
	return rt.cmd.ProcessState == nil
}

// AdminRuntimes returns active extensions that expose an admin capability.
func (m *Manager) AdminRuntimes() []*Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Admin != "" && rt.Model.Status == models.StatusActive {
			out = append(out, rt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Model.Sort < out[j].Model.Sort })
	return out
}

// RenderAdmin proxies an admin-area request to an extension admin handler.
func (m *Manager) RenderAdmin(ctx context.Context, extensionName, suffixPath string, r *http.Request, userCtx map[string]any) (string, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Admin == "" {
		return "", fmt.Errorf("extension %s has no admin capability", extensionName)
	}
	adminPath := "/" + strings.TrimPrefix(rt.Capabilities.Admin, "/")
	if suffixPath != "" {
		adminPath = strings.TrimRight(adminPath, "/") + suffixPath
	}
	if r.URL.RawQuery != "" {
		adminPath += "?" + r.URL.RawQuery
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(wire)
	resp, err := m.do(rt.Model.Socket, http.MethodPost, adminPath, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out WireResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return string(body), nil
	}
	return out.Body, nil
}

// Start launches an extension process if it is not already running.
func (m *Manager) Start(ctx context.Context, row models.Extension) error {
	m.mu.Lock()
	delete(m.suppressed, row.Name)
	m.mu.Unlock()
	return m.start(ctx, row)
}

// Install runs POST /install for an extension that has not been marked installed yet.
func (m *Manager) Install(ctx context.Context, row models.Extension) error {
	if row.Installed {
		return fmt.Errorf("extension is already installed")
	}
	if !m.IsRunning(row.Name) {
		return m.start(ctx, row)
	}
	if err := m.runInstall(ctx, row.Socket); err != nil {
		return fmt.Errorf("install extension %s: %w", row.Name, err)
	}
	return m.markInstalled(ctx, row.Name)
}

// Stop terminates a running extension process.
func (m *Manager) Stop(name string) {
	m.mu.Lock()
	rt, ok := m.runtimes[name]
	if !ok || rt.cmd == nil || rt.cmd.Process == nil {
		m.suppressed[name] = true
		m.mu.Unlock()
		return
	}
	proc := rt.cmd.Process
	delete(m.runtimes, name)
	m.suppressed[name] = true
	m.mu.Unlock()
	_ = proc.Kill()
}

func (m *Manager) isSuppressed(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.suppressed[name]
}

// Restart stops and starts an extension.
func (m *Manager) Restart(ctx context.Context, row models.Extension) error {
	m.Stop(row.Name)
	return m.Start(ctx, row)
}

func (m *Manager) do(socketPath, method, path string, body io.Reader) (*http.Response, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, "http://unix"+path, body)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(conn), req)
}

func (m *Manager) buildWireRequest(r *http.Request, userCtx map[string]any) (*WireRequest, error) {
	var body string
	if r.Body != nil {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = string(raw)
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	csrfToken := ""
	if svc, err := user.FromContext(r.Context()); err == nil {
		if token, err := svc.EnsureCSRFToken(); err == nil {
			csrfToken = token
		}
	}
	return &WireRequest{
		Method: r.Method,
		URL:    r.URL.String(),
		Header: r.Header,
		Body:   body,
		User:   userCtx,
		CSRF:   csrfToken,
		SiteID: m.site.ID,
	}, nil
}

func (m *Manager) applyWireRequest(base *http.Request, wire *WireRequest) *http.Request {
	clone := base.Clone(base.Context())
	if wire.Method != "" {
		clone.Method = wire.Method
	}
	if wire.URL != "" {
		if u, err := http.NewRequest(wire.Method, wire.URL, nil); err == nil {
			clone.URL = u.URL
		}
	}
	if wire.Header != nil {
		clone.Header = wire.Header
	}
	if wire.Body != "" {
		clone.Body = io.NopCloser(strings.NewReader(wire.Body))
	}
	return clone
}

func (m *Manager) buildHTTPResponse(wire WireResponse) *http.Response {
	status := wire.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Header:     wire.Header,
		Body:       io.NopCloser(strings.NewReader(wire.Body)),
	}
}
