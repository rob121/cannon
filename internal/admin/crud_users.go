package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/controllers/auth"
	"github.com/rob121/cannon/internal/models"
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
			http.NotFound(w, r)
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
	data := listPage(page, total, usersBase,
		"Manage user accounts and access.",
		"Add Account", map[string]any{"ActiveNav": "accounts"})
	order := applyListSort(r, data, map[string]string{
		"username": "username",
		"email":    "email",
		"status":   "status",
	}, "username")
	db.Offset((page - 1) * pageSize).Limit(pageSize).Preload("Groups").Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Accounts", "admin/users.html", data)
}

func (h *Handler) userForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.User
	if !isNew {
		if err := db.Preload("Groups").First(&row, id).Error; err != nil {
			http.NotFound(w, r)
			return
		}
	}
	var allGroups []models.Group
	db.Where("status = ?", models.StatusActive).Order("name asc").Find(&allGroups)

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
				h.renderUserForm(w, r, row, allGroups, isNew, err.Error())
				return
			}
			row = *u
		} else {
			row.GivenName = formString(r, "given_name")
			row.FamilyName = formString(r, "family_name")
			row.Email = formString(r, "email")
			row.Username = formString(r, "username")
			row.Status = formStatus(r)
			row.Locked = r.FormValue("locked") == "1"
			row.Validated = r.FormValue("validated") == "1"
			if pw := r.FormValue("password"); pw != "" {
				hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				row.Hash = string(hash)
			}
			if err := db.Save(&row).Error; err != nil {
				h.renderUserForm(w, r, row, allGroups, isNew, err.Error())
				return
			}
		}
		groupIDs := formUintList(r, "group_ids")
		var selected []models.Group
		if len(groupIDs) > 0 {
			db.Where("group_id IN ?", groupIDs).Find(&selected)
		}
		if err := db.Model(&row).Association("Groups").Replace(selected); err != nil {
			h.renderUserForm(w, r, row, allGroups, isNew, err.Error())
			return
		}
		if !row.Validated {
			_, _ = auth.EnsureVerifyToken(r.Context(), row.UserID)
		}
		redirectList(w, r, usersBase)
		return
	}
	h.renderUserForm(w, r, row, allGroups, isNew, "")
}

func (h *Handler) renderUserForm(w http.ResponseWriter, r *http.Request, row models.User, allGroups []models.Group, isNew bool, errMsg string) {
	selected := make([]uint, 0, len(row.Groups))
	for _, group := range row.Groups {
		selected = append(selected, group.GroupID)
	}
	title := "Add Account"
	subtitle := "Create a new account with local authentication."
	if !isNew {
		title = "Edit Account"
		subtitle = "Update account details, groups, and access status."
	}
	data := formData(map[string]any{
		"ActiveNav":   "accounts",
		"Row":         row,
		"IsNew":       isNew,
		"BasePath":    usersBase,
		"Subtitle":    subtitle,
		"AllGroups":   allGroups,
		"SelectedIDs": selected,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if !isNew && !row.Validated {
		if token, err := auth.EnsureVerifyToken(r.Context(), row.UserID); err == nil {
			data["VerifyURL"] = auth.VerifyURL(token)
		}
	}
	h.render(w, r, title, "admin/users_form.html", data)
}

func (h *Handler) userDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
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
