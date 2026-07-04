package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/paths"
	"gorm.io/gorm"
)

// BuiltinRoute describes a seeded controller route managed by Cannon.
type BuiltinRoute struct {
	Name             string
	Path             string
	Prefix           string
	Controller       string
	ControllerAction string
	Methods          string
}

var builtinRouteDefs = []BuiltinRoute{
	{Name: "Content Category", Path: "/content/category/*", Prefix: "content", Controller: "content", ControllerAction: "category", Methods: http.MethodGet},
	{Name: "Content Item", Path: "/content/item/*", Prefix: "content", Controller: "content", ControllerAction: "item", Methods: "GET, POST"},
	{Name: "Content Tag", Path: "/content/tag/*", Prefix: "content", Controller: "content", ControllerAction: "tag", Methods: http.MethodGet},
	{Name: "Content Author", Path: "/content/author/*", Prefix: "content", Controller: "content", ControllerAction: "author", Methods: http.MethodGet},
	{Name: "Content Search", Path: "/content/search", Prefix: "content", Controller: "content", ControllerAction: "search", Methods: http.MethodGet},
	{Name: "Content Featured", Path: "/content/featured", Prefix: "content", Controller: "content", ControllerAction: "featured", Methods: http.MethodGet},
	{Name: "Content Feed", Path: "/content/feed/*", Prefix: "content", Controller: "content", ControllerAction: "feed", Methods: http.MethodGet},
	{Name: "Content Create", Path: "/content/edit/new", Prefix: "content", Controller: "content", ControllerAction: "edit-new", Methods: "GET, POST"},
	{Name: "Content Edit", Path: "/content/edit/*", Prefix: "content", Controller: "content", ControllerAction: "edit", Methods: "GET, POST"},
	{Name: "Content Preview", Path: "/content/preview/*", Prefix: "content", Controller: "content", ControllerAction: "preview", Methods: http.MethodGet},
	{Name: "Login", Path: paths.AuthLogin, Prefix: "auth", Controller: "auth", ControllerAction: "login", Methods: "GET, POST"},
	{Name: "Logout", Path: paths.AuthLogout, Prefix: "auth", Controller: "auth", ControllerAction: "logout", Methods: "GET, POST"},
	{Name: "OAuth Sign In", Path: paths.AuthOAuth, Prefix: "auth", Controller: "auth", ControllerAction: "oauth", Methods: http.MethodGet},
	{Name: "Verify Account", Path: paths.AccountVerify, Prefix: "account", Controller: "auth", ControllerAction: "verify", Methods: http.MethodGet},
	{Name: "Verification Pending", Path: paths.AccountVerifyResend, Prefix: "account", Controller: "auth", ControllerAction: "verify-resend", Methods: http.MethodGet},
	{Name: "Reset Password", Path: paths.AccountResetRequest, Prefix: "account", Controller: "auth", ControllerAction: "reset-request", Methods: "GET, POST"},
	{Name: "Reset Password Submit", Path: paths.AccountResetSubmit, Prefix: "account", Controller: "auth", ControllerAction: "reset-submit", Methods: "GET, POST"},
	{Name: "MFA Challenge", Path: paths.AccountMFAChallenge, Prefix: "account", Controller: "auth", ControllerAction: "mfa-challenge", Methods: "GET, POST"},
	{Name: "Account Security", Path: paths.AccountSecurity, Prefix: "account", Controller: "auth", ControllerAction: "security", Methods: http.MethodGet},
	{Name: "Account Profile", Path: paths.AccountProfile, Prefix: "account", Controller: "auth", ControllerAction: "profile", Methods: "GET, POST"},
	{Name: "TOTP Setup", Path: paths.AccountSecurityTOTP, Prefix: "account", Controller: "auth", ControllerAction: "security-totp", Methods: http.MethodPost},
	{Name: "Passkey Setup", Path: paths.AccountSecurityPasskey, Prefix: "account", Controller: "auth", ControllerAction: "security-passkey", Methods: "GET, POST"},
	{Name: "Passkey Login", Path: paths.AuthPasskeyLogin, Prefix: "auth", Controller: "auth", ControllerAction: "passkey-login", Methods: http.MethodPost},
}

// BuiltinControllerRoutes returns built-in content, auth, and account controller routes.
func BuiltinControllerRoutes() []BuiltinRoute {
	out := make([]BuiltinRoute, len(builtinRouteDefs))
	copy(out, builtinRouteDefs)
	return out
}

// AuthRoutePrefix is the URL prefix for authentication actions (login, logout).
const AuthRoutePrefix = "/auth"

// AccountRoutePrefix is the URL prefix for account lifecycle actions (verify, reset).
const AccountRoutePrefix = "/account"

// ContentRoutePrefix is the URL prefix for content controller actions.
const ContentRoutePrefix = "/content"

// IsBuiltinControllerRoute reports whether a DB route is a seeded built-in controller route.
func IsBuiltinControllerRoute(row models.Route) bool {
	if row.Type != models.RouteTypeController {
		return false
	}
	for _, br := range builtinRouteDefs {
		if row.Controller == br.Controller && row.ControllerAction == br.ControllerAction && row.Path == br.Path {
			return true
		}
	}
	return false
}

// ConflictsWithReservedPath reports whether a path overlaps reserved system or built-in routes.
func ConflictsWithReservedPath(path string) bool {
	path = normalizeReservedPath(path)
	if path == "" {
		return false
	}
	for _, sr := range SystemRoutes() {
		if pathsOverlap(path, sr.Path) {
			return true
		}
	}
	for _, br := range BuiltinControllerRoutes() {
		if pathsOverlap(path, br.Path) {
			return true
		}
	}
	return false
}

func normalizeReservedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func pathsOverlap(a, b string) bool {
	a = normalizeReservedPath(a)
	b = normalizeReservedPath(b)
	if a == b {
		return true
	}
	if strings.HasSuffix(b, "/*") {
		prefix := strings.TrimSuffix(b, "/*")
		if prefix == "" {
			prefix = "/"
		}
		return a == prefix || strings.HasPrefix(a, prefix+"/")
	}
	if strings.HasSuffix(a, "/*") {
		return pathsOverlap(b, a)
	}
	return false
}

// BuiltinRouteModels returns seed rows for EnsureDefaultRoute.
func BuiltinRouteModels() []models.Route {
	routes := BuiltinControllerRoutes()
	out := make([]models.Route, 0, len(routes))
	for _, br := range routes {
		out = append(out, models.Route{
			Name:             br.Name,
			Path:             br.Path,
			Type:             models.RouteTypeController,
			Status:           models.StatusActive,
			Controller:       br.Controller,
			ControllerAction: br.ControllerAction,
		})
	}
	return out
}

func migrateLegacyAuthPaths(db *gorm.DB) error {
	type move struct {
		from string
		to   string
	}
	for _, m := range []move{{"/login", paths.AuthLogin}, {"/logout", paths.AuthLogout}} {
		var row models.Route
		err := db.Where("path = ? AND controller = ?", m.from, "auth").First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		var existing models.Route
		err = db.Where("path = ?", m.to).First(&existing).Error
		if err == nil && existing.RouteID != row.RouteID {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row.Path = m.to
		if err := db.Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
