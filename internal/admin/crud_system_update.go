package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/appupdate"
	"github.com/rob121/cannon/internal/version"
)

const systemUpdateBase = "/admin/system/update"

func (h *Handler) systemUpdate(w http.ResponseWriter, r *http.Request) {
	mgr := h.chain.AppUpdates()
	switch r.Method {
	case http.MethodGet:
		h.renderSystemUpdate(w, r, mgr, "", "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			h.renderSystemUpdate(w, r, mgr, err.Error(), "")
			return
		}
		var (
			err     error
			success string
			restart bool
		)
		switch r.FormValue("action") {
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
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		if err != nil {
			h.renderSystemUpdate(w, r, mgr, err.Error(), "")
			return
		}
		if restart {
			mgr.ScheduleRestart()
		}
		h.renderSystemUpdate(w, r, mgr, "", success)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) renderSystemUpdate(w http.ResponseWriter, r *http.Request, mgr *appupdate.Manager, errMsg, success string) {
	status := mgr.Status()
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
