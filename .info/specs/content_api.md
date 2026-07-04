# Content API (Headless)

Cannon exposes a **versioned REST API** so external frontends (Next.js, mobile apps, static site generators) can use the CMS **without Cannon's HTML templates or controllers**.

The API has two layers:

1. **Published content (read-only)** — items, categories, tags, search, media, and approved comments, with the same group visibility as the rendered site.
2. **Account & engagement** — self-service account tasks (profile, password, MFA, notifications, password reset) require a user JWT; **posting comments** is available to guests or signed-in users (with captcha), matching the rendered site.

The API does **not** expose **CMS administration** (`/admin/*`): no content editing for editors/managers, no user/role/route management, no site configuration. Editorial administration stays in the Cannon admin UI.

This spec covers dual authentication (application credentials + user JWT), frontend parity, admin credential management, OpenAPI/Swagger documentation, and phased delivery. It complements `.info/specs/content.md` (content model) and `.info/specs/user_group_role.md` (groups vs roles).

Product Decisisons - answers are prepended with "A-"

Remaining open items (need a product decision)
GET /media/{id} access control — Can any API client fetch any media id, or only assets tied to visible content? (Open question #1, narrowed.)

A-Visible Items

Contributor editing — /content/edit still deferred; parity matrix flags it but it's the biggest functional gap vs the rendered site for writers.

A- keep defered

Passkey MFA at login — Spec lists passkey in methods but passkey login/security is phase 2. Phase 1a should clarify TOTP-only for POST /auth/mfa unless you want passkeys in 1a.

A-Expose a way to respond that it's necessary and endpoint to send to 

GET /tags without JWT — Tags have no group filter in code (ListTags returns all). That's probably fine, but item lists by tag aren't exposed — headless apps filter client-side or via search.

A-OK

OpenAPI path for docs — /api/docs is unversioned while everything else is /api/v1/…. Intentional, but worth noting in implementation.

A-Version the docs too

event_hooks.md reference — Spec cites context: "api" for login hooks; confirm that doc is updated when implementing (not a spec bug, just a cross-doc follow-up).

A-OK

---

## Goals

- **Headless viewing:** Published items, categories, tags, media, authors, search, and approved comments as JSON.
- **Frontend parity:** Anonymous visitors see **public** content; signed-in users see content visible to their **frontend groups** (same rules as `groups.ViewerGroupIDs` + `content.VisibleItemsQuery`).
- **Account parity:** Signed-in users can perform the same **self-service account tasks** as `/account/profile` on the rendered site — update identity, avatar, custom profile fields, password, notifications, and MFA — via JWT-authenticated API endpoints.
- **User sessions for headless apps:** `POST /api/v1/auth/login` validates credentials and returns a **JWT**; subsequent requests use that JWT for visibility and account operations.
- **Application credentials:** Each headless integration gets an API key (`cn_live_…`, header `X-Cannon-API-Key`) managed in admin, proving the client app is allowed to call the API.
- **Documented:** OpenAPI 3 + Swagger UI.
- **Versioned:** `/api/v1/…` with a clear deprecation policy.
- **Site-scoped:** Per-site credentials, users, and content.

---

## Non-goals

| Excluded | Notes |
|----------|-------|
| **CMS administration** | No `/admin/*` operations: item/category/tag CRUD, media library management, routes, blocks, users, roles, extensions, configuration, etc. |
| **Frontend contributor editing** | Rendered-site `/content/edit` (writer/editor item create/edit) is **not** in v1; revisit in a later phase if headless contributor workflows are needed. |
| **Draft / preview via API** | No unpublished content or preview tokens; editorial preview stays admin-only. |
| **GraphQL** | Defer; REST + OpenAPI first. |
| **Replacing admin UI** | API mirrors the **public frontend**, not the admin panel. |
| **Extension API routes** | Headless API is core Cannon only. |
| **Webhooks / real-time** | Defer. |
| **User registration via API** | Defer; login + account self-service first. |
| **OAuth token exchange** | Defer; local username/password login first. |

---

## Frontend parity principle

A headless client should support the same **end-user** experiences as the rendered frontend. If a visitor or signed-in member can do it on the public site (outside `/admin`), the API should eventually expose an equivalent JSON operation.

**In scope:** viewing published content, signing in, managing one's own account, posting comments, password reset, MFA enrollment.

**Out of scope:** anything that requires admin permissions (`core.admin.*`) or the admin UI.

### Parity matrix

Maps existing HTML controllers (`internal/controllers/auth`, `internal/controllers/content`) to API endpoints.

| Frontend capability | Rendered route / action | API endpoint | Phase |
|---------------------|-------------------------|--------------|-------|
| Login | `auth/login` POST | `POST /auth/login` | 1a |
| MFA challenge (login) | `auth/mfa-challenge` | `POST /auth/mfa` | 1a |
| Logout | `auth/logout` | `POST /auth/logout` | 1a |
| Current user + profile fields | `auth/profile` GET | `GET /auth/me` | 1a |
| Update profile identity | `auth/profile` POST | `PATCH /auth/me` | 1a |
| Custom author profile fields | `auth/profile` POST | `PATCH /auth/me` (`custom_fields`) | 1a |
| Upload avatar | `auth/profile` multipart | `POST /auth/avatar` | 1a |
| Remove avatar | `auth/profile` POST | `DELETE /auth/avatar` | 1a |
| Change password | `auth/profile` `form=password` | `POST /auth/password` | 1a |
| Notification subscriptions | `auth/profile` `form=notifications` | `GET` + `PUT /auth/notifications` | 1b |
| TOTP enroll / disable | `auth/security-totp/*` | `POST /auth/security/totp/*` | 1b |
| Passkey enroll / remove | `auth/security-passkey/*` | `POST /auth/security/passkeys/*` | 2 |
| Passkey login | `auth/passkey-login` | `POST /auth/passkey-login` | 2 |
| Password reset request | `auth/reset-request` | `POST /auth/password-reset/request` | 1b |
| Password reset confirm | `auth/reset-submit` | `POST /auth/password-reset/confirm` | 1b |
| Account verification | `auth/verify` | `POST /auth/verify` | 2 |
| Resend verification | `auth/verify-resend` | `POST /auth/verify/resend` | 2 |
| OAuth sign-in | `auth/oauth` | Defer | — |
| View items / categories / tags | `content/*` | `GET /items`, `/categories`, `/tags`, … | 1a |
| Search | `content/search` | `GET /search` | 1a |
| Read comments | item template | `GET /items/{id}/comments` | 1b |
| Post comment | item POST comment | `POST /items/{id}/comments` | 1b |
| Create / edit items (contributor) | `content/edit` | **Not in v1** (see non-goals) | — |

Implementation should **delegate** to existing packages (`user.UpdateProfileIdentity`, `user.UpdatePassword`, `user.SaveAvatarUpload`, `content.SaveProfileUserFieldValues`, `notifications.SaveUserSubscriptions`, `mfa.*`, `cms.CreateComment`, etc.) rather than duplicating business logic.

---

## Architecture overview

```
Headless frontend (Next.js, mobile, …)
        │
        │  Every request:
        │    X-Cannon-API-Key: cn_live_…     (application credential)
        │  When user signed in:
        │    Authorization: Bearer <JWT>     (user access token)
        ▼
┌───────────────────────────────────────┐
│  Cannon HTTP router                   │
│  /api/v1/*  /api/docs/*               │
└───────────────────────────────────────┘
        │
        ├── API key middleware (valid active credential)
        ├── JWT middleware (optional → user_id + viewer groups)
        ├── Content handlers (reuse internal/content queries)
        ├── Account handlers (reuse internal/user, auth/mfa, notifications)
        └── OpenAPI + Swagger UI
        │
        ▼
   Site database
```

**Visibility resolution (same as frontend)**

| Request | Viewer groups | Content visible |
|---------|---------------|-----------------|
| API key only, no JWT | `public` group only | Published items/categories with no restriction or assigned to `public` |
| API key + valid user JWT | User's frontend groups + `public` | Same as logged-in browser session |

Implementation reuses:

- `groups.UserGroupIDs(ctx, userID)` when JWT present
- `groups.ViewerGroupIDs` logic for anonymous (public only)
- `content.VisibleItemsQuery`, `content.ItemBySlug`, `content.CategoryBySlug`, `groups.CanViewContent`

The API does **not** use admin session cookies or CSRF for routine requests. **User auth is JWT-based** for v1. Some multi-step flows (MFA login, TOTP enrollment, passkey WebAuthn) require **short-lived opaque server-side state** (`mfa_token`, `totp_setup_token`) because the existing `internal/auth/mfa` package stores pending steps in session today — API adapters will issue tokens instead of cookies, but ephemeral server storage is still required for those steps.

---

## URL layout and versioning

### Path prefix

```
/api/v1/…
```

### Endpoints (overview)

**Auth & account (JWT required except login / password-reset)**

```
POST /api/v1/auth/login
POST /api/v1/auth/mfa                    (when MFA enrolled; phase 1a)
POST /api/v1/auth/refresh                (optional; phase 2)
GET  /api/v1/auth/me
PATCH /api/v1/auth/me                    (profile identity + custom fields)
POST /api/v1/auth/password               (change password)
POST /api/v1/auth/avatar                 (multipart upload)
DELETE /api/v1/auth/avatar
GET  /api/v1/auth/notifications          (phase 1b)
PUT  /api/v1/auth/notifications          (phase 1b)
POST /api/v1/auth/security/totp/begin    (phase 1b)
POST /api/v1/auth/security/totp/confirm
POST /api/v1/auth/security/totp/disable
POST /api/v1/auth/password-reset/request (phase 1b; no JWT)
POST /api/v1/auth/password-reset/confirm (phase 1b; no JWT)
POST /api/v1/auth/logout
```

**Content (read-only; JWT optional for group visibility)**

```
GET /api/v1/items
GET /api/v1/items/{id}
GET /api/v1/items/by-slug/{slug}
GET /api/v1/categories
GET /api/v1/categories/{id}
GET /api/v1/tags
GET /api/v1/tags/{id}
GET /api/v1/media/{id}
GET /api/v1/search
GET /api/v1/authors/{id}
GET /api/v1/items/{id}/comments          (approved only)
POST /api/v1/items/{id}/comments         (phase 1b; guest or JWT + captcha)
```

**Documentation**

```
GET /api/v1/openapi.json
GET /api/v1/docs
```

### Version policy

| Rule | Detail |
|------|--------|
| Major version in path | Breaking changes → `/api/v2/` |
| Additive changes | New JSON fields, new optional query params stay in v1 |
| Deprecation | `Sunset` / `Deprecation` headers when v2 ships |
| No unversioned `/api/…` | Clients must pin a version |

### Reserved system routes

Add to `internal/router/system_routes.go`:

| Path | Handler |
|------|---------|
| `/api/v1/*` | Content API v1 |
| `/api/docs/*` | Swagger UI |
| `/api/v1/openapi.json` | OpenAPI document |

---

## Authentication

Two independent layers work together on every content request.

### 1. Application credential (required)

Identifies the **headless application** allowed to call the API. Managed in **Admin → API → Credentials**.

```
api_credentials
  credential_id    PK
  name             string(128) not null
  token_prefix     string(16) not null
  token_hash       string(128) not null
  status           active | inactive
  expires_at       timestamp nullable
  last_used_at     timestamp nullable
  created_by       user_id nullable FK
  created_at, updated_at
```

**Token format:** `cn_live_<random>` (shown once on create; store hash only).

**Header (required on all `/api/v1/*` except openapi/docs):**

```http
X-Cannon-API-Key: cn_live_…
```

Alternative: same value as `Authorization: Bearer cn_live_…` **only when no user JWT is sent**. When a user JWT is present, use `X-Cannon-API-Key` for the app credential and `Authorization: Bearer <jwt>` for the user (see below).

**Middleware:** validate prefix + hash, check `status` and `expires_at`, attach `credential_id` to context, update `last_used_at` (throttled).

No per-credential content scopes in v1 — credentials are on/off gates for API access. Rate limits may be tied to credential later.

### 2. User JWT (optional on content reads; required on account endpoints)

Identifies the **end user** for group-based content visibility and account self-service. Obtained via login; not used for administration.

**Header when user is signed in:**

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs…
X-Cannon-API-Key: cn_live_…
```

**JWT access token claims (proposed)**

| Claim | Value |
|-------|-------|
| `sub` | User ID (`user_id`) |
| `sid` | Site ID |
| `typ` | `access` |
| `iat`, `exp` | Issued / expiry (e.g. 1 hour access) |
| `jti` | Unique token id (optional; for denylist on logout later) |

Groups are **not** embedded in JWT; resolve fresh from DB via `groups.UserGroupIDs` on each request so admin group changes take effect immediately.

**Signing:** HMAC-SHA256 (or RS256) with a per-site secret stored in settings (`api_jwt_secret`, generated on first use). Never commit secrets.

**Refresh tokens:** Defer to phase 2; v1 may use short-lived access tokens only and re-login.

---

## Auth endpoints

### `POST /api/v1/auth/login`

Authenticates a **frontend** user (not admin). Requires `X-Cannon-API-Key`.

**Request**

```json
{
  "username": "jane",
  "password": "secret",
  "captcha_token": "…"
}
```

`username` may be email or username (same as frontend login). `captcha_token` is required when the site has captcha enabled on login (`captcha.CaptchaContextLogin`), same as the rendered login form.

**Success — no MFA**

```json
{
  "access_token": "eyJ…",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "user_id": 12,
    "username": "jane",
    "email": "jane@example.com",
    "given_name": "Jane",
    "family_name": "Doe",
    "avatar_url": "/assets/…"
  }
}
```

**Success — MFA required (phase 1a)**

```json
{
  "mfa_required": true,
  "mfa_token": "opaque-temporary-token",
  "methods": ["totp", "passkey"]
}
```

Client then calls `POST /api/v1/auth/mfa` with `mfa_token` + TOTP code (or passkey assertion). Reuse existing `internal/auth/mfa` validation; issue JWT only after MFA passes. `mfa_token` maps to server-side pending MFA state (session equivalent).

**Success — verification required**

When credentials are valid but the account is not verified (`!user.Validated`), the frontend redirects to verify-resend instead of starting a session. The API returns:

```json
{
  "verify_required": true,
  "message": "Check your email to verify your account.",
  "resend_available": true
}
```

HTTP `403`. No JWT is issued until the account is verified.

**Errors**

| HTTP | Condition |
|------|-----------|
| 401 | Invalid credentials |
| 403 | Account not verified (`verify_required` body), locked, or inactive |
| 429 | Rate limited |

**Hooks:** Fire `onUserBeforeLogin` / `onUserAfterLogin` with `context: "api"` (new context value alongside `frontend` and `admin`). On successful JWT issuance, call `user.EnsureRegisteredGroup` (same as `completeFrontendLogin`).

**Does not:** Create admin session, set cookies, or grant admin permissions.

### `GET /api/v1/auth/me`

Returns the current user from JWT. Requires API key + valid JWT. Mirrors the profile page read state.

```json
{
  "user_id": 12,
  "username": "jane",
  "email": "jane@example.com",
  "given_name": "Jane",
  "family_name": "Doe",
  "avatar_url": "…",
  "groups": ["public", "registered"],
  "has_local_password": true,
  "custom_fields": [
    { "field_id": 3, "name": "bio", "type": "textarea", "value": "…" }
  ]
}
```

`groups` lists frontend group **names** (not role names). No admin permissions or roles exposed. `custom_fields` follows the site's active author profile field definitions (`content.ActiveProfileFields`).

### `PATCH /api/v1/auth/me`

Updates the signed-in user's profile. Requires API key + JWT. Reuses `user.UpdateProfileIdentity` and profile field save logic from `internal/controllers/auth/profile.go`.

**Request**

```json
{
  "given_name": "Jane",
  "family_name": "Doe",
  "username": "jane",
  "email": "jane@example.com",
  "custom_fields": {
    "3": "Updated bio text"
  }
}
```

All fields optional (partial update). `custom_fields` keys are `field_id` strings; file/media profile fields use separate upload flows (phase 2) or multipart on this endpoint.

**Response:** `200` with updated user object (same shape as `GET /auth/me`).

**Errors:** `409` username/email taken; `400` validation errors.

### `POST /api/v1/auth/password`

Change password for the signed-in user. Requires API key + JWT.

```json
{
  "current_password": "old",
  "new_password": "newsecret",
  "confirm_password": "newsecret"
}
```

`current_password` required when the user has a local password (`user.HasLocalPassword`). Reuses `user.UpdatePassword`.

### `POST /api/v1/auth/avatar` / `DELETE /api/v1/auth/avatar`

Upload or remove the user's avatar. `POST` accepts `multipart/form-data` with `avatar` file field; reuses `user.SaveAvatarUpload` / `user.ClearAvatar`. `DELETE` clears avatar. Requires API key + JWT.

### `GET /api/v1/auth/notifications` / `PUT /api/v1/auth/notifications` (phase 1b)

Read and update Layer 2 notification event subscriptions for the signed-in user. Reuses `notifications.LoadUserProfileState` and `notifications.SaveUserSubscriptions`.

**`GET` response** (mirrors account profile UI state):

```json
{
  "groups": [
    { "id": "account", "label": "Account", "events": ["onUserAfterLogin", "onUserVerified"] },
    { "id": "content", "label": "Content", "events": ["onItemAfterSave"] },
    { "id": "comments", "label": "Comments", "events": ["onCommentAfterSave"] }
  ],
  "checked": { "onUserAfterLogin": true, "onItemAfterSave": false },
  "role_defaults": { "onUserAfterLogin": true }
}
```

Event ids are **hook names** from `internal/hooks` (e.g. `onItemAfterSave`, `onCommentAfterSave`), not dotted CMS paths.

**`PUT` request** — selected event ids to subscribe (same form field as frontend: `notification_events`):

```json
{
  "events": ["onItemAfterSave", "onCommentAfterSave"]
}
```

### `POST /api/v1/auth/password-reset/request` / `confirm` (phase 1b)

Unauthenticated password reset flow mirroring `auth/reset-request` and `auth/reset-submit`. Requires API key + captcha token (same extension captcha contexts as frontend). Returns generic success even when email unknown (no enumeration).

### `POST /api/v1/auth/logout`

v1: optional endpoint that returns `204`; client deletes JWT. Server-side denylist deferred.

---

## Content visibility rules

### Published-only

All content endpoints return only **published** items within publish window (`content.PublishedScope`), matching the public frontend. No draft, pending, archived, or trashed content via API.

### Group visibility (JWT-driven)

| Content | Rule |
|---------|------|
| Items | `content.VisibleItemsQuery(ctx, viewerGroups)` |
| Item by slug | `content.ItemBySlug` + `groups.CanViewContent` |
| Categories | `content.CategoryBySlug` / tree filtered by viewer groups |
| Routes, blocks, menus | Not exposed in v1 (frontend routing is client responsibility) |
| Media | No `GET /media` collection in v1; only `GET /media/{id}` for assets referenced by visible content (or direct id when metadata is not restricted) |
| Comments | Approved comments on visible items only |
| Search | Same visibility filter as item list |

### Anonymous vs authenticated

```
Anonymous (API key only):
  viewerGroups = [public_id]

