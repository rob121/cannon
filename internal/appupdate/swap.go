package appupdate

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ApplyPendingAtStartup swaps a staged binary into place before the server listens.
// If {binary}.latest exists: rename live → .previous, then .latest → live.
func ApplyPendingAtStartup(dataRoot string) error {
	path, err := resolveBinaryPath("")
	if err != nil {
		return err
	}
	if err := applyPending(path); err != nil {
		return err
	}
	if strings.TrimSpace(dataRoot) != "" {
		_ = clearStagedState(filepath.Join(dataRoot, stateFileName))
	}
	return nil
}

func applyPending(binaryPath string) error {
	latest := latestPath(binaryPath)
	if !fileExists(latest) {
		return nil
	}
	prev := previousPath(binaryPath)
	log.Printf("appupdate: applying staged binary %s", latest)

	if fileExists(prev) {
		if err := os.Remove(prev); err != nil {
			return fmt.Errorf("remove previous binary: %w", err)
		}
	}
	if err := os.Rename(binaryPath, prev); err != nil {
		return fmt.Errorf("move live binary to previous: %w", err)
	}
	if err := os.Rename(latest, binaryPath); err != nil {
		// Restore live binary if possible.
		if rerr := os.Rename(prev, binaryPath); rerr != nil {
			return fmt.Errorf("install staged binary: %v (restore failed: %v)", err, rerr)
		}
		return fmt.Errorf("install staged binary: %w", err)
	}
	log.Printf("appupdate: staged update installed; previous kept at %s", prev)
	return nil
}

func clearStagedState(statePath string) error {
	raw, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	state.StagedVersion = ""
	state.StagedSHA256 = ""
	state.StagedAt = nil
	state.RestartRequired = false
	state.UpdateAvailable = false
	out, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, out, 0644)
}

// Rollback swaps the live binary with {binary}.previous.
func (m *Manager) Rollback() error {
	path := m.binaryPath
	if path == "" {
		return fmt.Errorf("current executable path is unavailable")
	}
	prev := previousPath(path)
	if !fileExists(prev) {
		return fmt.Errorf("no previous binary available to roll back to")
	}
	staging := latestPath(path)
	tmp := path + ".rollback-tmp"
	_ = os.Remove(tmp)
	_ = os.Remove(staging)

	if err := os.Rename(path, tmp); err != nil {
		return fmt.Errorf("stage current binary: %w", err)
	}
	if err := os.Rename(prev, path); err != nil {
		_ = os.Rename(tmp, path)
		return fmt.Errorf("restore previous binary: %w", err)
	}
	if err := os.Rename(tmp, prev); err != nil {
		// Live is previous; demoted current may be lost from .previous name — try tmp as previous.
		log.Printf("appupdate: warning: could not store demoted binary as previous: %v", err)
	}

	now := time.Now()
	state, _ := m.LoadState()
	state.RestartRequired = true
	state.UpdateError = ""
	state.LastRolledBackAt = &now
	if err := m.SaveState(state); err != nil {
		return err
	}
	log.Printf("appupdate: rollback prepared; restarting to run %s", path)
	return nil
}

// DiscardStaged removes a pending {binary}.latest without applying it.
func (m *Manager) DiscardStaged() error {
	path := m.binaryPath
	if path == "" {
		return fmt.Errorf("current executable path is unavailable")
	}
	latest := latestPath(path)
	if !fileExists(latest) {
		return fmt.Errorf("no staged update to discard")
	}
	if err := os.Remove(latest); err != nil {
		return err
	}
	state, _ := m.LoadState()
	state.StagedVersion = ""
	state.StagedSHA256 = ""
	state.StagedAt = nil
	state.RestartRequired = false
	state.UpdateError = ""
	return m.SaveState(state)
}
