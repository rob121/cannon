# Cannon — Tracked TODO

Status key: `[ ]` pending · `[~]` partial · `[x]` done

---

## Specs — extensions.md

- [x] Extension process model, socket HTTP, capabilities (request/page/data/endpoint/block/admin/help/hooks/templates/configuration/captcha)
- [x] CSRF on wire requests, SQLite WAL, extension DB access
- [x] Captcha capability contract + Cannon placeholder expansion and verify on login/comments
- [x] Extension permissions — `RegisterPermissions`, host sync, `UserCan`, wire `permissions` / `denied_permissions`
- [~] Extension block admin metadata (thinner than page/endpoint routes)

## Specs — blocks.md

- [x] Spaces, `{{space}}`, admin space filter, HTML/Markdown/extension blocks
- [x] Block wrapper templates under `partials/blocks/`
- [ ] Multi-space assignment per block (spec: one or more spaces)
- [~] Block show/hide by route (admin visibility)
- [~] Template wrapper dropdown from theme scan
- [x] Menu vertical/horizontal block types (beyond spec)

## Specs — templates.md

- [x] Theme folders, `template.json`, frontend/admin theme selection, versioning
- [x] `/theme/*`, `/admin/assets/*`, admin theme browser, override flow
- [ ] Legacy `template_dir` layout migration helper
- [~] Template editor save reliability
- [~] Theme/file list pagination

## Specs — event_hooks.md

- [x] All documented events wired (in-process + extension)
- [x] `onUserLocked` — fire from admin user lock toggle
- [x] `onUserSignup` / `onUserVerified` — fired from auth flows

## Specs — notifications.md

See `.info/specs/notifications.md` for Layer 1/2 design.

- [x] **Layer 1 — Admin destinations** — `Notification` model, Shoutrrr send, System → Notifications CRUD
- [x] **Layer 1 — Auth hook subscriptions** — `onUserSignup`, `onUserAfterLogin`, `onUserVerified`, `onUserLocked`
- [~] **Layer 1 — Expand event picker** — content/comment hooks in data model; admin form still flat list
- [x] **Layer 2 — `notification_subscriptions` model** — user_id or role_id + event + channel
- [x] **Layer 2 — Dispatch** — resolve role members, dedupe emails, send via site mail
- [x] **Layer 2 — Account UI** — per-user event subscription checklist
- [x] **Layer 2 — Role admin UI** — default subscriptions per role
- [ ] **Layer 2 — Filters** — category, status transition, author=self (`filters_json`)
- [ ] **Layer 2 — In-app inbox** — optional `channel: in_app` (later)

## Specs — admin_design.md

- [x] Bootstrap 5, Turbo, admin CSS, collapse sidebar
- [~] Apex-shadcn visual parity
- [~] Pagination on all tables (menu items missing; some cosmetic-only)
- [~] Title Case on all headers/buttons
- [x] Unified list card headers / toolbars

## Specs — content.md

See `.info/specs/content.md` for full feature list.

- [x] Items, categories, tags, custom fields, frontend editing
- [x] Comments, search, feeds, sitemap, author pages
- [x] Global content settings (display toggles, author profile schema; frontend permissions via RBAC roles)
- [~] Content spec polish (media picker, featured ordering, empty states, item form tabs)
- [ ] Remaining content spec items — review `content.md` §6–§16 for gaps

## Specs — user_group_role.md

See `.info/specs/user_group_role.md` and `.info/TODO_AUTH.md` for implementation detail.

- [x] RBAC engine, roles, permissions, inheritance, caching
- [x] Admin and content frontend permission namespaces
- [x] Visibility groups separate from capability roles
- [x] Explicit deny permissions
- [x] Extension permission registration
- [x] Legacy site migration path

---

## Auth & security

- [x] **RBAC authorization** — full system in `internal/security` (see `.info/TODO_AUTH.md`; phases 0–8 complete)
- [x] **MFA (TOTP) + passkeys** — global toggles, account security UI, login MFA step, admin MFA redirect
- [ ] OAuth SSO providers (Goth wired; login returns “not available yet”)

