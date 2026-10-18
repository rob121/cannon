# Cannon — Tracked TODO

Status key: `[ ]` pending · `[~]` partial · `[x]` done

*Last audited against codebase: 2026-10-18.*

---

## Specs — extensions.md

- [x] Extension process model, socket HTTP, capabilities (request/page/data/endpoint/block/admin/help/hooks/templates/configuration/captcha)
- [x] CSRF on wire requests, SQLite WAL, extension DB access
- [x] Captcha capability contract + Cannon placeholder expansion and verify on login/comments
- [x] Extension permissions — `RegisterPermissions`, host sync, `UserCan`, wire `permissions` / `denied_permissions`

- [~] Extension block admin metadata (thinner than page/endpoint routes; `BlockDefinition.Fields` exists)



## Specs — [blocks.md](http://blocks.md)

- [x] Spaces, `{{space}}`, admin space filter, HTML/Markdown/extension blocks
- [x] Block wrapper templates under `partials/blocks/`
- [x] Block show/hide by route (`RouteMode` / `RouteIDs` + admin UI)
- [x] Template wrapper dropdown from theme scan (`ListWrapperTemplates`)
- [x] Menu vertical/horizontal block types (beyond spec)

- [x] Multi-space assignment per block (`block_spaces` join + per-space sort)



## Specs — [templates.md](http://templates.md)

- [x] Theme folders, `template.json`, frontend/admin theme selection, versioning
- [x] `/theme/*`, `/admin/assets/*`, admin theme browser, override flow
- [x] Legacy `template_dir` layout migration helper (Admin → Templates detect + confirm)

- [x] Template editor save reliability (`fsutil.WriteFileAtomic` for templates, assets, meta)
- [x] Theme/file list pagination (theme root, theme browse, assets)



## Specs — event_hooks.md

- [x] All documented events wired (in-process + extension)
- [x] Document head/body hooks (`onPrepareDocumentHead`, `onAdminPrepareDocumentHead`, etc.)
- [x] Content lifecycle hooks (delete, trash, restore, category, media, revision)
- [x] Search, mail, sitemap, robots, settings hooks
- [x] `onUserLocked` — fire from admin user lock toggle
- [x] `onUserSignup` / `onUserVerified` — fired from auth flows



## Specs — [notifications.md](http://notifications.md)

See `.info/specs/notifications.md` for Layer 1/2 design.

- [x] **Layer 1 — Admin destinations** — `Notification` model, Shoutrrr send, System → Notifications CRUD
- [x] **Layer 1 — Auth + content/comment hooks** — signup/login/verify/lock + `onItemAfterSave` / `onCommentAfterSave` in event picker

- [x] **Layer 1 — Message templates** — per-destination per-event subject/body with `{{placeholders}}`
- [x] **Layer 1 — Delivery error visibility** — `last_error` / `last_sent_at` on destinations + admin list/form

- [x] **Layer 2 —** `notification_subscriptions` **model** — user_id or role_id + event + channel
- [x] **Layer 2 — Dispatch** — resolve role members, dedupe emails, send via site mail
- [x] **Layer 2 — Account UI** — per-user event subscription checklist
- [x] **Layer 2 — Role admin UI** — default subscriptions per role
- [x] **Layer 2 — Filters** — `filters_json` applied (`status`, `category`, `author_self`) + role form controls
- [ ] **Layer 2 — In-app inbox** — optional `channel: in_app` (later)



## Specs — admin_design.md

- [x] Bootstrap 5, Turbo, admin CSS, collapse sidebar
- [x] Unified list card headers / toolbars

- [~] Apex-shadcn visual parity (palette/theme in place; ongoing polish)
- [~] Pagination on all tables (most CRUD use `pageSizeFor`; menu items + template file lists still unpaginated)
- [~] Title Case on all headers/buttons (many still sentence case)



## Specs — [content.md](http://content.md)

See `.info/specs/content.md` for full feature list.

