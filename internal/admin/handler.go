package admin

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/help"
	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/middleware"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/user"
)

const pageSize = 20

type Handler struct {
	chain    *middleware.Chain
	activate ActivateFunc
	reload   ReloadFunc
}

func NewHandler(chain *middleware.Chain, activate ActivateFunc, reload ReloadFunc) *Handler {
	return &Handler{chain: chain, activate: activate, reload: reload}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/login" {
		http.NotFound(w, r)
		return
	}
	if !h.requireAccess(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/admin")
	switch {
	case path == "" || path == "/":
		h.dashboard(w, r)
	case strings.HasPrefix(path, "/users"):
		h.users(w, r, path)
	case strings.HasPrefix(path, "/groups"):
		h.groups(w, r, path)
	case strings.HasPrefix(path, "/roles"):
		h.roles(w, r, path)
	case strings.HasPrefix(path, "/routes"):
		h.routes(w, r, path)
	case strings.HasPrefix(path, "/items"):
		h.items(w, r, path)
	case strings.HasPrefix(path, "/categories"):
		h.categories(w, r, path)
	case strings.HasPrefix(path, "/tags"):
		h.tags(w, r, path)
	case strings.HasPrefix(path, "/field-groups"):
		h.fieldGroups(w, r, path)
	case strings.HasPrefix(path, "/comments"):
		h.comments(w, r, path)
	case strings.HasPrefix(path, "/media"):
		h.media(w, r, path)
	case strings.HasPrefix(path, "/templates"):
		h.templates(w, r, path)
	case strings.HasPrefix(path, "/menu-items"):
		h.menuItems(w, r, path)
	case strings.HasPrefix(path, "/menus"):
		h.menus(w, r, path)
	case strings.HasPrefix(path, "/extension-apps"):
		h.extensionApps(w, r, path)
	case strings.HasPrefix(path, "/extensions"):
		h.extensions(w, r, path)
	case strings.HasPrefix(path, "/authenticators"):
		h.authenticators(w, r, path)
	case strings.HasPrefix(path, "/profiles"):
		h.profiles(w, r, path)
	case strings.HasPrefix(path, "/languages"):
		h.languages(w, r, path)
	case strings.HasPrefix(path, "/help"):
		h.help(w, r, path)
	case strings.HasPrefix(path, "/configuration"):
		h.configuration(w, r, path)
	case strings.HasPrefix(path, "/notifications"):
		h.notifications(w, r, path)
	case strings.HasPrefix(path, "/blocks"):
		h.blocks(w, r, path)
	case strings.HasPrefix(path, "/sites"):
		h.sites(w, r, path)
	case strings.HasPrefix(path, "/system"):
		h.system(w, r, path)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) engine(r *http.Request, listExtra url.Values) (*templateengine.Engine, error) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return templateengine.New(site, nil, nil, templateengine.MergeFuncMaps(
		templateengine.CSRFFuncs(r),
		template.FuncMap{
		"lang": func(key string, pairs ...string) string {
			if mgr := middleware.LocaleFromContext(r.Context()); mgr != nil {
				return mgr.Fmt(key, pairs...)
			}
			return key
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: odd number of arguments")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key must be string")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
		"initials": func(parts ...string) string {
			var out string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					out += strings.ToUpper(p[:1])
				}
			}
			if out == "" {
				return "?"
			}
			if len(out) > 2 {
				return out[:2]
			}
			return out
		},
		"queryEscape": url.QueryEscape,
		"sortLink": func(basePath string, page int, currentSort, currentDir, col string) string {
			return sortLinkExtra(basePath, page, currentSort, currentDir, col, listExtra)
		},
		"listQuery": func(page int, sort, dir string) string {
			return listQueryExtra(page, sort, dir, listExtra)
		},
		"listQueryAmp": func(page int, sort, dir string) string {
			return listQueryAmpExtra(page, sort, dir, listExtra)
		},
		"internalHelpURL": func(folder, slug string) string {
			return help.ArticleURL(folder, slug)
		},
		"helpURL": func(extensionName, articlePath string) string {
			return extensions.HelpArticleURL(extensionName, articlePath)
		},
		"containsUint":  containsUint,
		"uintPtrEq":     uintPtrEq,
		"siteURL":       siteFrontendURL,
		"siteAdminURL":  siteAdminURL,
		"siteHostLabel": siteHostLabel,
		"joinRoleNames": func(roles []models.Role) string {
			names := make([]string, 0, len(roles))
			for _, role := range roles {
				names = append(names, role.Name)
			}
			return strings.Join(names, ", ")
		},
		"joinGroupNames": func(groups []models.Group) string {
			names := make([]string, 0, len(groups))
			for _, group := range groups {
				names = append(names, GroupDisplayName(group.Name))
			}
			return strings.Join(names, ", ")
		},
		"inString": func(list []string, v string) bool {
			for _, item := range list {
				if item == v {
					return true
				}
			}
			return false
		},
		"groupName": GroupDisplayName,
	})), nil
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, title, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["Title"] = title
	if _, ok := data["ActiveNav"]; !ok {
		data["ActiveNav"] = strings.ToLower(title)
	}
	for k, v := range layoutContext(r) {
		data[k] = v
	}
	var listExtra url.Values
	if sf, ok := data["SpaceFilter"].(string); ok && strings.TrimSpace(sf) != "" {
		listExtra = url.Values{"space": {sf}}
	}
	data["AdminExtensions"] = h.adminExtensionNav(r)
	if nav, ok := data["ActiveNav"].(string); ok {
		data["NavUsersOpen"] = usersNavOpen(nav)
		data["NavMenusOpen"] = menusNavOpen(nav)
		data["NavContentOpen"] = contentNavOpen(nav)
		data["NavSystemOpen"] = systemNavOpen(nav)
		data["NavExtensionAppsOpen"] = extensionAppsNavOpen(nav)
	}
	eng, err := h.engine(r, listExtra)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := eng.Render(w, "admin/layout.html", page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	h.renderDashboard(w, r, map[string]any{
		"ActiveNav": "dashboard",
		"Subtitle":  dashboardWelcomeSubtitle(r),
	})
}

