# Extensions 

Extensions are dynamically loaded processes, ideally they are in go but if they can establish a socket to communicate over http and be started as a process then they are fine.

All extensions are passed a cli of --site=$id and --socket=$location  when started the process should listen on the socket and use the context of site id for it's internal routing.

The extension should speak http and so has a minimal required amount of endpoints

/meta - meta data about the extension (version update link etc.)
/help - help list entries
/health (health check)
/capabilities
A json response that tells the parent process what this extension can do 

{
"capabilities": {
  "request": "request_handler"
  "page": "page_handler"
  "data": "data_handler"
  "endpoint": "endpoint_handler"
  "block": "block_handler"
  "admin": "admin_handler"
  "hooks": "hooks_handler"
  "templates": "templates_handler"
  "captcha": "captcha_handler"
}
}
```

Hooks exposes event subscriptions Cannon dispatches (`onBeforeRoute`, `onPrepareDocumentHead`, `onItemTrash`, `onSitemapGenerate`, etc.). `GET /hooks` returns `{"hooks":["onBeforeRoute"]}`. `POST /hooks` receives `event`, `arguments`, and the normal wire request. Register with `OnHook(event, fn)`. Fragment keys (`head_html`, `body_html`, `robots_append`, `sitemap_urls`) are appended across listeners. See `.info/specs/event_hooks.md`.

Captcha provides render and verify handlers for protected forms (login, registration, comments, extension forms). Cannon uses **one active captcha extension per site** — see the Captcha capability section below. Register with `RegisterCaptcha`. Multiple captcha extensions may be installed, but only the site-selected provider is called.

Request is triggered on a web request early in the process, the extension can make changes to the request body etc. make decisions about blocking, ddecoding etc. It is expected to be passed to in http routing middleware.

Page is to be associated to a route, for example /contact route may be associated with a extension and so when a request to /contact is triggered a sub request to /page/{id} with the request context is passed and the result is captured for output.

GET /page lists page definitions, including optional admin metadata fields:

```json
{
  "pages": [
    {
      "id": "contact-form",
      "title": "Contact Form",
      "fields": [
        {
          "name": "form_id",
          "label": "Contact Form",
          "type": "select",
          "required": true,
          "options": [
            {"value": "1", "label": "General Contact"}
          ]
        }
      ]
    }
  ]
}
```

POST /page/:item renders one page type. Cannon sends the normal wire request plus route page context:

```json
{
  "method": "GET",
  "url": "/contact",
  "site_id": "example",
  "csrf": "session-csrf-token",
  "page_item": "contact-form",
  "page_data": {
    "form_id": 1
  }
}
```

Wire requests include a session `csrf` token. Include it in extension-rendered forms (`_csrf` field or `X-CSRF-Token` header) for mutating browser requests; Cannon validates before proxying to data/endpoint routes.

The admin Routes UI stores page field values as route metadata and passes them back to the extension as `page_data`.

Data is for automatic public action URLs. Cannon always proxies `/ext/{route_hash}/{path}` to `POST /data/{path}` on the extension socket without requiring an admin route. `{route_hash}` is stable per extension + site (same as the socket filename without `.sock`); `GET /meta` returns it as `route_hash`.

Register paths with `HandleData("contact/submit", fn)`. Build form actions with `PublicDataURL(extensionName, siteID, "contact/submit")`. Wire requests include `data_path` (for example `contact/submit`).

Endpoint is for data/action routes at **admin-configured friendly paths** such as `/contact/submit`. Unlike page routes, Cannon writes the extension response directly to the browser without wrapping it in the site layout. Prefer `data` for extension-owned form posts that do not need a custom public path.

GET /endpoint lists endpoint definitions:

```json
{
  "endpoints": [
    {
      "id": "submit",
      "title": "Submit Contact Form"
    }
  ]
}
```

POST /endpoint/:item handles one endpoint. Cannon sends the normal wire request plus endpoint route context:

```json
{
  "method": "POST",
  "url": "/contact/submit",
  "body": "name=Jane&email=jane%40example.com",
  "site_id": "example",
  "endpoint_item": "submit",
  "endpoint_data": {}
}
```

The admin Routes UI can create **Extension Endpoint** routes. Endpoint metadata fields are stored on the route and passed back to the extension as `endpoint_data`. Endpoint handlers may return redirects, JSON, plain text, HTML, or any status/header/body combination.

Block is similar to page except as part of a block rendering callback blocks need more fleshing out but for now they are various tags in the template will render a block position if there is a extension connected to that position rendering is passed similar to Page.  calls to /block lists blocks calls to /block/:item will render a block of that type form /block definiton.

In the example above request is sent to /request_handler, likewise for page and block.

If a extension has a "admin" capability it should register a entry on Extensions to manage the extension and route requests to the configured admin handler endpoint for rendering in the admin area.  

If an extension embeds templates, it should expose a `templates` capability. The Go extension package does this automatically when `EmbedTemplates` is called.

`GET /templates` lists embedded HTML templates and the site-relative path Cannon should write when creating an override:

```json
{
  "templates": [
    {
      "path": "contact/form.html",
      "override_path": "extension/contact/form.html",
      "size": 1234
    }
  ]
}
```

`GET /templates/{path}` returns the embedded default source for one template:

```json
{
  "path": "contact/form.html",
  "override_path": "extension/contact/form.html",
  "content": "<form>...</form>"
}
```

Template override files live under `{template_dir}/extension/...`. For example an embedded local template named `calendar/page.html` is overridden by `{template_dir}/extension/calendar/page.html`. Extension template paths should be namespaced to the extension, such as `contact/form.html`, so multiple extensions do not collide.

## Captcha capability

Extensions that protect forms with a third-party captcha provider (Cloudflare Turnstile, hCaptcha, reCAPTCHA, etc.) should expose a dedicated `captcha` capability instead of folding the logic into `request`, `hooks`, or `data`.

**Why a separate capability**

- Captcha has a fixed contract: render a widget, verify a token, expose public client config.
- Cannon integrates captcha in multiple places (login, comments, registration, extension forms) from one provider.
- Hooks and request middleware are the wrong abstraction for HTML widget rendering and provider-specific verification.

**One active provider per site**

Sites may install several captcha extensions, but Cannon calls **only the configured active captcha extension** for render and verify. Other extensions that expose `captcha` remain idle until selected in site configuration. Do not run multiple captcha providers on the same form.

If the active captcha extension is stopped or verification fails, Cannon should **fail closed** and reject the protected action.

### Form placement with `<captcha>` placeholders

Authors choose **where** captcha appears by placing a literal placeholder in any form HTML:

```html
<captcha context="login" provider="any"></captcha>
<captcha context="comment" provider="cloudflare"></captcha>
```

Template helper (equivalent):

```html
{{captcha "login"}}
{{captcha "comment" "any"}}
```

Attributes:

| Attribute | Required | Meaning |
|-----------|----------|---------|
| `context` | yes | Protected form context: `login`, `register`, `comment`, or `form` |
| `provider` | no | `any` (default) uses the site's active captcha extension; otherwise an extension name or provider alias such as `cloudflare`, `turnstile`, or `recaptcha` |
| `type` | no | Legacy alias for `provider` |

**Cannon expands placeholders — extensions do not scan HTML via `onAfterRender`.**

After the layout renders, Cannon:

1. Reads **Configuration → General → Captcha** (`captcha_enabled`, `captcha_active_extension`, `captcha_skip_authenticated`).
2. If captcha is disabled, removes all placeholders.
3. If the viewer is signed in and **Skip Captcha For Signed-In Users** is enabled, removes placeholders (no widget, no verify on submit).
4. For each remaining placeholder, resolves `provider` (`any` → active extension) and calls `POST /captcha/render` on that extension.
5. Replaces the tag with returned `html`, injects `head_html` before `</head>` once per page, and records `field_name` + `context` for submit verification.
6. Fires `onAfterRender` on the final body so hooks may adjust output — captcha extensions should **not** rely on this for primary widget injection.

Extension-generated forms (contact extension, etc.) may include the same literal `<captcha context="form" provider="any"></captcha>` string in their HTML output.

On mutating form submit, Cannon verifies only when captcha applied to that request: it reads the token from the rendered `field_name` (stored in session during render) and calls `POST /captcha/verify` on the same extension. Protected flows wired today: frontend login, admin login, item comments, and extension `/data` form submissions proxied through `/ext/{route_hash}/...`.

Cannon implementation notes:

- Placeholder expansion runs after layout render and in `RenderFragment` (for login blocks).
- `head_html` from render is injected before `</head>`.
- Each expanded widget includes `<input type="hidden" name="captcha_context" value="…">`.
- The token field name returned by render is stored in session under `captcha:field:{context}` for verify on submit.

Register on the extension server with `RegisterCaptcha`. Default socket paths:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/captcha` | GET | Provider metadata and public client configuration |
| `/captcha/render` | POST | HTML/head fragment for one protected context |
| `/captcha/verify` | POST | Validate a submitted captcha token |

