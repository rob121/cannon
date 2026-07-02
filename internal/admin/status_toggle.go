package admin

import (
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

func (h *Handler) postToggleModel(w http.ResponseWriter, r *http.Request, idStr string, dest any, redirectBase string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, err := sites.DB(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := toggleModelStatus(db, id, dest); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, redirectBase+listRedirectQuery(r))
}

func (h *Handler) blockToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Block{}, blocksBase)
}

func (h *Handler) routeToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Route{}, routesBase)
}

func (h *Handler) userToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.User{}, usersBase)
}

func (h *Handler) groupToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Group{}, groupsBase)
}

func (h *Handler) roleToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Role{}, rolesBase)
}

func (h *Handler) menuToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Menu{}, menusBase)
}

func (h *Handler) authenticatorToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Authenticator{}, authenticatorsBase)
}

func (h *Handler) profileFieldToggleStatus(w http.ResponseWriter, r *http.Request, profileIDStr, fieldIDStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profileID, ok := parseID(profileIDStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	fieldID, ok := parseID(fieldIDStr)
	if !ok {
		http.NotFound(w, r)
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
			http.NotFound(w, r)
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
