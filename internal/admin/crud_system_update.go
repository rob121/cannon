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
		h.renderSystemUpdate(w, r, mgr, "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			h.renderSystemUpdate(w, r, mgr, err.Error())
			return
		}
		var err error
		switch r.FormValue("action") {
		case "check":
			err = mgr.Check()
		case "apply":
			err = mgr.Apply()
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		if err != nil {
			h.renderSystemUpdate(w, r, mgr, err.Error())
			return
		}
		redirectList(w, r, systemUpdateBase+"?saved=1")
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) renderSystemUpdate(w http.ResponseWriter, r *http.Request, mgr *appupdate.Manager, errMsg string) {
	state, _ := mgr.LoadState()
	data := formData(map[string]any{
		"ActiveNav":       "system_update",
		"Subtitle":        "Check for Cannon application updates and apply new releases.",
		"CurrentVersion":  mgr.CurrentVersion(),
		"Commit":          version.Commit,
		"UpdateURLBase":   mgr.UpdateURLBase(),
		"BinaryPath":      mgr.BinaryPath(),
		"State":           state,
		"BasePath":        systemUpdateBase,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if r.URL.Query().Get("saved") == "1" {
		data["Success"] = "Update action completed."
	}
	h.render(w, r, "Cannon Update", "admin/system_update.html", data)
}