**GET /captcha**

```json
{
  "id": "cannon-extension-turnstile",
  "title": "Cloudflare Turnstile",
  "contexts": ["login", "register", "comment", "form"],
  "client": {
    "site_key": "0x4AAAAAAA..."
  }
}
```

- `id` — stable provider identifier (usually the extension name).
- `contexts` — form contexts this provider supports (`login`, `register`, `comment`, `form`).
- `client` — public values only. Never return secret keys.

**POST /captcha/render**

Cannon sends the normal wire request plus captcha context:

```json
{
  "method": "GET",
  "url": "/login",
  "site_id": "example",
  "csrf": "session-csrf-token",
  "captcha_context": "login",
  "captcha_action": "login"
}
```

Respond with:

```json
{
  "html": "<div class=\"cf-turnstile\" data-sitekey=\"...\"></div>",
  "head_html": "<script src=\"https://challenges.cloudflare.com/turnstile/v0/api.js\" async defer></script>",
  "field_name": "cf-turnstile-response"
}
```

- `html` — widget markup inserted into the protected form.
- `head_html` — optional scripts/styles Cannon may inject into the page head once per request.
- `field_name` — form field Cannon reads on submit and forwards to verify.

**POST /captcha/verify**

```json
{
  "method": "POST",
  "url": "/login",
  "site_id": "example",
  "captcha_context": "login",
  "captcha_token": "token-from-client",
  "captcha_remote_ip": "203.0.113.10"
}
```

