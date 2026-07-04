# Authorization

Cannon uses role-based access control (RBAC) to decide who can access the admin panel, manage content, and configure the site.

## Concepts

- **Permissions** — atomic capabilities such as `core.admin.items.read` or `core.admin.users.write`. Core permissions are registered at startup; extensions can register their own.
- **Roles** — named bundles of permissions. Roles can inherit from other roles so child roles receive parent permissions automatically.
- **Groups** — organizational membership. Groups are assigned roles; members receive those capabilities through their group memberships.
- **Users** — accounts can belong to groups and may also have **direct roles** assigned on the account.

Effective permissions for a user combine:

1. Direct roles on the account
2. Roles assigned to each group the user belongs to
3. Inherited permissions from parent roles in the role hierarchy

## Default roles

Built-in system roles form a ladder from read-only access to full control:

| Role | Purpose |
|------|---------|
| `viewer` | Admin panel access only |
| `writer` | Create and edit own content |
| `editor` | Broader content editing |
| `publisher` | Publish and feature content |
| `manager` | Manage most admin areas |
| `administrator` | Full access (`*` permission) |

System roles cannot be renamed, deactivated, or deleted.

## Admin screens

Under **Users** in the sidebar:

- **Accounts** — user profiles, group membership, and direct roles
- **Groups** — organizational groups with assigned roles
- **Roles** — permission sets and inheritance
- **Permissions** — read-only browser of registered capabilities

Category forms include an **Access** tab for visibility groups (who can view categories and items). Frontend create, edit, publish, and comment actions are controlled by **Content Frontend** permissions on roles — not by configuration group toggles.

### Content Frontend permissions

| Permission | Purpose |
|------------|---------|
| `core.content.frontend.view` | View protected frontend content features |
| `core.content.frontend.create` | Create items on the frontend |
| `core.content.frontend.edit` | Edit any item on the frontend |
| `core.content.frontend.edit.own` | Edit own items on the frontend |
| `core.content.frontend.publish` | Publish or feature items |
| `core.content.frontend.delete` | Permanently delete items |
| `core.content.frontend.comment.view` | View comment sections when signed in |
| `core.content.frontend.comment.create` | Post comments |
| `core.content.frontend.comment.moderate` | Moderate comments |

Assign these through **Groups → Roles** and **Users → direct roles**. The default `writer`, `editor`, and `publisher` roles include appropriate frontend permissions.

## Wildcards

Permission checks support wildcards:

- `*` — all permissions
- `core.*` — all core permissions
- `core.admin.*` — all admin-area permissions

When assigning permissions to a custom role, choose explicit grants rather than wildcards unless you intend broad access.

## Explicit deny

Roles can deny permissions as well as allow them. A deny on any role overrides matching allows (including wildcards such as `*` or `core.admin.*`).

Use deny to carve exceptions out of broad roles — for example, allow `core.admin.*` but deny `core.admin.users.write` for a support role that should not manage accounts.

## Extensions

Extensions register permissions through the extension host API. Those permissions appear in the Permissions browser and can be assigned to roles like core permissions.

- `core.admin.extension-apps.read` grants access to every extension admin UI.
- When an extension registers its own permissions, users need a matching extension permission (for example `my-extension.manage`) unless they have `core.admin.extension-apps.read` or `*`.

Extension admin links in the sidebar are shown only for extensions the current user can access.

See **Extensions → Authoring** for how to register permissions from an extension.
