package admin

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/themes"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

const configurationBase = "/admin/configuration"

type configurationNavItem struct {
	Label   string
	URL     string
	Active  bool
	Heading bool
}

func (h *Handler) configuration(w http.ResponseWriter, r *http.Request, path string) {
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())

	parts := pathParts("/configuration", path)
	switch {
	case len(parts) == 0:
		redirectList(w, r, configurationBase+"/global/general")
	case parts[0] == "global":
		sectionID := ""
		if len(parts) >= 2 {
			sectionID = parts[1]
		}
		querySection := strings.TrimSpace(r.URL.Query().Get("section"))
		if sectionID == "" {
			sectionID = querySection
		}
		if r.Method == http.MethodGet && len(parts) < 2 && querySection != "" {
			redirectList(w, r, configurationBase+"/global/"+url.PathEscape(querySection))
			return
		}
		h.configurationGlobal(w, r, extMgr, sectionID)
	case parts[0] == "extensions" && len(parts) >= 2:
		sectionID := ""
		if len(parts) >= 3 {
			sectionID = parts[2]
		}
		querySection := strings.TrimSpace(r.URL.Query().Get("section"))
		if sectionID == "" {
			sectionID = querySection
		}
		if r.Method == http.MethodGet && len(parts) < 3 && querySection != "" {
			redirectList(w, r, configurationBase+"/extensions/"+url.PathEscape(parts[1])+"/"+url.PathEscape(querySection))
			return
		}
		h.configurationExtension(w, r, extMgr, parts[1], sectionID)
	default:
		h.notFound(w, r)
	}
}

