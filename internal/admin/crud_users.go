package admin

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/controllers/auth"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"golang.org/x/crypto/bcrypt"
)

const usersBase = "/admin/users"

func (h *Handler) users(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/users", path)
	switch {
	case len(parts) == 0:
		h.userList(w, r)
	case parts[0] == "new":
		h.userForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.userDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.userToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.userForm(w, r, id)
	}
}

func (h *Handler) userList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.User
	var total int64
	db.Model(&models.User{}).Count(&total)
	data := listPage(r, page, total, usersBase,
		"Manage accounts, group membership, and role assignments.",
		"Add Account", map[string]any{"ActiveNav": "accounts"})
	order := applyListSort(r, data, map[string]string{
		"username": "username",
		"email":    "email",
		"status":   "status",
	}, "username")
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Preload("Groups").Preload("Roles").Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Accounts", "admin/users.html", data)
}

func (h *Handler) userForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.User
	if !isNew {
		if err := db.Preload("Groups").Preload("Roles").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	}
	var allGroups []models.Group
	allGroups = loadMembershipGroups(db)
	var allRoles []models.Role
	db.Where("status = ?", models.StatusActive).Order("name asc").Find(&allRoles)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if isNew {
			u, err := user.CreateLocalUser(r.Context(),
				formString(r, "given_name"), formString(r, "family_name"),
				formString(r, "email"), formString(r, "username"), r.FormValue("password"))
			if err != nil {
				h.renderUserForm(w, r, row, allGroups, allRoles, isNew, err.Error())
				return
			}
			row = *u
		} else {
			var prior models.User
			wasLocked := false
			if err := db.First(&prior, id).Error; err == nil {
				wasLocked = prior.Locked
			}
			row.GivenName = formString(r, "given_name")
			row.FamilyName = formString(r, "family_name")
			row.Email = formString(r, "email")
			row.Username = formString(r, "username")
			row.Status = formStatus(r)
			row.Locked = r.FormValue("locked") == "1"
			row.Validated = r.FormValue("validated") == "1"
			row.AvatarURL = formString(r, "avatar_url")
			if pw := r.FormValue("password"); pw != "" {
				hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				row.Hash = string(hash)
			}
			if err := db.Save(&row).Error; err != nil {
				h.renderUserForm(w, r, row, allGroups, allRoles, isNew, err.Error())
				return
			}
			if row.Locked && !wasLocked {
				_, _ = hooks.Fire(r.Context(), hooks.OnUserLocked, map[string]any{
					"user_id":  row.UserID,
					"username": row.Username,
					"email":    row.Email,
				})
			}
		}
		groupIDs := formUintList(r, "group_ids")
		var selected []models.Group
		if len(groupIDs) > 0 {
			db.Where("group_id IN ?", groupIDs).Find(&selected)
		}
		if err := db.Model(&row).Association("Groups").Replace(selected); err != nil {
			h.renderUserForm(w, r, row, allGroups, allRoles, isNew, err.Error())
			return
		}
		roleIDs := formUintList(r, "role_ids")
		var selectedRoles []models.Role
		if len(roleIDs) > 0 {
			db.Where("role_id IN ?", roleIDs).Find(&selectedRoles)
		}
		if err := db.Model(&row).Association("Roles").Replace(selectedRoles); err != nil {
			h.renderUserForm(w, r, row, allGroups, allRoles, isNew, err.Error())
			return
		}
		security.InvalidateSiteUser(r.Context(), row.UserID)
		if !isNew {
			if profileID, err := content.AuthorProfileID(r.Context()); err == nil && profileID > 0 {
				fields, _ := content.ActiveProfileFields(r.Context(), profileID)
				if err := content.SaveProfileUserFieldValues(db, row.UserID, fields, r); err != nil {
					h.renderUserForm(w, r, row, allGroups, allRoles, isNew, err.Error())
					return
				}
			}
		}
		if !row.Validated {
			_, _ = auth.EnsureVerifyToken(r.Context(), row.UserID)
		}
		redirectList(w, r, usersBase)
		return
	}
	h.renderUserForm(w, r, row, allGroups, allRoles, isNew, "")
}

func (h *Handler) renderUserForm(w http.ResponseWriter, r *http.Request, row models.User, allGroups []models.Group, allRoles []models.Role, isNew bool, errMsg string) {
	selected := make([]uint, 0, len(row.Groups))
	for _, group := range row.Groups {
		selected = append(selected, group.GroupID)
	}
	selectedRoles := make([]uint, 0, len(row.Roles))
	for _, role := range row.Roles {
		selectedRoles = append(selectedRoles, role.RoleID)
	}
	profileFields := []models.ProfileField{}
	profileInputs := []models.ContentField{}
	profileValues := map[uint]string{}
	authorProfileName := ""
	if profileID, err := content.AuthorProfileID(r.Context()); err == nil && profileID > 0 {
		profileFields, _ = content.ActiveProfileFields(r.Context(), profileID)
		for _, field := range profileFields {
			profileInputs = append(profileInputs, content.ProfileFieldAsContentField(field))
		}
		if !isNew && row.UserID > 0 {
			db, _ := sites.DB(r.Context())
			profileValues, _ = content.ProfileUserFieldValues(db, row.UserID, profileFields)
		}
		var profile models.Profile
		if db, err := sites.DB(r.Context()); err == nil {
			if db.First(&profile, profileID).Error == nil {
				authorProfileName = profile.Name
			}
		}
	}
	title := "Add Account"
	subtitle := "Create a new account with local authentication."
	if !isNew {
		title = "Edit Account"
		subtitle = "Update account details, group membership, and role assignments."
	}
	data := formData(map[string]any{
		"ActiveNav":   "accounts",
		"Row":         row,
		"IsNew":       isNew,
		"BasePath":    usersBase,
		"Subtitle":    subtitle,
		"AllGroups":         allGroups,
		"AllRoles":          allRoles,
		"SelectedIDs":       selected,
		"SelectedRoleIDs":   selectedRoles,
		"AuthorProfileName": authorProfileName,
		"ProfileFields":     profileInputs,
		"ProfileValues":     profileValues,
	})
	if !isNew {
		if avatar, err := content.ResolveUserAvatar(r.Context(), row.UserID); err == nil {
			data["ResolvedAvatarURL"] = avatar
		}
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if !isNew && !row.Validated {
		if token, err := auth.EnsureVerifyToken(r.Context(), row.UserID); err == nil {
			data["VerifyURL"] = userVerifyURL(r, token)
		}
	}
	if !isNew && row.UserID > 0 {
		if preview, err := effectivePermissionPreview(r.Context(), row.UserID); err == nil {
			data["EffectivePermissions"] = preview
		}
	}
	h.render(w, r, title, "admin/users_form.html", data)
}

func userVerifyURL(r *http.Request, token string) string {
	path := auth.VerifyURL(r.Context(), token)
	site, err := sites.FromContext(r.Context())
	if err != nil || strings.TrimSpace(site.Host) == "" {
		return path
	}
	return absoluteSiteURL(site.Host, path)
}

func (h *Handler) userDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	if err := db.Exec("DELETE FROM user_groups WHERE user_user_id = ?", id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&models.User{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, usersBase)
}

func rowFromForm(r *http.Request) models.User {
	return models.User{
		GivenName:  formString(r, "given_name"),
		FamilyName: formString(r, "family_name"),
		Email:      formString(r, "email"),
		Username:   formString(r, "username"),
		Status:     formStatus(r),
		Locked:     r.FormValue("locked") == "1",
		Validated:  r.FormValue("validated") == "1",
	}
}
