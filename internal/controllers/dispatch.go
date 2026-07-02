package controllers

import (
	"fmt"
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/templateengine"
)

// Dispatch runs a matched controller route.
func Dispatch(w http.ResponseWriter, r *http.Request, route models.Route, tpl *templateengine.Engine, viewerGroups []uint) error {
	if route.Controller == "" || route.ControllerAction == "" {
		return fmt.Errorf("controller route missing controller or action")
	}
	def, ctrl, ok := Lookup(route.Controller)
	if !ok {
		return fmt.Errorf("unknown controller %q", route.Controller)
	}
	action, ok := LookupAction(route.Controller, route.ControllerAction)
	if !ok {
		return fmt.Errorf("unknown action %q on controller %q", route.ControllerAction, route.Controller)
	}
	if !MethodAllowed(action, r.Method) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	ctx, err := NewContext(r, route, action, tpl, viewerGroups)
	if err != nil {
		return err
	}

	if action.RequireGuest && ctx.Authenticated() {
		return Redirect(http.StatusSeeOther, "/").Write(w, r, ctx)
	}
	if action.RequireAuth && !ctx.Authenticated() {
		allowed, err := settings.AllowLogin(ctx.GoContext())
		if err != nil {
			return Error(http.StatusInternalServerError, err.Error()).Write(w, r, ctx)
		}
		if !allowed {
			return Redirect(http.StatusSeeOther, "/").Write(w, r, ctx)
		}
		loginURL, err := AppendReturn(ctx.Site, "/login", r.URL.Path)
		if err != nil {
			loginURL = "/login"
		}
		return Redirect(http.StatusSeeOther, loginURL).Write(w, r, ctx)
	}
	if ctx.Authenticated() && !action.AllowUnverified {
		u, err := ctx.CurrentUser()
		if err == nil && !u.Validated {
			if allowed, _ := settings.AllowLogin(ctx.GoContext()); allowed {
				return Redirect(http.StatusSeeOther, "/login?verified=0").Write(w, r, ctx)
			}
			return Redirect(http.StatusSeeOther, "/").Write(w, r, ctx)
		}
	}

	_ = def
	return ctrl.Handle(ctx, route.ControllerAction).Write(w, r, ctx)
}
