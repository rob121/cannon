package security

// Core permission id constants.
const (
	PermAdminAccess            = "core.admin.access"
	PermAdminExtensionAppsRead = "core.admin.extension-apps.read"

	PermContentFrontendView            = "core.content.frontend.view"
	PermContentFrontendCreate          = "core.content.frontend.create"
	PermContentFrontendEdit            = "core.content.frontend.edit"
	PermContentFrontendEditOwn         = "core.content.frontend.edit.own"
	PermContentFrontendPublish         = "core.content.frontend.publish"
	PermContentFrontendDelete          = "core.content.frontend.delete"
	PermContentFrontendCommentView     = "core.content.frontend.comment.view"
	PermContentFrontendCommentCreate   = "core.content.frontend.comment.create"
	PermContentFrontendCommentModerate = "core.content.frontend.comment.moderate"

	PermUsersRead   = "core.users.read"
	PermUsersCreate = "core.users.create"
	PermUsersUpdate = "core.users.update"
	PermUsersDelete = "core.users.delete"

	PermRolesManage       = "core.roles.manage"
	PermGroupsManage      = "core.groups.manage"
	PermPermissionsRead   = "core.permissions.read"
	PermWildcardAll       = "*"
	PermWildcardCore      = "core.*"
	PermWildcardCoreAdmin = "core.admin.*"
)

// Legacy content permission ids migrated to core.content.frontend.*.
var legacyContentPermissionMap = map[string]string{
	"core.content.create":          PermContentFrontendCreate,
	"core.content.edit":            PermContentFrontendEdit,
	"core.content.edit.own":        PermContentFrontendEditOwn,
	"core.content.publish":         PermContentFrontendPublish,
	"core.content.delete":          PermContentFrontendDelete,
	"core.content.comments.manage": PermContentFrontendCommentModerate,
}

// System role name constants.
const (
	RoleAdministrator = "administrator"
	RoleManager       = "manager"
	RoleEditor        = "editor"
	RolePublisher     = "publisher"
	RoleWriter        = "writer"
	RoleViewer        = "viewer"

	// Legacy names kept for migration mapping.
	RoleLegacyAdmin  = "admin"
	RoleLegacyAuthor = "author"
)

func contentFrontendPerms() []string {
	return []string{
		PermContentFrontendView,
		PermContentFrontendCreate,
		PermContentFrontendEdit,
		PermContentFrontendEditOwn,
		PermContentFrontendPublish,
		PermContentFrontendDelete,
		PermContentFrontendCommentView,
		PermContentFrontendCommentCreate,
		PermContentFrontendCommentModerate,
	}
}

func registerCorePermissions() {
	adminSections := []struct {
		key   string
		label string
	}{
		{"items", "Items"},
		{"categories", "Categories"},
		{"tags", "Tags"},
		{"field-groups", "Field Groups"},
		{"comments", "Comments"},
		{"media", "Media"},
		{"blocks", "Blocks"},
		{"routes", "Routes"},
		{"templates", "Templates"},
		{"menus", "Menus"},
		{"menu-items", "Menu Items"},
		{"users", "Users"},
		{"groups", "Groups"},
		{"roles", "Roles"},
		{"notifications", "Notifications"},
		{"configuration", "Configuration"},
		{"extensions", "Extensions"},
		{"extension-apps", "Extension Apps"},
		{"help", "Help"},
		{"languages", "Languages"},
		{"sites", "Sites"},
		{"system", "System"},
		{"authenticators", "Authenticators"},
		{"profiles", "Profiles"},
		{"api", "API"},
	}
	for _, section := range adminSections {
		base := "core.admin." + section.key
		RegisterPermission(Permission{
			ID:          base + ".read",
			DisplayName: "View " + section.label,
			Description: "Allows read access to the " + section.label + " admin section.",
			Category:    "Admin",
		})
		RegisterPermission(Permission{
			ID:          base + ".write",
			DisplayName: "Manage " + section.label,
			Description: "Allows write access to the " + section.label + " admin section.",
			Category:    "Admin",
			Dangerous:   section.key == "users" || section.key == "roles" || section.key == "system" || section.key == "sites",
		})
	}

	RegisterPermission(Permission{
		ID:          PermAdminAccess,
		DisplayName: "Access Admin",
		Description: "Allows signing in to and viewing the admin area.",
		Category:    "Admin",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendView,
		DisplayName: "View Frontend Content",
		Description: "Allows viewing protected frontend content and comment sections.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendCreate,
		DisplayName: "Create Content",
		Description: "Allows creating content items on the frontend.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendEdit,
		DisplayName: "Edit Any Content",
		Description: "Allows editing any content item on the frontend.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendEditOwn,
		DisplayName: "Edit Own Content",
		Description: "Allows editing content items authored by the user on the frontend.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendPublish,
		DisplayName: "Publish Content",
		Description: "Allows publishing and unpublishing content items on the frontend.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendDelete,
		DisplayName: "Delete Content",
		Description: "Allows permanently deleting content items.",
		Category:    "Content Frontend",
		Dangerous:   true,
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendCommentView,
		DisplayName: "View Comments",
		Description: "Allows viewing item comment sections on the frontend.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendCommentCreate,
		DisplayName: "Post Comments",
		Description: "Allows posting comments on content items.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermContentFrontendCommentModerate,
		DisplayName: "Moderate Comments",
		Description: "Allows moderating comments.",
		Category:    "Content Frontend",
	})
	RegisterPermission(Permission{
		ID:          PermUsersRead,
		DisplayName: "View Users",
		Description: "Allows viewing user accounts.",
		Category:    "Users",
	})
	RegisterPermission(Permission{
		ID:          PermUsersCreate,
		DisplayName: "Create Users",
		Description: "Allows creating user accounts.",
		Category:    "Users",
	})
	RegisterPermission(Permission{
		ID:          PermUsersUpdate,
		DisplayName: "Update Users",
		Description: "Allows updating user accounts.",
		Category:    "Users",
	})
	RegisterPermission(Permission{
		ID:          PermUsersDelete,
		DisplayName: "Delete Users",
		Description: "Allows deleting user accounts.",
		Category:    "Users",
		Dangerous:   true,
	})
	RegisterPermission(Permission{
		ID:          PermRolesManage,
		DisplayName: "Manage Roles",
		Description: "Allows creating and editing roles and permissions.",
		Category:    "Security",
		Dangerous:   true,
	})
	RegisterPermission(Permission{
		ID:          PermGroupsManage,
		DisplayName: "Manage Groups",
		Description: "Allows creating and editing user groups.",
		Category:    "Security",
	})
	RegisterPermission(Permission{
		ID:          PermPermissionsRead,
		DisplayName: "View Permissions",
		Description: "Allows browsing the permission catalog.",
		Category:    "Security",
	})
}