Authenticated (API key + JWT):
  viewerGroups = groups.UserGroupIDs(ctx, user_id)
  // includes public + registered + any assigned frontend groups
```

This mirrors `internal/groups/groups.go` `ViewerGroupIDs` behavior without a browser session.

---

## Admin UI — API section

### Navigation

New admin group **API** (not under System):

```
API
  ├── Credentials     /admin/api/credentials
  └── Documentation   /admin/api/docs   (link to Swagger UI)
```

Settings (CORS, JWT TTL, rate limits) under **Configuration → API** or **API → Settings** in phase 2.

### Credentials CRUD

Same as prior draft but **without scope checklists** — credentials are simple on/off keys.

| Action | Behavior |
|--------|----------|
| Create | Name, optional expiry → show token once |
| Edit | Rename, activate/deactivate, extend expiry |
| Rotate | Invalidate old hash, issue new token |
| List | Name, prefix, status, last used, expires |

### RBAC (admin)

| Permission | Purpose |
|------------|---------|
| `core.admin.api.read` | View credentials (masked), open docs |
| `core.admin.api.write` | Create, rotate, revoke credentials |

Administrators only by default. **No API access to manage credentials** — admin UI only.

### JWT secret

- Auto-generated per site on first API enable (stored in global settings).
- Admin may rotate JWT secret (invalidates all outstanding user JWTs).
- Shown/regenerated under **API → Settings** (phase 2) or Configuration.

---

## OpenAPI / Swagger

- **OpenAPI 3.1** at `GET /api/v1/openapi.json`
- **Swagger UI** at `GET /api/docs`
- Checked-in spec: `api/openapi/v1.yaml`

**Security schemes**

```yaml
securitySchemes:
  ApiKeyAuth:
    type: apiKey
    in: header
    name: X-Cannon-API-Key
  UserBearerAuth:
    type: http
    scheme: bearer
    bearerFormat: JWT
