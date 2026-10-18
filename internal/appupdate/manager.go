package appupdate

import (
	"encoding/json"
	"fmt"
	"log"
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
	binaryName          = "cannon"
	manifestName        = "cannon.json"
	defaultUpdateBase   = "https://github.com/rob121/cannon/releases/download"
	updateCheckInterval = 6 * time.Hour
	stateFileName       = "cannon-update-state.json"
	restartDelay        = 1500 * time.Millisecond
)

// exitAfterRestart is overridden in tests.
var exitAfterRestart = func() {
	time.Sleep(restartDelay)
	os.Exit(0)
}

// State tracks remote update metadata and staged/rollback status.
type State struct {
	LatestVersion     string     `json:"latest_version"`
	UpdateAvailable   bool       `json:"update_available"`
	UpdateAssetURL    string     `json:"update_asset_url"`
	UpdateAssetSHA256 string     `json:"update_asset_sha256"`
	UpdateCheckedAt   *time.Time `json:"update_checked_at"`
	UpdateError       string     `json:"update_error"`
	RestartRequired   bool       `json:"restart_required"`
	StagedVersion     string     `json:"staged_version"`
	StagedSHA256      string     `json:"staged_sha256"`
	StagedAt          *time.Time `json:"staged_at"`
	LastRolledBackAt  *time.Time `json:"last_rolled_back_at"`
}

// Status is UI-facing update status combining state file and disk siblings.
type Status struct {
	State          State
	BinaryPath     string
	LatestPath     string
	PreviousPath   string
	HasStaged      bool
	HasPrevious    bool
	StagedVersion  string
	CurrentVersion string
	Commit         string
}

// Manager checks for and applies Cannon binary updates.
type Manager struct {
	cfg             *config.App
	statePath       string
	binaryPath      string
	fixedBinaryPath bool // when true, tests pinned binaryPath; do not re-resolve
	client          *updater.Client
	checkOnce       sync.Once
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
	m.binaryPath, _ = resolveBinaryPath("")
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
	return m.refreshBinaryPath()
}

func (m *Manager) refreshBinaryPath() string {
	if m.fixedBinaryPath && strings.TrimSpace(m.binaryPath) != "" {
		return m.binaryPath
	}
	if path, err := resolveBinaryPath(""); err == nil && path != "" {
		m.binaryPath = path
	}
	return m.binaryPath
}

// SetBinaryPath pins the executable path (used in tests; may be a *.previous inode path).
func (m *Manager) SetBinaryPath(path string) {
	m.binaryPath = filepath.Clean(path)
	m.fixedBinaryPath = true
}

// Status returns combined state + disk sibling presence for the admin UI.
func (m *Manager) Status() Status {
	state, _ := m.LoadState()
	raw := m.refreshBinaryPath()
	path := normalizeLiveBinaryPath(raw)
	_, hasStaged := findSibling(path, ".latest")
	_, hasPrevious := findSibling(path, ".previous")
	stagedVersion := state.StagedVersion
	if hasStaged && stagedVersion == "" {
		stagedVersion = state.LatestVersion
	}
	return Status{
		State:          state,
		BinaryPath:     path,
		LatestPath:     latestPath(path),
		PreviousPath:   previousPath(path),
		HasStaged:      hasStaged,
		HasPrevious:    hasPrevious,
		StagedVersion:  stagedVersion,
		CurrentVersion: m.CurrentVersion(),
		Commit:         version.Commit,
	}
}

// Check fetches the latest release metadata and updates local state.
func (m *Manager) Check() error {
	now := time.Now()
	info, err := m.client.LatestInfo(m.UpdateURLBase(), binaryName)
	state, _ := m.LoadState()
	state.UpdateCheckedAt = &now
	if err != nil {
		state.UpdateError = err.Error()
		log.Printf("appupdate: check failed: %v", err)
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
		log.Printf("appupdate: update available %s -> %s", m.CurrentVersion(), info.Version)
	} else {
		log.Printf("appupdate: up to date (%s)", m.CurrentVersion())
	}
	return m.SaveState(state)
}

// Apply downloads the pending update into {binary}.latest (does not replace the running binary).
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
	target := normalizeLiveBinaryPath(m.refreshBinaryPath())
	if target == "" {
		return fmt.Errorf("current executable path is unavailable")
	}
	staged := latestPath(target)
	log.Printf("appupdate: staging %s -> %s", assetURL, staged)

	tmp, err := updater.Download(m.client.HTTP, assetURL, staged)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if checksum := strings.TrimSpace(state.UpdateAssetSHA256); checksum != "" {
		if err := updater.VerifySHA256(tmp, checksum); err != nil {
			return err
		}
	}
	_ = os.Remove(staged)
	if err := os.Rename(tmp, staged); err != nil {
		return fmt.Errorf("stage update binary: %w", err)
	}
	now := time.Now()
	state.StagedVersion = state.LatestVersion
	state.StagedSHA256 = state.UpdateAssetSHA256
	state.StagedAt = &now
	state.RestartRequired = true
	state.UpdateError = ""
	if err := m.SaveState(state); err != nil {
		return err
	}
	log.Printf("appupdate: staged %s at %s (restart to apply)", state.StagedVersion, staged)
	return nil
}

// ScheduleRestart exits the process after a short delay so systemd can restart it.
func (m *Manager) ScheduleRestart() {
	log.Printf("appupdate: scheduling process exit for systemd restart")
	go exitAfterRestart()
}
