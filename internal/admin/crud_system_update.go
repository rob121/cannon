package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/appupdate"
	"github.com/rob121/cannon/internal/version"
)

const systemUpdateBase = "/admin/system/update"

type systemUpdateStatusJSON struct {
	CurrentVersion  string     `json:"current_version"`
	Commit          string     `json:"commit"`
	BinaryPath      string     `json:"binary_path"`
	LatestPath      string     `json:"latest_path"`
	PreviousPath    string     `json:"previous_path"`
	LatestVersion   string     `json:"latest_version"`
	UpdateAvailable bool       `json:"update_available"`
	HasStaged       bool       `json:"has_staged"`
	HasPrevious     bool       `json:"has_previous"`
	StagedVersion   string     `json:"staged_version"`
	RestartRequired bool       `json:"restart_required"`
	UpdateError     string     `json:"update_error"`
	UpdateCheckedAt *time.Time `json:"update_checked_at,omitempty"`
	UpdateURLBase   string     `json:"update_url_base"`
}

type systemUpdateActionJSON struct {
	OK         bool                   `json:"ok"`
	Action     string                 `json:"action"`
	Message    string                 `json:"message,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Restarting bool                   `json:"restarting,omitempty"`
	Status     systemUpdateStatusJSON `json:"status"`
}

func (h *Handler) systemUpdate(w http.ResponseWriter, r *http.Request) {
	mgr := h.chain.AppUpdates()
	switch r.Method {
	case http.MethodGet:
		h.renderSystemUpdate(w, r, mgr, "", "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			if wantsJSON(r) {
				writeSystemUpdateJSON(w, http.StatusBadRequest, systemUpdateActionJSON{
					OK:     false,
					Error:  err.Error(),
					Status: systemUpdateStatusPayload(mgr),
				})
				return
			}
			h.renderSystemUpdate(w, r, mgr, err.Error(), "")
			return
		}
		var (
			err     error
			success string
			restart bool
			action  = r.FormValue("action")
		)
		switch action {
		case "check":
			err = mgr.Check()
			if err == nil {
				success = "Update check completed."
			}
		case "apply":
			err = mgr.Apply()
			if err == nil {
				success = "Update staged as cannon.latest. Process is restarting so the new binary can be applied."
				restart = true
			}
		case "rollback":
			err = mgr.Rollback()
			if err == nil {
				success = "Rolled back to cannon.previous. Process is restarting."
				restart = true
			}
		case "discard":
			err = mgr.DiscardStaged()
			if err == nil {
				success = "Staged update discarded."
			}
		default:
			if wantsJSON(r) {
				writeSystemUpdateJSON(w, http.StatusBadRequest, systemUpdateActionJSON{
					OK:     false,
					Action: action,
					Error:  "unknown action",
					Status: systemUpdateStatusPayload(mgr),
				})
				return
			}
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		if err != nil {
			if wantsJSON(r) {
				writeSystemUpdateJSON(w, http.StatusBadRequest, systemUpdateActionJSON{
					OK:     false,
					Action: action,
					Error:  err.Error(),
					Status: systemUpdateStatusPayload(mgr),
				})
				return
			}
			h.renderSystemUpdate(w, r, mgr, err.Error(), "")
			return
		}
		if restart {
			mgr.ScheduleRestart()
		}
		if wantsJSON(r) {
			writeSystemUpdateJSON(w, http.StatusOK, systemUpdateActionJSON{
				OK:         true,
				Action:     action,
				Message:    success,
				Restarting: restart,
				Status:     systemUpdateStatusPayload(mgr),
			})
			return
		}
		h.renderSystemUpdate(w, r, mgr, "", success)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) systemUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mgr := h.chain.AppUpdates()
	writeSystemUpdateJSON(w, http.StatusOK, systemUpdateStatusPayload(mgr))
}

func systemUpdateStatusPayload(mgr *appupdate.Manager) systemUpdateStatusJSON {
	status := mgr.Status()
	return systemUpdateStatusJSON{
		CurrentVersion:  status.CurrentVersion,
		Commit:          version.Commit,
		BinaryPath:      status.BinaryPath,
		LatestPath:      status.LatestPath,
		PreviousPath:    status.PreviousPath,
		LatestVersion:   status.State.LatestVersion,
		UpdateAvailable: status.State.UpdateAvailable,
		HasStaged:       status.HasStaged,
		HasPrevious:     status.HasPrevious,
		StagedVersion:   status.StagedVersion,
		RestartRequired: status.State.RestartRequired,
		UpdateError:     status.State.UpdateError,
		UpdateCheckedAt: status.State.UpdateCheckedAt,
		UpdateURLBase:   mgr.UpdateURLBase(),
	}
}

func writeSystemUpdateJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func wantsJSON(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *Handler) renderSystemUpdate(w http.ResponseWriter, r *http.Request, mgr *appupdate.Manager, errMsg, success string) {
	status := mgr.Status()
	w.Header().Set("Cache-Control", "no-store")
	data := formData(map[string]any{
		"ActiveNav":      "system_update",
		"Subtitle":       "Stage Cannon releases beside the running binary, then restart to apply. Rollback uses cannon.previous.",
		"CurrentVersion": status.CurrentVersion,
		"Commit":         version.Commit,
		"UpdateURLBase":  mgr.UpdateURLBase(),
		"BinaryPath":     status.BinaryPath,
		"LatestPath":     status.LatestPath,
		"PreviousPath":   status.PreviousPath,
		"State":          status.State,
		"HasStaged":      status.HasStaged,
		"HasPrevious":    status.HasPrevious,
		"StagedVersion":  status.StagedVersion,
		"BasePath":       systemUpdateBase,
		"StatusPath":     systemUpdateBase + "/status",
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if success != "" {
		data["Success"] = success
	} else if r.URL.Query().Get("saved") == "1" {
		data["Success"] = "Update action completed."
	}
	h.render(w, r, "Cannon Update", "admin/system_update.html", data)
}