```

- Auth login / password-reset request: `ApiKeyAuth` only
- Content read: `ApiKeyAuth` + optional `UserBearerAuth`
- Account write (`PATCH /auth/me`, password, avatar, notifications, security): `ApiKeyAuth` + `UserBearerAuth` (required)
- Document both in operation `security` arrays

**Tags:** Auth, Account, Items, Categories, Tags, Media, Search, Authors, Comments

---

## API conventions

### JSON

- Request/response: `application/json`
- Timestamps: RFC 3339 UTC
- IDs: numeric (`item_id`, `category_id`)
- Locale: `?locale=en-US` or `Accept-Language`

### Pagination

```json
{
  "data": [],
  "meta": { "page": 1, "page_size": 20, "total": 142 }
}
```

Max `page_size`: 100.

### Errors

```json
{
  "error": {
    "code": "unauthorized",
    "message": "Invalid or missing API key"
  }
}
```

| HTTP | Meaning |
|------|---------|
| 400 | Bad request / validation |
| 401 | Missing/invalid API key or JWT |
| 403 | Valid auth but user not allowed (locked, unverified) |
| 404 | Not found or not visible (same as frontend — do not leak existence of restricted content) |
| 429 | Rate limited |
| 500 | Server error |

### CORS

Phase 2: allowed origins in admin settings for browser-based headless apps calling Cannon directly.

---

## Resource surface (v1)

### Auth & account

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/login` | API key | Issue user JWT |
| POST | `/auth/mfa` | API key | Complete MFA challenge (1a) |
| GET | `/auth/me` | API key + JWT | Current user + profile fields |
| PATCH | `/auth/me` | API key + JWT | Update profile identity + custom fields |
| POST | `/auth/password` | API key + JWT | Change password |
| POST | `/auth/avatar` | API key + JWT | Upload avatar (multipart) |
| DELETE | `/auth/avatar` | API key + JWT | Remove avatar |
| GET | `/auth/notifications` | API key + JWT | Notification subscription state (1b) |
| PUT | `/auth/notifications` | API key + JWT | Update subscriptions (1b) |
| POST | `/auth/security/totp/begin` | API key + JWT | Start TOTP enrollment (1b) |
| POST | `/auth/security/totp/confirm` | API key + JWT | Confirm TOTP with code (1b) |
| POST | `/auth/security/totp/disable` | API key + JWT | Disable TOTP (1b) |
| POST | `/auth/password-reset/request` | API key | Request reset email (1b) |
| POST | `/auth/password-reset/confirm` | API key | Set new password from token (1b) |
| POST | `/auth/logout` | API key + JWT | Client logout |

