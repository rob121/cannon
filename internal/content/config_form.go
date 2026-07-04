package content

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// MergeConfigurationAuthorProfileFormData stores the author profile id from the admin form.
func MergeConfigurationAuthorProfileFormData(data map[string]any, r *http.Request) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	if v, err := strconv.ParseUint(strings.TrimSpace(r.FormValue(settingAuthorProfileID)), 10, 64); err == nil && v > 0 {
		data[settingAuthorProfileID] = v
	} else {
		data[settingAuthorProfileID] = nil
	}
	return data
}

// InjectConfigurationAuthorProfileField appends the author profile picker to a JSON Forms configuration form.
func InjectConfigurationAuthorProfileField(formHTML string, profiles []models.Profile, authorProfileID uint) string {
	block := `<div class="row g-3 admin-jsonforms-extra">` +
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
		b.WriteString(`<p class="admin-form-help mb-0">Field values are edited on each user account in the admin. Frontend content permissions are managed under <strong>Users → Roles</strong>.</p>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}