func parsePage(r *http.Request) int {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		return 1
	}
	return page
}

// LoginHandler provides admin login.
type LoginHandler struct {
	Chain *middleware.Chain
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		svc, err := user.FromContext(r.Context())
		if err == nil {
			if userID, ok := svc.CurrentID(); ok {
				if allowed, err := CanAccessAdmin(r.Context(), userID, "/admin", false); err == nil && allowed {
					httpx.Redirect(w, r, "/admin")
					return
				}
			}
		}
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		svc, _ := user.FromContext(r.Context())
		username := strings.TrimSpace(r.FormValue("username"))
		loginArgs := hooks.RequestArgs(r)
		loginArgs["username"] = username
		loginArgs["context"] = "admin"
		if _, err := hooks.Fire(r.Context(), hooks.OnUserBeforeLogin, loginArgs); err != nil {
		if errors.Is(err, hooks.ErrAborted) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			token, _ := csrfToken(r)
			fmt.Fprint(w, loginHTML(token))
			return
		}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		u, err := user.AuthenticateLocal(r.Context(), username, r.FormValue("password"))
		if err == nil {
			afterArgs := map[string]any{
				"context":  "admin",
				"user_id":  u.UserID,
				"username": u.Username,
				"email":    u.Email,
			}
			if _, err := hooks.Fire(r.Context(), hooks.OnUserAfterLogin, afterArgs); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			allowed, err := CanAccessAdmin(r.Context(), u.UserID, "/admin", false)
			if err != nil || !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			_ = svc.Login(u.UserID)
			httpx.Redirect(w, r, "/admin")
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	token, _ := csrfToken(r)
	fmt.Fprint(w, loginHTML(token))
}

func csrfToken(r *http.Request) (string, error) {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return "", err
	}
	return svc.EnsureCSRFToken()
}

func loginHTML(token string) string {
	csrfField := string(csrf.HiddenField(token))
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sign In - Cannon Admin</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
  <link href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css" rel="stylesheet">
  <link href="/admin/assets/admin.css?v=6" rel="stylesheet">
</head>
<body class="admin-login-page">
  <div class="admin-install-wrap" style="max-width:420px">
    <div class="admin-install-brand">
      <div class="admin-brand-icon"><img class="admin-cannon-icon" src="/admin/assets/cannon-icon.svg?v=4" alt="" width="32" height="32"></div>
      <h1>Sign In</h1>
      <p>Enter your credentials to access the admin panel.</p>
    </div>
    <div class="admin-form-card admin-install-card">
      <div class="admin-form-card-body">
        <form method="post">
          ` + csrfField + `
          <div class="mb-3">
            <label class="admin-form-label">Username</label>
            <input class="admin-form-control" name="username" required autofocus>
          </div>
          <div class="mb-4">
            <label class="admin-form-label">Password</label>
            <input class="admin-form-control" type="password" name="password" required>
          </div>
          <button type="submit" class="btn-admin-primary w-100">Sign In</button>
        </form>
      </div>
    </div>
  </div>
</body>
</html>`
}