- [x] Items, categories, tags, custom fields, frontend editing
- [x] Comments, search (SQLite FTS5 + bm25 + custom-field filters; LIKE fallback), feeds, sitemap, author pages
- [x] Global content settings (display toggles, author profile schema; frontend permissions via RBAC roles)
- [x] Content revisions + trash restore; editorial submit/review/publish; draft preview
- [x] Multilingual content — locale + `translation_group_id`, URL locale prefixes (`/fr-FR/...`)

- [~] Content spec polish (media picker, featured ordering, empty states, item form tabs)

- [ ] Remaining content spec items — review `content.md` for leftover gaps after recent work



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
- [x] **OAuth SSO providers** — Goth wired end-to-end (`BeginOAuth` / `CompleteOAuth`); login UI shows configured/active providers

---



## Notifications

Spec: `.info/specs/notifications.md`. Two layers: admin destinations (Layer 1) + user/role subscriptions (Layer 2).

**Layer 1 — done / extend**

- [x] Model + Shoutrrr UI under System → Notifications
- [x] Auth + content/comment hook subscriptions in admin event picker
- [x] Message templates; delivery error visibility

**Layer 2 — user + role subscriptions**

- [x] Subscription model (`user_id` or `role_id`, event, channel, status, optional filters column)
- [x] Dispatch: role → member emails, dedupe, user opt-out overrides role default
- [x] Account/profile notifications tab (per-user)
- [x] Role form: default notification subscriptions
- [x] Apply `filters_json` in dispatch
- [ ] In-app inbox channel

- [~] Depends on mail config (From, SMTP) for reliable email delivery — **mail settings exist** (`smtp_username` / `smtp_password` / `from_name` in Configuration → Mail)

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
- [x] Assign registered group on login (`EnsureRegisteredGroup` from login helpers, admin login, API auth)
- [ ] Production site auth migration smoke test (manual)

---



## Configuration & system

- [x] Site offline, log level, default list limit (stored + used via `pageSizeFor` / `settings.DefaultListLimit`)
- [x] Global default meta tags in Configuration → General (description, keywords, OG, Twitter, head extra)
- [x] Captcha settings in Configuration → General (enable, active extension, skip authenticated)
- [x] Access log (file rotation + System → Access Log tail UI)
- [x] Mail config — SMTP host/port, username/password, from address/name (`internal/settings/definitions/mail.json` + `internal/mail`)
- [x] Site Display Name — `general.site_name` drives admin brand / titles (`adminSiteDisplayName`)
- [x] Cannon self-update — stage `cannon.latest`, startup swap, rollback via `cannon.previous`, admin System → Update UI + polling

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

- [x] Block page visibility / route tree not expanding correctly — auto-open sections with checked `route_ids` in `initBlockRoutePicker`
- [x] Category route with predefined slug shows empty on home — fixed-path slug from metadata; `DefaultRoutePath` / pagination base resolve concrete paths for wildcard+slug routes

---



## Extensions (runtime)

Out-of-repo extension polish (tracked here for product completeness):

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
- [x] OAuth SSO (Goth) login flow for configured authenticators
- [x] Mail SMTP credentials + from name in Configuration → Mail
- [x] Registered group assignment on login
- [x] Site Display Name in admin chrome
- [x] SQLite FTS5 search + custom field filters
- [x] Multilingual content locales + URL prefixes + translation groups
- [x] Cannon binary self-update (stage/swap/rollback) + interactive admin update UI
- [x] Extension update checks via GitHub Releases API (prefer API over CDN-cached latest/download)
- [x] README restructure (features, setup, templating, production notes)
- [x] Media image styles — global named derivatives + `imageStyle` helper
- [x] Anonymous page cache + item object cache (Configuration → General)
- [x] Templates polish — atomic saves, list pagination, legacy theme migration helper

---



## CMS parity — priority (Joomla / Drupal gaps)