### Content (read)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/items` | API key (+ JWT optional) | Paginated visible items |
| GET | `/items/{id}` | API key (+ JWT optional) | Item by ID |
| GET | `/items/by-slug/{slug}` | API key (+ JWT optional) | Item by slug + locale |
| GET | `/categories` | API key (+ JWT optional) | Category tree |
| GET | `/categories/{id}` | API key (+ JWT optional) | Single category |
| GET | `/tags` | API key | Tag list |
| GET | `/tags/{id}` | API key | Single tag |
| GET | `/search` | API key (+ JWT optional) | FTS + field filters |
| GET | `/authors/{id}` | API key | Public author profile |
| GET | `/media/{id}` | API key | Media metadata |

### Engagement (write — frontend parity)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/items/{id}/comments` | API key (+ JWT optional) | Approved comments |
| POST | `/items/{id}/comments` | API key (+ JWT optional) | Post comment; captcha required (1b) |

**`POST /items/{id}/comments` request** (guest or signed-in):

```json
{
  "body": "Great article!",
  "author_name": "Guest",
  "author_email": "guest@example.com",
  "captcha_token": "…"
}
```

`author_name` / `author_email` required for guests; omitted when JWT present (author taken from user). Captcha required for all submissions when enabled (`captcha.CaptchaContextComment`). Reuses `content.CreateComment`.