---

## Notifications

Spec: `.info/specs/notifications.md`. Two layers: admin destinations (Layer 1) + user/role subscriptions (Layer 2).

**Layer 1 — done / extend**

- [x] Model + Shoutrrr UI under System → Notifications
- [x] Auth hook subscriptions (signup, login, verify, lock)
- [ ] Content/comment hooks in admin event picker
- [ ] Message templates; delivery error visibility

**Layer 2 — user + role subscriptions**

- [x] Subscription model (`user_id` or `role_id`, event, channel, status, optional filters)
- [x] Dispatch: role → member emails, dedupe, user opt-out overrides role default
- [x] Account/profile notifications tab (per-user)
- [x] Role form: default notification subscriptions
- [~] Depends on mail config (From, SMTP) for reliable email delivery

---

## Users & groups

Detailed checklist: `.info/TODO_AUTH.md`

- [x] RBAC engine — permission registry, role inheritance, wildcards, per-user cache, default deny
- [x] Default system roles (viewer → administrator) with seeded permissions and inheritance
- [x] Admin section permissions (`core.admin.*`) — replaced legacy group admin route matrix
- [x] Content frontend permissions (`core.content.frontend.*`) on roles — removed config/category permission overlays
- [x] Explicit deny permissions on roles (override allows, including wildcards)
- [x] Direct user role assignment; roles, groups, and permissions admin UI
- [x] Effective permissions preview on user edit form
- [x] Extension permissions — host sync, extension nav filtering, `core.admin.extension-apps.read`
- [x] Visibility groups (frontend kind) separate from RBAC — Access tabs on categories, items, routes, blocks
- [x] Backend vs frontend group kinds; admin UI copy and group pickers aligned (membership vs visibility)
- [x] Legacy migration — group admin routes, role names, `core.content.*` → `core.content.frontend.*`
- [x] Help — `admin/authorization.md`, extension authoring permissions section
- [ ] Assign registered group on login (only on create today)
- [ ] Production site auth migration smoke test (manual)

---

## Configuration & system

- [x] Site offline, log level, default list limit (stored)
- [~] `default_list_limit` — admin lists still hard-code page size 20
- [x] Global default meta tags in Configuration → General (description, keywords, OG, Twitter, head extra)
- [x] Captcha settings in Configuration → General (enable, active extension, skip authenticated)
- [x] Access log (file rotation + System → Access Log tail UI)
- [ ] Mail config: username/password, from name

---

## Admin UX polish

- [x] Content list card headers — unified toolbar partial
- [x] Media: file explorer UI, upload preview, fix upload flow
- [x] Items form: top toolbar for save/cancel, less visual noise
- [x] Field groups: required as toggle; fix add-field-within-group bug
- [x] Categories/routes/blocks: sort arrows like extensions
- [x] Routes: group visibility copy aligned; sidebar layout parity with categories
- [x] All admin checkboxes → toggles where missing
- [x] Help: dedupe Extensions entries in Help nav (authoring + authorization docs written)

---

## Blocks & routes (bugs)

- [ ] Block page visibility / route tree not expanding correctly
- [ ] Category route with predefined slug shows empty on home

---

## Extensions (runtime)

- [ ] Contact extension: Bootstrap structure on HTML output
- [ ] Calendar extension: `meeting_details` column; hide Google Meet boilerplate in description

---

## Done recently

- [x] Full RBAC authorization system (`internal/security`, admin UI, migrations, help)
- [x] Explicit deny permissions on roles + effective permissions preview on user form
- [x] Content frontend permissions via roles; visibility groups on Access tabs only
- [x] Extension permissions + filtered extension admin nav
- [x] Admin auth UI alignment — backend/frontend groups, membership vs visibility copy
- [x] Global content frontend permissions via RBAC (`core.content.frontend.*` on roles)
- [x] Author profile schema in Configuration → Content + user account fields
- [x] Access log middleware fix + admin tail viewer
- [x] Theme asset management (list, edit, create, delete) with CodeMirror
- [x] Captcha extension capability + Cannon runtime (placeholders, expand, verify)

