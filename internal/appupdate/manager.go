package appupdate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/updater"
	"github.com/rob121/cannon/internal/version"
)

const (
	binaryName         = "cannon"
	manifestName       = "cannon.json"
	defaultUpdateBase  = "https://github.com/rob121/cannon/releases/download"
	updateCheckInterval = 6 * time.Hour
	stateFileName      = "cannon-update-state.json"
)

// State tracks remote update metadata for the Cannon binary.
type State struct {
	LatestVersion     string     `json:"latest_version"`
	UpdateAvailable   bool       `json:"update_available"`
	UpdateAssetURL    string     `json:"update_asset_url"`
	UpdateAssetSHA256 string     `json:"update_asset_sha256"`
	UpdateCheckedAt   *time.Time `json:"update_checked_at"`
	UpdateError       string     `json:"update_error"`
	RestartRequired   bool       `json:"restart_required"`
}

// Manager checks for and applies Cannon binary updates.
type Manager struct {
	cfg        *config.App
	statePath  string
	binaryPath string
	client     *updater.Client
	checkOnce  sync.Once
}

// New creates an update manager for the running Cannon process.
func New(cfg *config.App) *Manager {
	if cfg == nil {
		cfg = config.Get()
	}
	m := &Manager{
		cfg:       cfg,
		statePath: filepath.Join(cfg.DataRoot, stateFileName),
		client: &updater.Client{
			HTTP:     &http.Client{Timeout: 2 * time.Minute},
			Manifest: manifestName,
		},
	}
	m.binaryPath, _ = os.Executable()
	return m
}

// StartChecker runs an initial update check and then checks every six hours.
func (m *Manager) StartChecker() {
	m.checkOnce.Do(func() {
		go func() {
			_ = m.Check()
			ticker := time.NewTicker(updateCheckInterval)
			defer ticker.Stop()
			for range ticker.C {
				_ = m.Check()
			}
		}()
	})
}

// LoadState reads persisted update metadata.
func (m *Manager) LoadState() (State, error) {
	raw, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

// SaveState persists update metadata.
func (m *Manager) SaveState(state State) error {
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.statePath, raw, 0644)
}

// CurrentVersion returns the running Cannon version.
func (m *Manager) CurrentVersion() string {
	return version.Version
}

// UpdateURLBase returns the configured release base URL.
func (m *Manager) UpdateURLBase() string {
	if m.cfg != nil && strings.TrimSpace(m.cfg.Update.URLBase) != "" {
		return strings.TrimRight(strings.TrimSpace(m.cfg.Update.URLBase), "/")
	}
	return defaultUpdateBase
}

// BinaryPath returns the path to the running executable.
func (m *Manager) BinaryPath() string {
	return m.binaryPath
}

// Check fetches the latest release metadata and updates local state.
func (m *Manager) Check() error {
	now := time.Now()
	info, err := m.client.LatestInfo(m.UpdateURLBase(), binaryName)
	state, _ := m.LoadState()
	state.UpdateCheckedAt = &now
	if err != nil {
		state.UpdateError = err.Error()
		return m.SaveState(state)
	}
	available := updater.NewerVersion(info.Version, m.CurrentVersion())
	state.LatestVersion = info.Version
	state.UpdateAvailable = available
	state.UpdateAssetURL = ""
	state.UpdateAssetSHA256 = ""
	state.UpdateError = ""
	if available {
		state.UpdateAssetURL = info.URL
		state.UpdateAssetSHA256 = info.SHA256
	}
	return m.SaveState(state)
}

// Apply downloads and installs the pending update binary.
func (m *Manager) Apply() error {
	if err := m.Check(); err != nil {
		return err
	}
	state, err := m.LoadState()
	if err != nil {
		return err
	}
	if !state.UpdateAvailable {
		return fmt.Errorf("cannon is already up to date")
	}
	assetURL := strings.TrimSpace(state.UpdateAssetURL)
	if assetURL == "" {
		assetURL = updater.DefaultAssetURL(m.UpdateURLBase(), state.LatestVersion, binaryName)
	}
	if assetURL == "" {
		return fmt.Errorf("update asset URL is unavailable")
	}
	target := m.binaryPath
	if target == "" {
		return fmt.Errorf("current executable path is unavailable")
	}
	tmp, err := updater.Download(m.client.HTTP, assetURL, target)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if checksum := strings.TrimSpace(state.UpdateAssetSHA256); checksum != "" {
		if err := updater.VerifySHA256(tmp, checksum); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	state.UpdateAvailable = false
	state.UpdateAssetURL = ""
	state.UpdateAssetSHA256 = ""
	state.UpdateError = ""
	state.RestartRequired = true
	return m.SaveState(state)
}