**Item JSON (representative fields)**

`item_id`, `locale`, `translation_group_id`, `title`, `slug`, `intro`, `body`, `featured`, `publish_start`, `publish_end`, `author`, `category`, `tags`, `image`, `gallery`, `embeds`, `attachments`, `meta`, `custom_fields`, `created_at`, `updated_at`

- Never expose `preview_token`, internal status beyond published, or admin-only fields.
- `body` as stored; optional `?format=html` rendering later.

---

## Security considerations

| Topic | Approach |
|-------|----------|
| API key storage | Hash only; show once on create |
| JWT secret | Per-site; rotatable; min 256-bit entropy |
| HTTPS | Required in production |
| Login rate limit | Per IP + per username (reuse/brute-force patterns from auth) |
| JWT expiry | Short-lived access token (default 1h) |
| Content enumeration | 404 for restricted items (same as frontend) |
| Admin separation | JWT grants frontend group visibility only, not `core.admin.*` |
| MFA | Same enrollment as web; API completes MFA step before JWT |
| Injection | Reuse parameterized content queries |

---

## Rate limiting (phase 2)

- Per API credential + per IP defaults
- Stricter limits on `/auth/login`
- `429` + `Retry-After`

---

## Multilingual

When `content_multilingual` is enabled:

- `?locale=` on list/detail endpoints
- Response includes `locale`, `translation_group_id`
- `GET /items/{id}/translations` — phase 2 (read sibling locales in group)