Respond with HTTP `200` when valid:

```json
{"valid": true}
```

Respond with HTTP `403` when invalid:

```json
{"valid": false, "error": "captcha verification failed"}
```

Verification happens server-to-server between Cannon and the extension. The extension calls the provider API using secrets from its own `/configuration` store.


Another built in capability should be /help which provides help docs to be included in the parents /admin/help area.  /help should respond with the list of help articles the extension provides ie 

help:{
 "/help/how-to-config",
 "/help/overview"
}

Each entry when requested should respond as markdown to be rendered and included a side bar menu should be generated out of teh help list by the order of entry.

/meta requests should return meta info to be shown in teh admin extensions area ie

{
  "name": "cannon-extension-contact",
  "version": "0.1.0",
  "update_url_base": "https://github.com/rob121/cannon-extension-contact/releases/download"
}



/install should be posted to the first time the exteension is installed - the etension metadata in the db should show if it's been installed or not (bool installed) 

/install should do things like copy files, add configurations, create tables etc - any one time install related task.  

/install should also support a system configuration via /configuration - which uses https://jsonforms.io defined forms to allow update of the values the extension needs

this library will render the forms: https://github.com/TobiEiss/go-jsonforms

### Configuration field types

Cannon extends plain JSON Forms with reusable widgets for admin configuration (global sections and extension `/configuration`):

| Kind | How to declare | Notes |
|------|----------------|-------|
| Boolean | `"type": "boolean"` | Rendered as an enable/disable toggle; unchecked fields save as `false` |
| Enum select | `"enum": [...]` on `string` or `integer` | Standard dropdown |
| Textarea | UI `"options": {"multi": true}` | Multi-line text for long strings |
| Category | `"format": "category"` on an integer property and/or UI `"options": {"format": "category"}` | Dropdown of active site categories by ID; supports `"type": ["integer", "null"]` |
| Dynamic enum | Patched at render (e.g. theme names, captcha extensions) | Schema ships with a placeholder `enum`; Cannon fills options from site data |

Global section JSON lives in `internal/settings/definitions/*.json`. Extensions return the same `schema` / `ui_schema` / `data` shape from `GET /configuration`.

See `internal/help/docs/admin/configuration-fields.md` and `EXTENSIONS.md` for examples.


## Databses

With the excelption of sqlite extensions can call the database to perform read/write ops - for sqlite lwe need to take care that we are using wal logging for connection handling.


# SQLite Access For Extensions

Cannon extensions may need site database access. The CMS passes `--site=$id` and `--socket=$location` today

## Goal

Allow extensions to safely read/write the current site database when needed, including SQLite, MySQL, and PostgreSQL.

## Site DB Context

When starting an extension, the extension can read teh sites.json and determine from the context of the arguments provided what db connection is necessary

SQLite Requirements
SQLite can be opened by both the parent Cannon process and extension processes, but it only supports one writer at a time. Configure SQLite consistently in both Cannon and extensions.
For SQLite DSNs, use WAL mode, busy timeout, and foreign keys:
dsn := "file:" + path + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
After opening GORM for SQLite, limit the connection pool:
sqlDB, err := db.DB()
if err == nil {
    sqlDB.SetMaxOpenConns(1)
}
Implementation Notes
Update Cannon’s SQLite database open logic in internal/database/database.go so SQLite always uses:
_journal_mode=WAL
_busy_timeout=5000
_foreign_keys=on
SetMaxOpenConns(1)
Extensions that open the SQLite DB directly must use the same settings.
Extension Table Ownership
Extensions should own their own tables and avoid mutating Cannon core tables unless explicitly designed.
Example contact extension tables:
contact_forms
contact_submissions
Keep extension writes short and transactional.
Guidance
Direct extension DB access is acceptable for small extension-owned data, such as contact forms and submissions.
For write-heavy workflows or operations that modify core CMS data, prefer a CMS-owned API or request/response contract so the Cannon parent process remains the primary writer.
```