---

## CMS parity — priority (Joomla / Drupal gaps)

- [ ] OAuth SSO + mail — unlock real deployments (see Auth & security, Configuration & system)
- [x] Content revisions + restore from trash — editorial safety net
- [x] Multilingual content — language-specific items/categories/URLs
- [x] Search upgrade — full-text index + custom field filters
- [x] Media polish — picker, image styles, drag/drop (see Admin UX polish)
- [x] Editorial workflow — submit/review/publish for teams
- [x] Headless content API — published content read + JWT account parity (see `.info/specs/content_api.md`)

---

## CMS parity — editorial workflow

- [x] Submit for review, pending state, approver queue (Drupal Moderation / Joomla Workflow)
- [x] Content item revisions — rollback, compare, audit trail (`/admin/items/{id}/revisions`)
- [x] Draft preview — secret preview URL for unpublished content
- [x] Trash manager — dedicated restore action and empty-trash UX (trash status exists today)

---

## CMS parity — multilingual

- [~] UI string translations — `.ini` language files (admin/site scopes)
- [ ] Translated content — language-specific items, categories, and associations
- [ ] Localized URLs — e.g. `/fr/blog/post` linked translations (Joomla Language Associations / Drupal content translation)

---

## CMS parity — search & discovery

- [~] Basic item search — SQL `LIKE` on title/intro/body
- [ ] Full-text search — index, relevance ranking, optional Elasticsearch/Solr
- [ ] Custom field filtering in search (spec: `content.md` §13)
- [ ] View/analytics-based popularity — today “popular” uses comment count only

---

## CMS parity — integrations & headless

Spec: `.info/specs/content_api.md` — headless frontend parity; read-only published content + JWT account self-service; no CMS admin API.

- [x] Content API — REST + OpenAPI; app credentials + JWT login + account/profile writes
- [ ] OAuth SSO — finish Goth provider login flow (see Auth & security)
- [ ] Mail transport — SMTP username/password/from name (see Configuration & system)

---

## CMS parity — media

- [~] Thumbnails — single fixed width (320px); no image style profiles
- [ ] Configurable image sizes — global and per-category (Joomla/Drupal image styles)
- [x] Media UX — file explorer, drag/drop upload, upload flow polish (see Admin UX polish)
- [ ] Image transforms — on-the-fly crop, focal point, WebP variants, CDN/Imgix-style URLs

---

## CMS parity — platform & ops

- [ ] Page/object cache — cache tags, Varnish, Joomla-style page cache
- [ ] Staging environment — content/config sync between dev/stage/prod
- [ ] Backup/update manager — Akeeba-style backups or one-click core updates

---

## CMS parity — content modeling

- [~] Single “Item” type with field groups per category (not distinct content types)
- [ ] Distinct content types — Article vs Page vs Product with separate admin menus
- [ ] Hierarchical taxonomies / vocabularies — tags are flat today (Drupal Taxonomy)
- [x] Duplicate/clone item

---

## CMS parity — comments & community

- [~] Comment spam — honeypot + IP rate limit; captcha on comment forms when enabled
- [ ] Akismet, reCAPTCHA, or moderation ML integration
- [x] Event notifications — Layer 1 Shoutrrr + Layer 2 user/role email subscriptions (see `notifications.md`)

---

## CMS parity — architectural notes (not gaps)

These are deliberate design choices, not missing features:

- Component/module/plugin sprawl → extensions + hooks + blocks
- Visual page builder (Gutenberg, SP Page Builder) → templates + blocks + spaces
- Views / query builder (Drupal Views) → content blocks + controller templates + Go
- Granular ACL asset tree (Joomla) → groups + roles + per-section admin permissions + explicit deny + item/category visibility groups