---

## Implementation phases

### Phase 0 — Plumbing

- [x] Spec approved
- [x] `api_credentials` model
- [x] System routes `/api/v1/*`, `/api/docs`
- [x] Admin **API** nav + credentials CRUD
- [x] Permissions `core.admin.api.read` / `write`
- [x] Per-site JWT secret in settings

### Phase 1a — Auth + account + core read

- [x] API key middleware
- [x] `POST /auth/login` → JWT (including `POST /auth/mfa` when MFA enrolled)
- [x] JWT middleware → viewer groups
- [x] `GET /auth/me`, `PATCH /auth/me`
- [x] `POST /auth/password`, avatar upload/delete
- [x] Read: items, categories, tags, search (visibility parity tests)
- [x] OpenAPI + Swagger UI

### Phase 1b — Notifications, comments, password reset, TOTP security

- [x] `GET` + `PUT /auth/notifications`
- [x] `POST /auth/security/totp/*`
- [x] `GET` + `POST /items/{id}/comments`
- [x] `POST /auth/password-reset/*`
- [x] Authors endpoint

### Phase 2 — Operations & remaining parity

- [x] CORS settings
- [x] Rate limiting
- [x] JWT refresh tokens (optional)
- [~] Passkey login + passkey security endpoints (`POST /auth/passkey-login`, `/auth/security/passkeys/*`) — MFA response includes passkey URLs; full WebAuthn flow returns 501 until wired
- [x] Account verification + resend
- [x] Translation group helper endpoint
- [x] Media metadata policy finalized (visible items only)
- [ ] Profile field file uploads via API

