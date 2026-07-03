package content

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// MergeConfigurationPermissionFormData stores permission group ids from the admin form.
func MergeConfigurationPermissionFormData(data map[string]any, r *http.Request) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	data[settingCreateGroupIDs] = formUintListAny(r, "create_group_ids")
	data[settingEditGroupIDs] = formUintListAny(r, "edit_group_ids")
	data[settingPublishGroupIDs] = formUintListAny(r, "publish_group_ids")
	if v, err := strconv.ParseUint(strings.TrimSpace(r.FormValue(settingAuthorProfileID)), 10, 64); err == nil && v > 0 {
		data[settingAuthorProfileID] = v
	} else {
		data[settingAuthorProfileID] = nil
	}
	return data
}

// InjectConfigurationPermissionFields appends permission and author profile fields to a JSON Forms configuration form.
func InjectConfigurationPermissionFields(formHTML string, groups []models.Group, profiles []models.Profile, createIDs, editIDs, publishIDs []uint, authorProfileID uint) string {
	block := `<div class="row g-3 admin-jsonforms-extra">` +
		renderPermissionFieldsHTML(groups, createIDs, editIDs, publishIDs) +
		renderAuthorProfileFieldHTML(profiles, authorProfileID) +
		`</div>`
	if idx := configurationSubmitIndex(formHTML); idx >= 0 {
		return formHTML[:idx] + block + formHTML[idx:]
	}
	end := strings.LastIndex(formHTML, "</form>")
	if end < 0 {
		return formHTML + block
	}
	return formHTML[:end] + block + formHTML[end:]
}

func configurationSubmitIndex(formHTML string) int {
	marker := `class="btn btn-admin-primary" type="submit"`
	idx := strings.Index(formHTML, marker)
	if idx < 0 {
		return strings.Index(formHTML, `type="submit"`)
	}
	if start := strings.LastIndex(formHTML[:idx], "<button"); start >= 0 {
		return start
	}
	return idx
}

func renderPermissionFieldsHTML(groups []models.Group, createIDs, editIDs, publishIDs []uint) string {
	var b strings.Builder
	b.WriteString(`<div class="col-12"><div class="admin-form-divider"></div></div>`)
	b.WriteString(`<div class="col-12"><h3 class="admin-form-section-title">Frontend Permissions</h3>`)
	b.WriteString(`<p class="admin-form-section-desc">Choose which groups may create, edit, or publish items on the public site. Leave a permission empty to fall back to role-based defaults.</p></div>`)
	b.WriteString(renderPermissionField("Create Items", "create_group_ids", "Groups allowed to create new items.", groups, createIDs))
	b.WriteString(renderPermissionField("Edit Items", "edit_group_ids", "Groups allowed to edit items. Authors can still edit their own items when role defaults apply.", groups, editIDs))
	b.WriteString(renderPermissionField("Publish Items", "publish_group_ids", "Groups allowed to publish or feature items on the frontend.", groups, publishIDs))
	return b.String()
}

func renderPermissionField(label, fieldName, help string, groups []models.Group, selected []uint) string {
	var b strings.Builder
	b.WriteString(`<div class="col-12"><label class="admin-form-label">`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</label><p class="admin-form-help mb-3">`)
	b.WriteString(html.EscapeString(help))
	b.WriteString(`</p><div class="admin-toggle-grid admin-toggle-grid-sidebar">`)
	if len(groups) == 0 {
		b.WriteString(`<p class="admin-cell-muted">No groups defined yet.</p>`)
	} else {
		for _, group := range groups {
			checked := ""
			if containsUint(selected, group.GroupID) {
				checked = " checked"
			}
			b.WriteString(fmt.Sprintf(`<label class="admin-form-toggle admin-form-toggle-inline"><input type="checkbox" class="admin-form-toggle-input" name="%s" value="%d"%s><span class="admin-form-toggle-track" aria-hidden="true"><span class="admin-form-toggle-thumb"></span></span><span class="admin-form-toggle-label">%s</span></label>`,
				html.EscapeString(fieldName), group.GroupID, checked, html.EscapeString(groupDisplayName(group.Name))))
		}
	}
	b.WriteString(`</div></div>`)
	return b.String()
}

func renderAuthorProfileFieldHTML(profiles []models.Profile, selectedID uint) string {
	var b strings.Builder
	b.WriteString(`<div class="col-12"><div class="admin-form-divider"></div></div>`)
	b.WriteString(`<div class="col-12"><h3 class="admin-form-section-title">Author Profiles</h3>`)
	b.WriteString(`<p class="admin-form-section-desc">Select the profile schema whose fields are shown as author information on item and author pages. Create profile schemas under Profiles in the admin.</p></div>`)
	b.WriteString(`<div class="col-md-8"><label class="admin-form-label">Author Profile</label>`)
	b.WriteString(`<select class="admin-form-control" name="`)
	b.WriteString(html.EscapeString(settingAuthorProfileID))
	b.WriteString(`"><option value="">— None —</option>`)
	for _, profile := range profiles {
		selected := ""
		if profile.ProfileID == selectedID {
			selected = " selected"
		}
		b.WriteString(fmt.Sprintf(`<option value="%d"%s>%s</option>`, profile.ProfileID, selected, html.EscapeString(profile.Name)))
	}
	b.WriteString(`</select>`)
	if len(profiles) == 0 {
		b.WriteString(`<p class="admin-form-help mb-0">No profiles defined yet. Add one under <strong>Profiles</strong> first.</p>`)
	} else {
		b.WriteString(`<p class="admin-form-help mb-0">Field values are edited on each user account in the admin.</p>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func formUintListAny(r *http.Request, key string) []any {
	values := r.Form[key]
	if len(values) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(values))
	for _, raw := range values {
		id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		out = append(out, id)
	}
	return out
}

func containsUint(ids []uint, id uint) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func groupDisplayName(name string) string {
	switch name {
	case "public":
		return "Public"
	case "registered":
		return "Registered"
	default:
		return name
	}
}