- [x] OAuth SSO + mail — unlock real deployments
- [x] Content revisions + restore from trash — editorial safety net
- [x] Multilingual content — language-specific items/categories/URLs
- [x] Search upgrade — full-text index + custom field filters (SQLite FTS5; no ES/Solr)
- [x] Media polish — picker, image styles (global named styles), drag/drop (see Admin UX polish / media section)
- [x] Editorial workflow — submit/review/publish for teams
- [x] Headless content API — published content read + JWT account parity (see `.info/specs/content_api.md`)
- [x] One-click core updates — admin System → Update (staged binary + systemd restart)

---



## CMS parity — editorial workflow

- [x] Submit for review, pending state, approver queue (Drupal Moderation / Joomla Workflow)
- [x] Content item revisions — rollback, compare, audit trail (`/admin/items/{id}/revisions`)
- [x] Draft preview — secret preview URL for unpublished content
- [x] Trash manager — dedicated restore action and empty-trash UX (trash status exists today)

---



## CMS parity — multilingual

- [~] UI string translations — `.ini` language files (admin/site scopes)

- [x] Translated content — language-specific items/categories and associations (`translation_group_id`)
- [x] Localized URLs — locale prefix stripping + `LocalePrefix` (e.g. `/fr-FR/...`)

---



## CMS parity — search & discovery

- [x] Item search — SQLite FTS5 index, bm25 ranking, custom-field filters; LIKE fallback if FTS unavailable
- [ ] Optional Elasticsearch/Solr backend (not required for parity baseline)

- [~] View/analytics-based popularity — live visitor presence via Centrifuge; page-view history still pending

---



## CMS parity — integrations & headless

Spec: `.info/specs/content_api.md` — headless frontend parity; read-only published content + JWT account self-service; no CMS admin API.

- [x] Content API — REST + OpenAPI; app credentials + JWT login + account/profile writes
- [x] OAuth SSO — Goth provider login flow
- [x] Mail transport — SMTP username/password/from name

---



## CMS parity — media

- [x] Thumbnails / image styles — named global styles (`thumbnail`/`medium`/`large`); `_thumb` compat; `imageStyle` template helper

- [~] Configurable image sizes — global styles shipped; per-category style maps still open

- [x] Media UX — file explorer, drag/drop upload, upload flow polish (see Admin UX polish)
- [ ] Image transforms — on-the-fly crop, focal point, WebP variants, CDN/Imgix-style URLs

---



## CMS parity — platform & ops

- [x] Page/object cache — anonymous full-page LRU + item object cache; site flush on content/layout/settings (cache tags / Varnish / CDN later)
- [ ] Staging environment — content/config sync between dev/stage/prod
- [x] Core self-update — admin System → Update (`cannon.latest` / `.previous`, systemd `Restart=always`)
- [x] Site backup manager — provided by backup extension (not core)

---



## CMS parity — content modeling

- [~] Single “Item” type with field groups per category (not distinct content types)

- [ ] Distinct content types — Article vs Page vs Product with separate admin menus
- [ ] Hierarchical taxonomies / vocabularies — tags are flat today (Drupal Taxonomy); categories already hierarchical
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

---



## BugFix

- [x] Buttons on `/admin/api/settings` for save and back — actions moved inside card body with standard `.admin-form-actions`
- [x] Site General Configuration Site Display Name should show instead of "Cannon" in the admin menu — wired via `adminSiteDisplayName` / `{{.SiteName}}`
- [x] Menu edit page up/down sort arrows for items (like registry / menu-items list) — `/admin/menus/{id}`
- [x] Block edit space suggestions from theme `{{space}}` scan + DistinctSpaces + extension `BlockDefinition.Spaces`

---



## Open backlog (highest leverage remaining)

Use this list when picking the next workstream:

1. **Modeling** — distinct content types; hierarchical tag vocabularies
2. **Media (follow-ups)** — per-category styles; crop/focal; on-the-fly / CDN transforms
3. **Ops (follow-ups)** — cache tags / Varnish / shared Redis; staging sync
4. **Extension polish** (separate repos) — Contact Bootstrap markup; Calendar `meeting_details`
5. **Notifications** — in-app inbox channel (Layer 1/2 polish shipped)





