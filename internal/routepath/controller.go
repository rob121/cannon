package routepath

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/paths"
	"github.com/rob121/cannon/internal/sites"
)

// Controller returns the active path for a controller action.
func Controller(ctx context.Context, controller, action string) string {
	fallback := builtin(controller, action)
	db, err := sites.DB(ctx)
	if err != nil {
		return link(fallback)
	}
	var route models.Route
	err = db.Where("controller = ? AND controller_action = ? AND status = ?", controller, action, models.StatusActive).
		Order("route_id asc").
		First(&route).Error
	if err != nil || route.Path == "" {
		return link(fallback)
	}
	return link(route.Path)
}

// ControllerWithSuffix appends a path segment after a controller route base.
func ControllerWithSuffix(ctx context.Context, controller, action, suffix string) string {
	base := Controller(ctx, controller, action)
	if suffix == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimPrefix(suffix, "/")
}

func builtin(controller, action string) string {
	switch controller + "/" + action {
	case "auth/login":
		return paths.AuthLogin
	case "auth/logout":
		return paths.AuthLogout
	case "auth/oauth":
		return paths.AuthOAuth
	case "auth/verify":
		return paths.AccountVerify
	case "auth/verify-resend":
		return paths.AccountVerifyResend
	case "auth/reset-request":
		return paths.AccountResetRequest
	case "auth/reset-submit":
		return paths.AccountResetSubmit
	case "content/category":
		return "/content/category/*"
	case "content/item":
		return "/content/item/*"
	case "content/tag":
		return "/content/tag/*"
	case "content/author":
		return "/content/author/*"
	case "content/search":
		return "/content/search"
	case "content/featured":
		return "/content/featured"
	case "content/feed":
		return "/content/feed/*"
	case "content/edit-new":
		return "/content/edit/new"
	case "content/edit":
		return "/content/edit/*"
	default:
		return ""
	}
}

func link(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasSuffix(path, "/*") {
		return path[:len(path)-2]
	}
	return path
}