func (h *Handler) configurationGlobal(w http.ResponseWriter, r *http.Request, extMgr *extensions.Manager, sectionID string) {
	sectionID = strings.TrimSpace(sectionID)
	if sectionID == "" {
		redirectList(w, r, configurationBase+"/global/general")
		return
	}
	def, ok, err := settings.GlobalDefinition(sectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		h.notFound(w, r)
		return
	}

	store := settings.NewStore()
	postURL := configurationBase + "/global/" + url.PathEscape(sectionID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data := settings.FormDataFromRequest(r, def.Schema)
		if sectionID == content.SettingsSection {
			data = content.MergeConfigurationPermissionFormData(data, r)
		}
		if err := settings.Save(r.Context(), store, settings.ScopeGlobal, sectionID, data); err != nil {
			h.renderConfigurationError(w, r, extMgr, "global", sectionID, def.Title, "", err.Error())
			return
		}
		redirectList(w, r, postURL+"?saved=1")
		return
	}

	defs, err := settings.GlobalDefinitions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	doc, err := settings.Document(r.Context(), store, settings.ScopeGlobal, defs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	section, ok := settings.FindSection(doc, sectionID)
	if !ok {
		h.notFound(w, r)
		return
	}
	if sectionID == settings.SectionGeneral {
		site, _ := sites.FromContext(r.Context())
		templateDir := ""
		if site != nil {
			templateDir = site.TemplateDir
		}
		if patched, err := themes.PatchGeneralSchema(section, templateDir); err == nil {
			section = patched
		}
	}
	formHTML, err := settings.RenderForm(section, postURL, configurationCSRFToken(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sectionID == content.SettingsSection {
		db, _ := sites.DB(r.Context())
		createIDs, editIDs, publishIDs, _ := content.LoadPermissionGroupIDs(r.Context())
		settings, _ := content.LoadSettings(r.Context())
		formHTML = content.InjectConfigurationPermissionFields(formHTML, loadActiveGroups(db), loadActiveProfiles(db), createIDs, editIDs, publishIDs, settings.AuthorProfileID)
	}
	h.renderConfiguration(w, r, extMgr, section.Title, "global", section.ID, section.Title, "", formHTML, r.URL.Query().Get("saved") == "1", "", nil)
}

func (h *Handler) configurationExtension(w http.ResponseWriter, r *http.Request, extMgr *extensions.Manager, rawName, sectionID string) {
	name, err := url.PathUnescape(rawName)
	if err != nil || name == "" {
		h.notFound(w, r)
		return
	}

	rt, ok := extMgr.Runtime(name)
	if !ok || rt.Capabilities.Configuration == "" {
		h.notFound(w, r)
		return
	}

	sectionID = strings.TrimSpace(sectionID)

	doc, err := extMgr.FetchConfiguration(rt.Model.Socket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if sectionID == "" {
		section, ok := settings.FindSection(doc, "")
		if !ok {
			h.notFound(w, r)
			return
		}
		redirectList(w, r, configurationBase+"/extensions/"+url.PathEscape(name)+"/"+url.PathEscape(section.ID))
		return
	}

	section, ok := settings.FindSection(doc, sectionID)
	if !ok {
		h.notFound(w, r)
		return
	}
	postURL := configurationBase + "/extensions/" + url.PathEscape(name) + "/" + url.PathEscape(section.ID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		saveSection := strings.TrimSpace(r.FormValue("section"))
		if saveSection == "" {
			saveSection = section.ID
		}
		data := settings.FormDataFromRequest(r, section.Schema)
		raw, _ := json.Marshal(data)
		if err := extMgr.SaveConfiguration(rt.Model.Socket, extension.ConfigurationSaveRequest{
			Section: saveSection,
			Data:    raw,
		}); err != nil {
			h.renderConfigurationError(w, r, extMgr, "extension", section.ID, section.Title, name, err.Error())
			return
		}
		redirectList(w, r, postURL+"?saved=1")
		return
	}

	formHTML, err := settings.RenderForm(section, postURL, configurationCSRFToken(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderConfiguration(w, r, extMgr, section.Title, "extension", section.ID, section.Title, name, formHTML, r.URL.Query().Get("saved") == "1", "", doc.Sections)
}

func configurationCSRFToken(r *http.Request) string {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return ""
	}
	token, err := svc.EnsureCSRFToken()
	if err != nil {
		return ""
	}
	return token
}

func (h *Handler) renderConfigurationError(w http.ResponseWriter, r *http.Request, extMgr *extensions.Manager, scope, sectionID, sectionTitle, extensionName, errMsg string) {
	h.renderConfiguration(w, r, extMgr, sectionTitle, scope, sectionID, sectionTitle, extensionName, "", false, errMsg, nil)
}

func (h *Handler) renderConfiguration(w http.ResponseWriter, r *http.Request, extMgr *extensions.Manager, title, scope, sectionID, sectionTitle, extensionName, formHTML string, saved bool, errMsg string, extensionSections []extension.ConfigurationSection) {
	db, _ := sites.DB(r.Context())
	subtitle := "Global configuration settings."
	if scope == "extension" {
		label := extensionMenuLabel(extensionName, extensionMenuNames(db))
		subtitle = "Settings for the " + label + " extension."
	}
	data := map[string]any{
		"ActiveNav":         "configuration",
		"Subtitle":          subtitle,
		"ConfigurationBase": configurationBase,
		"Scope":             scope,
		"SectionID":         sectionID,
		"SectionTitle":      sectionTitle,
		"ExtensionName":     extensionName,
		"NavItems":          configurationNav(extMgr, db, scope, sectionID, extensionName, extensionSections),
		"FormHTML":          template.HTML(formHTML),
		"Saved":             saved,
	}
	if extensionName != "" {
		data["ExtensionLabel"] = extensionMenuLabel(extensionName, extensionMenuNames(db))
		data["ExtensionURL"] = configurationBase + "/extensions/" + url.PathEscape(extensionName)
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/configuration.html", data)
}

func configurationNav(extMgr *extensions.Manager, db *gorm.DB, scope, activeSection, extensionName string, extensionSections []extension.ConfigurationSection) []configurationNavItem {
	menuNames := extensionMenuNames(db)
	items := []configurationNavItem{
		{Label: "Global", Heading: true},
	}
	defs, err := settings.GlobalDefinitions()
	if err == nil {
		for _, def := range defs {
			items = append(items, configurationNavItem{
				Label:  def.Title,
				URL:    configurationBase + "/global/" + url.PathEscape(def.ID),
				Active: scope == "global" && def.ID == activeSection,
			})
		}
	}
	runtimes := extMgr.ConfigurationRuntimes()
	if len(runtimes) > 0 {
		items = append(items, configurationNavItem{Label: "Extensions", Heading: true})
		for _, rt := range runtimes {
			label := extensionMenuLabel(rt.Model.Name, menuNames)
			activeExtension := scope == "extension" && rt.Model.Name == extensionName
			items = append(items, configurationNavItem{
				Label:  label,
				URL:    configurationBase + "/extensions/" + url.PathEscape(rt.Model.Name),
				Active: activeExtension && len(extensionSections) <= 1,
			})
			if activeExtension && len(extensionSections) > 1 {
				for _, sec := range extensionSections {
					items = append(items, configurationNavItem{
						Label:  sec.Title,
						URL:    configurationBase + "/extensions/" + url.PathEscape(rt.Model.Name) + "/" + url.PathEscape(sec.ID),
						Active: sec.ID == activeSection,
					})
				}
			}
		}
	}
	return items
}
