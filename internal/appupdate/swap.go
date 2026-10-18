package appupdate

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// reexecAfterPending is overridden in tests to avoid replacing the test process.
var reexecAfterPending = true

// ApplyPendingAtStartup swaps a staged binary into place before the server listens.
// If {binary}.latest exists: rename live → .previous, then .latest → live, then re-exec
// so this process actually runs the new binary (Linux renames the running inode to .previous).
func ApplyPendingAtStartup(dataRoot string) error {
	path, err := resolveBinaryPath("")
	if err != nil {
		return err
	}
	applied, err := applyPending(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(dataRoot) != "" {
		_ = clearStagedState(filepath.Join(dataRoot, stateFileName))
	}
	if applied && reexecAfterPending {
		return reexecLive(path)
	}
	return nil
}

func applyPending(binaryPath string) (bool, error) {
	live := normalizeLiveBinaryPath(binaryPath)
	latest := latestPath(live)
	if !fileExists(latest) {
		// Repair older builds that staged beside Executable() == *.previous.
		if mistaken := legacyMistakenLatest(live); fileExists(mistaken) {
			log.Printf("appupdate: renaming mistaken staged file %s -> %s", mistaken, latest)
			if err := os.Rename(mistaken, latest); err != nil {
				return false, fmt.Errorf("repair staged name: %w", err)
			}
		}
	}
	if !fileExists(latest) {
		return false, nil
	}
	prev := previousPath(live)
	log.Printf("appupdate: applying staged binary %s -> %s (previous -> %s)", latest, live, prev)

	if fileExists(prev) {
		if err := os.Remove(prev); err != nil {
			return false, fmt.Errorf("remove previous binary: %w", err)
		}
	}
	// Rename by canonical live path, not necessarily os.Executable() (which may already be *.previous).
	if fileExists(live) {
		if err := os.Rename(live, prev); err != nil {
			return false, fmt.Errorf("move live binary to previous: %w", err)
		}
	}
	if err := os.Rename(latest, live); err != nil {
		if fileExists(prev) {
			if rerr := os.Rename(prev, live); rerr != nil {
				return false, fmt.Errorf("install staged binary: %v (restore failed: %v)", err, rerr)
			}
		}
		return false, fmt.Errorf("install staged binary: %w", err)
	}
	log.Printf("appupdate: staged update installed at %s; previous kept at %s", live, prev)
	return true, nil
}

func reexecLive(binaryPath string) error {
	live := normalizeLiveBinaryPath(binaryPath)
	log.Printf("appupdate: re-exec %s to run the newly installed binary", live)
	return syscall.Exec(live, os.Args, os.Environ())
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
	path := m.refreshBinaryPath()
	if path == "" {
		return fmt.Errorf("current executable path is unavailable")
	}
	live := normalizeLiveBinaryPath(path)
	prev, ok := findSibling(live, ".previous")
	if !ok {
		return fmt.Errorf("no previous binary available to roll back to (looked beside %s)", live)
	}
	staging := latestPath(live)
	tmp := live + ".rollback-tmp"
	_ = os.Remove(tmp)
	_ = os.Remove(staging)
	if mistaken := legacyMistakenLatest(live); fileExists(mistaken) {
		_ = os.Remove(mistaken)
	}

	if err := os.Rename(live, tmp); err != nil {
		return fmt.Errorf("stage current binary: %w", err)
	}
	if err := os.Rename(prev, live); err != nil {
		_ = os.Rename(tmp, live)
		return fmt.Errorf("restore previous binary: %w", err)
	}
	if err := os.Rename(tmp, prev); err != nil {
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
	log.Printf("appupdate: rollback prepared; restarting to run %s", live)
	return nil
}

// DiscardStaged removes a pending {binary}.latest without applying it.
func (m *Manager) DiscardStaged() error {
	path := m.refreshBinaryPath()
	if path == "" {
		return fmt.Errorf("current executable path is unavailable")
	}
	live := normalizeLiveBinaryPath(path)
	latest, ok := findSibling(live, ".latest")
	if !ok {
		return fmt.Errorf("no staged update to discard (looked beside %s)", live)
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
