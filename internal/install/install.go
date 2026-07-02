package install

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rob121/cannon/internal/auth"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/session"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

// ActivateFunc reloads the server after install completes.
type ActivateFunc func(*config.App) error

// Handler serves the install wizard.
type Handler struct {
	cfg      *config.App
	tpl      *template.Template
	activate ActivateFunc
}

func NewHandler(cfg *config.App, activate ActivateFunc) (*Handler, error) {
	tpl, err := template.New("install").Funcs(template.FuncMap{
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
	}).Parse(installTemplates)
	if err != nil {
		return nil, err
	}
	return &Handler{cfg: cfg, tpl: tpl, activate: activate}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cfg, _, err := config.Reload(); err == nil {
		h.cfg = cfg
	}

	if !config.NeedsInstall(h.cfg) {
		if h.activate != nil {
			if err := h.activate(h.cfg); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		httpx.RedirectSeeOther(w, r, "/admin/login")
		return
	}

	switch r.URL.Path {
	case "/install", "/install/":
		if len(h.cfg.Sites) > 0 {
			httpx.Redirect(w, r, "/install/admin")
			return
		}
		if r.Method == http.MethodPost {
			h.handleSiteSetup(w, r)
			return
		}
		h.render(w, "site", nil)
	case "/install/admin":
		if r.Method == http.MethodPost {
			h.handleAdminSetup(w, r)
			return
		}
		h.render(w, "admin", nil)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleSiteSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	siteName := strings.TrimSpace(r.FormValue("site_name"))
	host := strings.TrimSpace(r.FormValue("host"))
	dataRoot := strings.TrimSpace(r.FormValue("data_root"))
	if dataRoot == "" {
		dataRoot = "./data"
	}
	dataRoot = filepath.Clean(dataRoot)

	dbType := strings.TrimSpace(r.FormValue("database_type"))
	if dbType == "" {
		dbType = "sqlite"
	}

	id := slugify(siteName)
	siteDir := filepath.Join(dataRoot, id)
	templateDir := filepath.Join(siteDir, "templates")
	assetsDir := filepath.Join(siteDir, "assets")
	tmpDir := filepath.Join(siteDir, "tmp")
	languageDir := filepath.Join(siteDir, "language")
	extensionsDir := filepath.Join(dataRoot, "extensions")
	socketsDir := filepath.Join(dataRoot, "sockets")

	dirs := []string{templateDir, assetsDir, tmpDir, languageDir, extensionsDir, socketsDir, siteDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	dsn := strings.TrimSpace(r.FormValue("database_dsn"))
	if dbType == "sqlite" {
		dsn = filepath.Join(siteDir, id+".sqlite")
	}
	if dbType == "mysql" {
		if dsn == "" {
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				r.FormValue("mysql_user"), r.FormValue("mysql_password"),
				r.FormValue("mysql_host"), defaultString(r.FormValue("mysql_port"), "3306"),
				r.FormValue("mysql_database"))
		}
	}
	if dbType == "postgres" {
		if dsn == "" {
			dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
				r.FormValue("postgres_host"), r.FormValue("postgres_user"), r.FormValue("postgres_password"),
				r.FormValue("postgres_database"), defaultString(r.FormValue("postgres_port"), "5432"))
		}
	}

	if host == "" {
		host = "http://localhost:8001"
	}

	site := config.SiteConfig{
		ID:          id,
		Name:        siteName,
		Host:        host,
		TemplateDir: templateDir,
		AssetsDir:   assetsDir,
		TmpDir:      tmpDir,
		LanguageDir: languageDir,
		Database: config.DatabaseConfig{
			Type: dbType,
			DSN:  dsn,
		},
	}

	h.cfg.InstallEnabled = true
	h.cfg.DataRoot = dataRoot
	h.cfg.Extensions = config.ExtensionsConfig{Dir: extensionsDir, SocketsDir: socketsDir}
	h.cfg.Sites = []config.SiteConfig{site}

	if err := config.Save(h.cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := lang.EnsureDefaults(languageDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := database.Connect(&site); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := database.Migrate(&site); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	db, _ := database.Get(site.ID)
	if err := auth.SeedAuthenticators(db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := roles.EnsureDefaults(db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := router.EnsureDefaultRoute(db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httpx.Redirect(w, r, "/install/admin")
}

func (h *Handler) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if len(h.cfg.Sites) == 0 {
		httpx.Redirect(w, r, "/install")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	site := &h.cfg.Sites[0]
	ctx := sites.WithContext(r.Context(), site)
	u, err := user.CreateLocalUser(ctx,
		r.FormValue("given_name"),
		r.FormValue("family_name"),
		r.FormValue("email"),
		r.FormValue("username"),
		r.FormValue("password"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := roles.AssignAdmin(ctx, u.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log the new admin in before switching to normal routes.
	store, err := session.NewStore(site, h.cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sessionID, err := store.Create()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	svc, err := user.NewService(store, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := svc.Login(u.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     store.CookieName(),
		Value:    sessionID,
		Path:     "/",
		MaxAge:   store.MaxAge(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	h.cfg.InstallEnabled = false
	if err := config.Save(h.cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.activate != nil {
		if err := h.activate(h.cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	httpx.Redirect(w, r, "/admin")
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func slugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	v = re.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	if v == "" {
		return "site"
	}
	return v
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

const installTemplates = `
{{define "install_head"}}<!doctype html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - Cannon</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
<link href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css" rel="stylesheet">
<link href="/admin/assets/admin.css?v=9" rel="stylesheet">
</head>{{end}}

{{define "wizard_header"}}
<div class="admin-wizard-header">
  <div class="admin-brand-icon"><img class="admin-cannon-icon" src="/admin/assets/cannon-icon.svg?v=4" alt="" width="32" height="32"></div>
  <h1>{{.Title}}</h1>
  <p>{{.Subtitle}}</p>
</div>
{{end}}

{{define "wizard_steps"}}
<nav class="admin-wizard-steps" aria-label="Setup progress">
  <div class="admin-wizard-step{{if eq . 1}} active{{else if gt . 1}} completed{{end}}">
    <span class="admin-wizard-step-marker">{{if gt . 1}}<i class="bi bi-check-lg"></i>{{else}}1{{end}}</span>
    <span class="admin-wizard-step-label">Site Setup</span>
  </div>
  <div class="admin-wizard-step-line{{if gt . 1}} completed{{end}}" aria-hidden="true"></div>
  <div class="admin-wizard-step{{if eq . 2}} active{{end}}">
    <span class="admin-wizard-step-marker">2</span>
    <span class="admin-wizard-step-label">Administrator</span>
  </div>
</nav>
{{end}}

{{define "site"}}{{template "install_head" (dict "Title" "Install")}}
<body class="admin-body admin-install-page">
<div class="admin-wizard">
  {{template "wizard_header" (dict "Title" "Cannon Setup Wizard" "Subtitle" "Configure your first site in just a few steps.")}}
  {{template "wizard_steps" 1}}
  <div class="admin-wizard-card">
    <div class="admin-wizard-card-header">
      <h2 class="admin-wizard-card-title">Site Configuration</h2>
      <p class="admin-wizard-card-desc">Define where Cannon stores data and how it connects to the database.</p>
    </div>
    <div class="admin-wizard-card-body">
      <form method="post" action="/install">
        <div class="row g-3">
          <div class="col-md-12">
            <label class="admin-form-label">Site Name</label>
            <input class="admin-form-control" name="site_name" required>
          </div>
          <div class="col-md-12">
            <label class="admin-form-label">Data Root Path</label>
            <input class="admin-form-control" name="data_root" value="./data" required>
            <p class="admin-form-help">Base directory for site data, extensions, and sockets.</p>
          </div>
          <div class="col-md-12">
            <label class="admin-form-label">Host URL</label>
            <input class="admin-form-control" name="host" placeholder="http://127.0.0.1:8001">
            <p class="admin-form-help">Must match the hostname you use in the browser.</p>
          </div>
          <div class="col-md-12">
            <label class="admin-form-label">Database Type</label>
            <select class="admin-form-control" name="database_type" id="dbtype">
              <option value="sqlite">SQLite</option>
              <option value="mysql">MySQL</option>
              <option value="postgres">PostgreSQL</option>
            </select>
          </div>
          <div class="col-12" id="mysql-fields" style="display:none">
            <div class="row g-3">
              <div class="col-md-6"><label class="admin-form-label">MySQL Host</label><input class="admin-form-control" name="mysql_host" value="127.0.0.1"></div>
              <div class="col-md-6"><label class="admin-form-label">MySQL Port</label><input class="admin-form-control" name="mysql_port" value="3306"></div>
              <div class="col-md-6"><label class="admin-form-label">MySQL Database</label><input class="admin-form-control" name="mysql_database"></div>
              <div class="col-md-6"><label class="admin-form-label">MySQL User</label><input class="admin-form-control" name="mysql_user"></div>
              <div class="col-md-12"><label class="admin-form-label">MySQL Password</label><input class="admin-form-control" name="mysql_password" type="password"></div>
            </div>
          </div>
          <div class="col-12" id="postgres-fields" style="display:none">
            <div class="row g-3">
              <div class="col-md-6"><label class="admin-form-label">Postgres Host</label><input class="admin-form-control" name="postgres_host" value="127.0.0.1"></div>
              <div class="col-md-6"><label class="admin-form-label">Postgres Port</label><input class="admin-form-control" name="postgres_port" value="5432"></div>
              <div class="col-md-6"><label class="admin-form-label">Postgres Database</label><input class="admin-form-control" name="postgres_database"></div>
              <div class="col-md-6"><label class="admin-form-label">Postgres User</label><input class="admin-form-control" name="postgres_user"></div>
              <div class="col-md-12"><label class="admin-form-label">Postgres Password</label><input class="admin-form-control" name="postgres_password" type="password"></div>
            </div>
          </div>
        </div>
        <div class="admin-wizard-footer">
          <button type="submit" class="btn-admin-primary">Next <i class="bi bi-arrow-right"></i></button>
        </div>
      </form>
    </div>
  </div>
</div>
<script>
document.getElementById('dbtype').addEventListener('change', function(){
  document.getElementById('mysql-fields').style.display = this.value === 'mysql' ? 'block' : 'none';
  document.getElementById('postgres-fields').style.display = this.value === 'postgres' ? 'block' : 'none';
});
</script>
</body></html>{{end}}

{{define "admin"}}{{template "install_head" (dict "Title" "Create Administrator")}}
<body class="admin-body admin-install-page">
<div class="admin-wizard">
  {{template "wizard_header" (dict "Title" "Cannon Setup Wizard" "Subtitle" "Configure your first site in just a few steps.")}}
  {{template "wizard_steps" 2}}
  <div class="admin-wizard-card">
    <div class="admin-wizard-card-header">
      <h2 class="admin-wizard-card-title">Administrator Account</h2>
      <p class="admin-wizard-card-desc">Create the first admin user with full access to the panel.</p>
    </div>
    <div class="admin-wizard-card-body">
      <form method="post" action="/install/admin">
        <div class="row g-3">
          <div class="col-md-6"><label class="admin-form-label">Given Name</label><input class="admin-form-control" name="given_name" required></div>
          <div class="col-md-6"><label class="admin-form-label">Family Name</label><input class="admin-form-control" name="family_name"></div>
          <div class="col-md-6"><label class="admin-form-label">Email</label><input class="admin-form-control" type="email" name="email" required></div>
          <div class="col-md-6"><label class="admin-form-label">Username</label><input class="admin-form-control" name="username" required></div>
          <div class="col-md-12"><label class="admin-form-label">Password</label><input class="admin-form-control" type="password" name="password" required></div>
        </div>
        <div class="admin-wizard-footer">
          <button type="submit" class="btn-admin-primary">Finish Install</button>
        </div>
      </form>
    </div>
  </div>
</div>
</body></html>{{end}}
`
