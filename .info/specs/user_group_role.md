# Authorization System Design Specification

## Overview

The CMS authorization system is based on **Role-Based Access Control (RBAC)** using **Groups**, **Roles**, and **Namespaced Permissions**.

The design intentionally separates **organizational structure** from **authorization**.

- **Groups** describe who a user belongs to.
- **Roles** describe what a user can do.
- **Permissions** describe individual actions.
- **Plugins** register permissions automatically.
- **Roles** support inheritance.
- Authorization is always performed against permissions, never role names.

---

# Design Goals

The authorization system should be:

- Simple
- Fast
- Plugin-friendly
- Extensible
- Enterprise capable
- Easy to administer
- Easy to understand

---

# Core Model

```
User
    │
    ├── Groups
    │       │
    │       └── Roles
    │               │
    ├── Direct Roles│
    │               │
    └───────────────┘
            │
            ▼
     Role Inheritance
            │
            ▼
      Permissions
```

---

# Users

A user represents an authenticated account.

Users may:

- Belong to zero or more Groups.
- Receive zero or more Roles directly.
- Accumulate permissions from every assigned role.

Example:

```
Rob

Groups:
    Employees
    Marketing

Direct Roles:
    Site Administrator
```

---

# Groups

Groups represent organizational membership.

Examples:

- Employees
- Managers
- Marketing
- Accounting
- Editors
- Florida Office

Groups **never contain permissions directly**.

Groups are assigned one or more Roles.

Example:

```
Managers

Roles:
    Employee
    Manager
```

This avoids the need for group inheritance while allowing managers to receive all employee permissions.

---

# Roles

Roles represent capabilities.

Examples:

- Viewer
- Editor
- Publisher
- Manager
- Administrator

Roles contain permissions.

Roles may inherit other roles.

Example:

```
Administrator
        │
        ▼
Manager
        │
        ▼
Editor
        │
        ▼
Viewer
```

Administrator automatically receives every permission granted to Manager, Editor, and Viewer.

---

# Role Inheritance

Inheritance is stored separately from roles.

A role may inherit zero or more roles.

The inheritance graph should be validated to prevent circular references.

Example:

```
Viewer

Editor
    inherits Viewer

Manager
    inherits Editor

Administrator
    inherits Manager
```

Permission resolution expands inherited roles recursively.

---

# Permissions

Permissions represent one specific capability.

Permission IDs are globally unique.

## Naming Convention

```
plugin.resource.action
```

Examples:

```
core.users.read
core.users.create
core.users.update
core.users.delete

core.roles.manage

blog.article.read
blog.article.publish

gallery.image.upload

calendar.event.delete
```

---

# Permission Metadata

Permissions should contain metadata for display within the administration UI.

Example:

```go
type Permission struct {
    ID          string
    DisplayName string
    Description string
    Category    string
    Dangerous   bool
}
```

Example:

```
ID:
blog.article.publish

Display Name:
Publish Articles

Description:
Allows publishing blog articles.

Category:
Blog

Dangerous:
false
```

---

# Wildcard Permissions

The authorization engine should support wildcard matching.

Examples:

```
blog.*

blog.article.*

*.read

*
```

Checking

```
blog.article.update
```

should evaluate:

```
blog.article.update
blog.article.*
blog.*
*
```

---

# Authorization

Application code must never check role names.

Correct:

```go
auth.Can(user, "blog.article.publish")
```

Incorrect:

```go
if user.IsAdmin() {
    ...
}
```

Business logic always checks permissions.

---

# Permission Registration

Permissions are registered by plugins during startup.

Example:

```go
security.RegisterPermission(
    Permission{
        ID: "blog.article.publish",
        DisplayName: "Publish Articles",
        Description: "Allows publishing blog articles.",
        Category: "Blog",
    },
)
```

During application startup the CMS should:

- Register new permissions
- Update metadata
- Mark removed permissions as deprecated
- Never automatically delete assigned permissions

This makes code the source of truth.

---

# Recommended Registration API

```go
security.RegisterPermission(...)

security.RegisterRole(...)

security.RegisterPolicy(...)
```

Plugins should not manipulate database records directly.

---

# Permission Resolution

Permissions are resolved as follows:

