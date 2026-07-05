package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func flipStatus(status models.Status) models.Status {
	if status == models.StatusActive {
		return models.StatusInactive
	}
	return models.StatusActive
}

func listRedirectQuery(r *http.Request) string {
	page := parsePage(r)
	sort := r.URL.Query().Get("sort")
	dir := r.URL.Query().Get("dir")
	extra := url.Values{}
	for key, vals := range r.URL.Query() {
		if key == "page" || key == "sort" || key == "dir" {
			continue
		}
		extra[key] = vals
	}
	return listQueryExtra(page, sort, dir, extra)
}

func toggleModelStatus(db *gorm.DB, id uint, dest any) error {
	if err := db.First(dest, id).Error; err != nil {
		return err
	}
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("toggle status: dest must be pointer to struct")
	}
	field := val.Elem().FieldByName("Status")
	if !field.IsValid() || !field.CanSet() {
		return fmt.Errorf("toggle status: missing Status field")
	}
	current, ok := field.Interface().(models.Status)
	if !ok {
		return fmt.Errorf("toggle status: Status field has unexpected type")
	}
	field.Set(reflect.ValueOf(flipStatus(current)))
	return db.Save(dest).Error
}

func (h *Handler) postToggleModel(w http.ResponseWriter, r *http.Request, idStr string, dest any, redirectBase string, after func(context.Context)) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, err := sites.DB(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := toggleModelStatus(db, id, dest); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.notFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if after != nil {
		after(r.Context())
	}
	redirectList(w, r, redirectBase+listRedirectQuery(r))
}

func (h *Handler) blockToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Block{}, blocksBase, invalidateBlocksDataCache)
}

func (h *Handler) routeToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Route{}, routesBase, invalidateRoutesDataCache)
}

func (h *Handler) userToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.User{}, usersBase, func(ctx context.Context) {
		invalidateSecurityUserFromRequest(ctx, idStr)
	})
}

func (h *Handler) groupToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Group{}, groupsBase, invalidateGroupsDataCache)
}

func (h *Handler) roleToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, err := sites.DB(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var row models.Role
	if err := db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.notFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row.SystemRole {
		http.Error(w, "system roles cannot be deactivated", http.StatusBadRequest)
		return
	}
	h.postToggleModel(w, r, idStr, &models.Role{}, rolesBase, invalidateSecuritySiteContext)
}

func (h *Handler) menuToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Menu{}, menusBase, nil)
}

func (h *Handler) authenticatorToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Authenticator{}, authenticatorsBase, nil)
}

func (h *Handler) profileFieldToggleStatus(w http.ResponseWriter, r *http.Request, profileIDStr, fieldIDStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profileID, ok := parseID(profileIDStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	fieldID, ok := parseID(fieldIDStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, err := sites.DB(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var field models.ProfileField
	if err := db.Where("profile_id = ?", profileID).First(&field, fieldID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.notFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	field.Status = flipStatus(field.Status)
	if err := db.Save(&field).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, profilesBase+"/"+profileIDStr)
}
