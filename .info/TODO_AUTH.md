# Authorization System — Implementation Checklist

Target spec: `.info/specs/user_group_role.md`

Status key: `[ ]` pending · `[~]` partial · `[x]` done

---

## Phase 0 — Design decisions (blockers)

- [x] **GroupAdminRoute fate** — Remove/replace with permissions
- [x] **Category permission groups** — Removed; frontend actions use `core.content.frontend.*` on roles; visibility groups stay on Access tabs
- [x] **Editorial group hierarchy** — Flatten
- [x] **Author/writer ownership** — `core.content.frontend.edit.own` permission policy
- [x] **`HasBackendAccess`** — `core.admin.access`
- [x] **Visibility groups** — Stay separate from RBAC
- [x] **Package name** — `internal/security`

---

## Phase 1 — Core engine & schema

### 1.1 New package: `internal/security`

- [x] `Permission` struct (ID, DisplayName, Description, Category, Dangerous, Deprecated)
- [x] In-memory permission registry
- [x] `RegisterPermission(...)` — code is source of truth
- [x] `RegisterRole(...)` — default/system roles with metadata
- [x] `RegisterPolicy(...)` — stub for future ABAC
- [x] `Can(ctx, userID, permissionID string) (bool, error)` — default deny
- [x] Wildcard matching (`blog.article.update` → `blog.article.*`, `blog.*`, `*`)
- [x] Permission resolver: direct roles + group roles → expand inheritance → dedupe
- [x] In-memory cache per user (O(1) map lookup)
- [x] Cache invalidation triggers
- [x] Circular inheritance validation on write
- [x] Unit tests: resolution, inheritance, wildcards, cache invalidation, deny-by-default

### 1.2 Database models & migration

- [x] `permissions` table
- [x] `role_permissions` join table
- [x] `role_inheritance` table
- [x] `user_roles` join table (direct role assignment)
- [x] Extend `roles`: `description`, `system_role`
- [x] Extend `groups`: `description`
- [x] Add models to `models.All()` for AutoMigrate
- [x] Startup sync: register code permissions → upsert DB, mark removed as deprecated

---

## Phase 2 — Core permission registration

### 2.1 Admin section permissions

- [x] All `core.admin.{section}.read` / `.write` permissions mapped from `AdminRoutes`
- [x] `core.admin.access`

### 2.2 Content permissions (frontend)

- [x] `core.content.frontend.view`
- [x] `core.content.frontend.create`
- [x] `core.content.frontend.edit`
- [x] `core.content.frontend.edit.own`
- [x] `core.content.frontend.publish`
- [x] `core.content.frontend.delete`
- [x] `core.content.frontend.comment.view`
- [x] `core.content.frontend.comment.create`
- [x] `core.content.frontend.comment.moderate`
- [x] Legacy `core.content.*` keys migrated to `core.content.frontend.*` on startup

### 2.3 User & security management

- [x] `core.users.read` / `.create` / `.update` / `.delete`
- [x] `core.roles.manage`
- [x] `core.groups.manage`
- [x] `core.permissions.read`

### 2.4 Default roles & inheritance seed

- [x] Seed roles: viewer, writer, editor, publisher, manager, administrator
- [x] Seed inheritance chain (administrator → manager → publisher → editor → writer → viewer)
- [x] Assign default permission sets per role matching prior behavior
- [x] Mark system roles as `system_role = true`
- [x] Update `roles.EnsureDefaults` — permissions/inheritance, flattened groups

---

## Phase 3 — Replace authorization call sites

### 3.1 Admin access

- [x] Replace `CanAccessAdmin` with `security.Can` per section + method
- [x] Remove `admin` role name bypass
- [x] Admin handler uses permission checks via `requireAccess`
- [x] Remove `GroupAdminRoute` model and `group_admin_routes` table (migrated on startup)
- [x] Remove admin route matrix UI

### 3.2 Content permissions

- [x] Refactor `internal/content/permissions.go`
- [x] Replace `hasAnyRole` / `roles.HasRole` with `security.Can`
- [x] Encode author/writer ownership as `core.content.frontend.edit.own` policy
- [x] Removed configuration/category permission group overlays (RBAC only for actions)
- [x] Update content controllers (unchanged call sites; logic in permissions package)
- [x] Update `internal/content/permissions_test.go`

### 3.3 Roles package

