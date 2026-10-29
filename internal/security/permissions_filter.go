package security

import (
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// PermissionExtensionOwner returns the extension name when id uses extension namespace (name.id).
func PermissionExtensionOwner(permissionID string) (string, bool) {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" || permissionID == PermWildcardAll {
		return "", false
	}
	if strings.HasPrefix(permissionID, "core.") {
		return "", false
	}
	idx := strings.IndexByte(permissionID, '.')
	if idx <= 0 {
		return "", false
	}
	return permissionID[:idx], true
}

// InstalledExtensionNames returns extension names marked installed in the site database.
func InstalledExtensionNames(db *gorm.DB) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if db == nil {
		return out, nil
	}
	var names []string
	if err := db.Model(&models.Extension{}).Where("installed = ?", true).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out, nil
}

// FilterPermissionsByInstalledExtensions keeps core permissions and extension permissions
// only for extensions currently marked installed on the site.
func FilterPermissionsByInstalledExtensions(db *gorm.DB, perms []Permission) ([]Permission, error) {
	installed, err := InstalledExtensionNames(db)
	if err != nil {
		return nil, err
	}
	out := make([]Permission, 0, len(perms))
	for _, p := range perms {
		ext, ok := PermissionExtensionOwner(p.ID)
		if !ok {
			out = append(out, p)
			continue
		}
		if _, ok := installed[ext]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}
