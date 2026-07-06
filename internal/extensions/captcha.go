package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/settings"
)

// InvokeCaptchaInfo performs GET /captcha on an extension.
func (m *Manager) InvokeCaptchaInfo(ctx context.Context, extensionName string, r *http.Request, userCtx map[string]any) (extension.CaptchaProviderInfo, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || strings.TrimSpace(rt.Capabilities.Captcha) == "" {
		return extension.CaptchaProviderInfo{}, fmt.Errorf("extension %s has no captcha capability", extensionName)
	}
	path := "/" + strings.TrimPrefix(strings.TrimSpace(rt.Capabilities.Captcha), "/")
	resp, err := m.do(rt.Model.Socket, http.MethodGet, path, nil)
	if err != nil {
		return extension.CaptchaProviderInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return extension.CaptchaProviderInfo{}, fmt.Errorf("captcha info: %s", strings.TrimSpace(string(body)))
	}
	var info extension.CaptchaProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return extension.CaptchaProviderInfo{}, err
	}
	return info, nil
}

// InvokeCaptchaRender performs POST /captcha/render.
func (m *Manager) InvokeCaptchaRender(ctx context.Context, extensionName, captchaContext, captchaAction string, r *http.Request, userCtx map[string]any) (extension.CaptchaRenderResult, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || strings.TrimSpace(rt.Capabilities.Captcha) == "" {
		return extension.CaptchaRenderResult{}, fmt.Errorf("extension %s has no captcha capability", extensionName)
	}
	if !m.IsRunning(extensionName) {
		return extension.CaptchaRenderResult{}, fmt.Errorf("extension %s is not running", extensionName)
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return extension.CaptchaRenderResult{}, err
	}
	wire.CaptchaContext = captchaContext
	wire.CaptchaAction = captchaAction
	raw, _ := json.Marshal(wire)
	base := strings.TrimSuffix("/"+strings.TrimPrefix(strings.TrimSpace(rt.Capabilities.Captcha), "/"), "/")
	resp, err := m.do(rt.Model.Socket, http.MethodPost, base+"/render", bytesReader(raw))
	if err != nil {
		return extension.CaptchaRenderResult{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return extension.CaptchaRenderResult{}, fmt.Errorf("captcha render: %s", strings.TrimSpace(string(body)))
	}
	var out extension.CaptchaRenderResult
	if err := json.Unmarshal(body, &out); err != nil {
		return extension.CaptchaRenderResult{}, err
	}
	if strings.TrimSpace(out.FieldName) == "" {
		return extension.CaptchaRenderResult{}, fmt.Errorf("captcha render returned empty field_name")
	}
	return out, nil
}

// InvokeCaptchaVerify performs POST /captcha/verify.
func (m *Manager) InvokeCaptchaVerify(ctx context.Context, extensionName, captchaContext, token, remoteIP string, r *http.Request, userCtx map[string]any) (extension.CaptchaVerifyResult, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || strings.TrimSpace(rt.Capabilities.Captcha) == "" {
		return extension.CaptchaVerifyResult{}, fmt.Errorf("extension %s has no captcha capability", extensionName)
	}
	if !m.IsRunning(extensionName) {
		return extension.CaptchaVerifyResult{}, fmt.Errorf("extension %s is not running", extensionName)
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return extension.CaptchaVerifyResult{}, err
	}
	wire.CaptchaContext = captchaContext
	wire.CaptchaToken = token
	wire.CaptchaRemoteIP = remoteIP
	raw, _ := json.Marshal(wire)
	base := strings.TrimSuffix("/"+strings.TrimPrefix(strings.TrimSpace(rt.Capabilities.Captcha), "/"), "/")
	resp, err := m.do(rt.Model.Socket, http.MethodPost, base+"/verify", bytesReader(raw))
	if err != nil {
		return extension.CaptchaVerifyResult{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out extension.CaptchaVerifyResult
	if err := json.Unmarshal(body, &out); err != nil {
		return extension.CaptchaVerifyResult{}, fmt.Errorf("decode captcha verify: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden || !out.Valid {
		if out.Error == "" {
			out.Error = "captcha verification failed"
		}
		return out, fmt.Errorf("%s", out.Error)
	}
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("captcha verify: %s", strings.TrimSpace(string(body)))
	}
	return out, nil
}

// ResolveCaptchaExtension picks the captcha extension for a provider attribute value.
func (m *Manager) ResolveCaptchaExtension(ctx context.Context, provider string) (string, error) {
	provider = normalizeCaptchaProvider(provider)
	if provider == "" || provider == "any" {
		name, err := settings.CaptchaActiveExtension(ctx)
		if err != nil {
			return "", err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return "", fmt.Errorf("no active captcha extension configured")
		}
		if !m.captchaExtensionAvailable(name) {
			return "", fmt.Errorf("active captcha extension %q is unavailable", name)
		}
		return name, nil
	}
	if m.captchaExtensionAvailable(provider) {
		return provider, nil
	}
	for name, rt := range m.allCaptchaRuntimes() {
		if !m.IsRunning(name) {
			continue
		}
		if strings.EqualFold(name, provider) {
			return name, nil
		}
		if strings.Contains(strings.ToLower(name), provider) {
			return name, nil
		}
		if strings.TrimSpace(rt.Meta.Name) != "" && strings.EqualFold(rt.Meta.Name, provider) {
			return name, nil
		}
		if strings.TrimSpace(rt.Meta.Title) != "" && strings.EqualFold(rt.Meta.Title, provider) {
			return name, nil
		}
	}
	return "", fmt.Errorf("no captcha extension matches provider %q", provider)
}

func (m *Manager) allCaptchaRuntimes() map[string]*Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]*Runtime, len(m.runtimes))
	for name, rt := range m.runtimes {
		if strings.TrimSpace(rt.Capabilities.Captcha) != "" {
			out[name] = rt
		}
	}
	return out
}

// CaptchaExtensionAvailable reports whether a captcha extension is loaded and running.
func (m *Manager) CaptchaExtensionAvailable(name string) bool {
	return m.captchaExtensionAvailable(name)
}

func (m *Manager) captchaExtensionAvailable(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	rt, ok := m.runtime(name)
	if !ok || strings.TrimSpace(rt.Capabilities.Captcha) == "" {
		return false
	}
	return m.IsRunning(name)
}

// CaptchaExtensionNames returns sorted extension names that expose the captcha capability.
func (m *Manager) CaptchaExtensionNames() []string {
	runtimes := m.allCaptchaRuntimes()
	names := make([]string, 0, len(runtimes))
	for name := range runtimes {
		if m.IsRunning(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func bytesReader(raw []byte) *bytes.Reader {
	return bytes.NewReader(raw)
}

func normalizeCaptchaProvider(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "any"
	}
	return raw
}