- [x] Remove `roles.HasRole` from authorization paths
- [x] `roles.AssignAdmin` via group assignment + cache invalidation
- [x] `groups.HasBackendAccess` → `core.admin.access`

### 3.4 Router / offline bypass

- [x] `canBypassSiteOffline` uses `HasBackendAccess` → `core.admin.access`

---

## Phase 4 — Request-scoped permission context

- [x] Resolve effective permissions once per authenticated request (Session middleware)
- [x] Attach to context via `security.WithPermissions`
- [x] Expose in `user.Service.Context()` for templates/extensions
- [x] `security.CanCurrent(ctx, permission)` helper
- [x] Normal requests use cache (no permission table queries on hit)

---

## Phase 5 — Admin UI

### 5.1 Roles

- [x] Role form: permission assignment (grouped by category)
- [x] Role form: inheritance picker
- [x] Protect system roles from delete
- [x] List view: permission count

### 5.2 Groups

- [x] Keep role multi-select on group form
- [x] Remove admin route read/write matrix
- [x] Clarify groups = organizational membership

### 5.3 Users

- [x] Add direct role assignment (`user_roles`)
- [x] Show effective permissions preview

### 5.4 Permissions browser

- [x] Read-only list of registered permissions
- [x] Filter by category
- [x] Show dangerous permissions styling

### 5.5 Templates

- [x] `admin/roles_form.html` — permissions + inheritance UI
- [x] `admin/users_form.html` — direct roles
- [x] `admin/groups_form.html` — route matrix removed
- [x] `admin/permissions.html`

---

## Phase 6 — Data migration (existing sites)

- [x] `EnsureDefaults` upgrade path via `security.EnsureForSite`
- [x] Map legacy `admin`/`author`/etc. role names to new system roles
- [x] Convert `group_admin_routes` rows → role permissions (then drop table)
- [x] Flatten editorial group hierarchy
- [x] Migrate legacy `core.content.*` role permission keys to `core.content.frontend.*`
- [ ] Verify on production site smoke test (manual)
- [x] Rollback: restore DB backup if migration unsatisfactory

---

## Phase 7 — Extensions / plugins

- [x] Extension capabilities response includes `permissions` array
- [x] `extension.Server.RegisterPermissions(...)`
- [x] Host syncs extension permissions to DB on bootstrap
- [x] Wire `User` scope includes `permissions` list
- [x] `extension.UserCan(req, permission)` helper (wildcard-aware)
- [x] Document in `EXTENSIONS.md` and `internal/help/docs/extensions/authoring.md`
- [x] Example: `extension/server_test.go` + `extension/permissions_test.go`

---

## Phase 8 — Cleanup & documentation

- [x] Remove dead code: `GroupAdminRoute`, `group_permissions.go`, configuration/category permission overlays
- [x] Grep cleanup: no `roles.HasRole` in authorization paths
- [x] Update `.info/TODO.md` — Users & groups section
- [x] Update spec: visibility groups vs RBAC in `user_group_role.md`
- [x] Admin help article: `internal/help/docs/admin/authorization.md`
- [x] Integration tests: admin access, content permissions, security unit tests

---

## Test plan (per slice)

- [x] Unit: resolver, inheritance, wildcards, cache
- [x] Unit: `CanAccessAdmin` equivalent for admin section access
- [x] Unit: content create/edit with role + policy combinations
- [x] Unit: direct user role assignment
- [x] Unit: deprecated permission still honored if assigned
- [x] Unit: legacy `core.content.*` key migration
- [ ] Integration: non-admin user with partial admin access (post-migration; manual smoke)
- [x] Integration: author/writer via `edit.own` in permission tests
- [x] Integration: administrator wildcard access
- [x] Performance: permission check uses in-memory cache after first resolve

---

## Known follow-ups (not blocking Phase 7/8)

- [x] Seed `core.admin.extension-apps.read` and per-extension admin access checks
- [x] Permission-filter extension nav items in admin sidebar
- [x] Users form: effective permissions preview

---

## Out of scope (spec future extensions — do not block v1)

- [x] Explicit deny permissions
- [ ] Resource ownership policies (beyond `edit.own`)
- [ ] Multi-tenant authorization
- [ ] Time-limited role assignments
- [ ] Permission conditions
- [ ] Audit logging
- [ ] Approval workflows
- [ ] Full ABAC

Design v1 so these can be added without redesigning the core model.