```
User

↓

Direct Roles

+

Roles from Groups

↓

Expand Role Inheritance

↓

Collect Permissions

↓

Remove Duplicates

↓

Cache Result
```

The resulting permission set should be cached in memory.

Permission lookups should be O(1).

Example:

```go
permissions["blog.article.publish"] == true
```

---

# Database Schema

## users

```
id
username
...
```

---

## groups

```
id
name
description
```

---

## group_members

```
group_id
user_id
```

---

## roles

```
id
name
description
system_role
```

---

## role_inheritance

```
parent_role_id
child_role_id
```

Meaning:

```
Child inherits Parent
```

Example:

```
Parent: Viewer

Child: Editor
```

Editor inherits Viewer.

---

## permissions

```
id
permission_id
display_name
description
category
dangerous
deprecated
```

---

## role_permissions

```
role_id
permission_id
```

---

## group_roles

```
group_id
role_id
```

---

## user_roles

```
user_id
role_id
```

Allows assigning temporary or individual roles.

---

# Caching

Permission calculation should occur only when necessary.

Rebuild cache when:

- User role changes
- User group changes
- Group role changes
- Role inheritance changes
- Role permissions change

Normal requests should never query permission tables.

---

# Explicit deny permissions

Roles can assign **deny** permissions in addition to allows. An explicit deny overrides any matching allow from any role (including wildcards).

- Use **Allow** on the role form to grant a permission.
- Use **Deny** to block a permission even when inherited from parent roles or wildcards.
- Example: a role with `core.admin.*` allow and `core.admin.users.write` deny prevents user management writes while keeping other admin access.

Resolution order:

1. Collect allows and denies from all roles (direct, group, inherited).
2. If any deny matches the requested permission → **deny**.
3. Else if any allow matches → **allow**.
4. Else → **deny** (default).

Wire requests to extensions include `denied_permissions` alongside `permissions` when denies are present.

---

# Future Extensions

The design should allow future support for:

- Resource ownership policies
- Multi-tenant authorization
- Time-limited role assignments
- Permission conditions
- Audit logging
- Approval workflows
- Attribute-based access control (ABAC)

These additions should not require redesigning the core authorization model.

---

# Visibility groups (separate from RBAC)

Organizational groups and roles control **what users can do**. Item and category **visibility groups** control **who can see** published content on the frontend.

- Assign roles to groups (and optionally to users directly) for capabilities such as `core.admin.items.write` or `core.content.frontend.edit`.
- Assign visibility groups on item/category **Access** tabs to restrict read access. Empty visibility means public.
- `Public` and `Registered` are frontend visibility groups, not capability roles.

Do not use configuration toggles or category overlays to duplicate role-based frontend permissions. Use `core.content.frontend.*` permissions on roles instead.

---

# Content frontend permissions

Frontend create, edit, publish, delete, and comment actions use the `core.content.frontend.*` namespace:

- `core.content.frontend.view`
- `core.content.frontend.create`
- `core.content.frontend.edit`
- `core.content.frontend.edit.own`
- `core.content.frontend.publish`
- `core.content.frontend.delete`
- `core.content.frontend.comment.view`
- `core.content.frontend.comment.create`
- `core.content.frontend.comment.moderate`

Admin panel sections continue to use `core.admin.{section}.read` / `.write`.

---

# Notifications and roles

Notification delivery is specified in `.info/specs/notifications.md`. Authorization and notifications are separate concerns:

- **Groups** control visibility and membership; they do not route notifications.
- **Roles** grant capabilities **and** (Layer 2) may define **default hook event subscriptions** for all users assigned that role (e.g. Editors notified when content enters `pending` review).
- **Users** may override role defaults with per-account subscriptions (opt in or opt out per event).

Role-based subscriptions are configured on the role admin form; user subscriptions on the account/profile UI. Dispatch expands role subscriptions to member emails at send time; explicit user preferences take precedence over role defaults for the same event.

---

# Guiding Principles

1. Groups represent organizational membership.
2. Roles represent capabilities.
3. Permissions represent individual actions.
4. Roles inherit other roles.
5. Plugins own their permission namespace.
6. Business logic checks permissions, never role names.
7. Permissions are registered by code.
8. Effective permissions are cached.
9. Authorization defaults to deny.
10. The system should remain plugin-friendly and scalable.