### Explicitly out of scope

- CMS admin APIs (`/admin/*`)
- Frontend contributor item create/edit (`/content/edit`)
- Draft or preview tokens via API
- GraphQL
- Webhooks

---

## Related docs

- `.info/specs/content.md` — content model
- `.info/specs/user_group_role.md` — frontend groups vs capability roles
- `.info/specs/event_hooks.md` — login hooks (`context: "api"`)
- `.info/TODO.md` — headless checklist

---

## Resolved decisions (from review)

| Question | Decision |
|----------|----------|
| CMS administration via API? | **No** — `/admin/*` operations excluded |
| Account self-service via API? | **Yes** — profile, password, avatar, notifications, MFA (frontend parity) |
| Group visibility for content? | **Yes** — JWT carries user identity; reuse `ViewerGroupIDs` / `UserGroupIDs` |
| Published content writes? | **No** — no item/category admin; contributor `/content/edit` deferred |
| Auth model | **API key (app) + JWT (user)** |
| GraphQL | **Deferred** |
| Login MFA | **Ship with 1a** — login is unusable for MFA-enrolled users without `POST /auth/mfa` |
| Captcha | **`captcha_token` in JSON** when site captcha enabled (login, comments, password reset) |

---

## Open questions

1. **Media by id:** Confirm `GET /media/{id}` is allowed without visibility checks, or restrict to assets on visible items only?
2. **JWT TTL:** Default 1 hour — acceptable for headless SPAs?
3. **Swagger UI:** Public at `/api/docs`, or admin-only?
4. **Passkey MFA on API:** Required in 1b or defer to 2?
5. **OAuth:** Future `POST /auth/oauth/{provider}` token exchange — document now as non-goal?
6. **Contributor editing:** Should headless apps support `/content/edit` parity for writers, or is admin UI sufficient?
7. **Captcha on API:** Resolved for v1 — pass `captcha_token` on login, comment, and password-reset when captcha is enabled.

---

## TODO tracker

- [ ] Content API spec approved (this file)
- [ ] Phase 0: credentials + routes + admin nav
- [ ] Phase 1a: API key + JWT login/MFA + account profile + read endpoints + Swagger
- [ ] Phase 1b: notifications + comments + password reset + TOTP security