func registerCoreRoles() {
	contentPerms := contentFrontendPerms()
	adminReadPerms := adminSectionPerms(false)
	adminWritePerms := adminSectionPerms(true)

	RegisterRole(RoleDef{
		Name:        RoleViewer,
		Description: "Basic admin sign-in access.",
		SystemRole:  true,
		Permissions: []string{PermAdminAccess},
	})
	RegisterRole(RoleDef{
		Name:        RoleWriter,
		Description: "Create and edit own content.",
		SystemRole:  true,
		Inherits:    []string{RoleViewer},
		Permissions: []string{
			PermContentFrontendView,
			PermContentFrontendCreate,
			PermContentFrontendEditOwn,
			PermContentFrontendCommentView,
			PermContentFrontendCommentCreate,
		},
	})
	RegisterRole(RoleDef{
		Name:        RoleEditor,
		Description: "Full content management and most admin sections.",
		SystemRole:  true,
		Inherits:    []string{RoleWriter},
		Permissions: append(contentPerms, adminReadPerms...),
	})
	RegisterRole(RoleDef{
		Name:        RolePublisher,
		Description: "Editor capabilities focused on publishing.",
		SystemRole:  true,
		Inherits:    []string{RoleEditor},
		Permissions: []string{PermContentFrontendPublish},
	})
	RegisterRole(RoleDef{
		Name:        RoleManager,
		Description: "Manage content and admin operations.",
		SystemRole:  true,
		Inherits:    []string{RolePublisher},
		Permissions: append(adminWritePerms, PermGroupsManage),
	})
	RegisterRole(RoleDef{
		Name:        RoleAdministrator,
		Description: "Full system access.",
		SystemRole:  true,
		Inherits:    []string{RoleManager},
		Permissions: []string{PermWildcardAll, PermRolesManage, PermPermissionsRead},
	})
}

func adminSectionPerms(write bool) []string {
	sections := []string{
		"items", "categories", "tags", "field-groups", "comments", "media", "blocks",
		"routes", "templates", "menus", "menu-items", "users", "groups", "roles",
		"notifications", "configuration", "extensions", "extension-apps", "help", "languages", "sites",
		"system", "authenticators", "profiles", "api",
	}
	var out []string
	for _, s := range sections {
		suffix := ".read"
		if write {
			suffix = ".write"
		}
		out = append(out, "core.admin."+s+suffix)
	}
	return out
}

// AdminPermissionForPath returns the permission id for an admin section path.
func AdminPermissionForPath(section string, write bool) string {
	if section == "" || section == "/" {
		return PermAdminAccess
	}
	section = trimAdminSection(section)
	suffix := ".read"
	if write {
		suffix = ".write"
	}
	return "core.admin." + section + suffix
}

func trimAdminSection(section string) string {
	if len(section) > 0 && section[0] == '/' {
		section = section[1:]
	}
	return section
}

func init() {
	registerCorePermissions()
	registerCoreRoles()
	RegisterPolicy(PolicyDef{
		ID:          PermContentFrontendEditOwn,
		Description: "Allows editing only items authored by the current user.",
	})
}
