# Cannon Extensions

Cannon extensions are separate processes that communicate with the CMS over HTTP on a Unix domain socket. Each extension receives `--site=<id>` and `--socket=<path>` when Cannon starts it, listens on that socket, and implements the wire protocol described below.

The `github.com/rob121/cannon/extension` package handles socket setup, built-in endpoints, and capability registration so you can focus on your extension logic.

## How Cannon uses extensions

1. Cannon discovers binaries in the configured extensions directory and records them in the database.
2. On bootstrap, Cannon starts each active extension with `--site` and `--socket`, and sets `CANNON_CONFIG` to the path of `sites.json`.
3. Cannon calls `GET /capabilities` to learn what the extension supports (request middleware, page rendering, admin UI, help, etc.).
4. On first start, if the extension is not marked installed, Cannon sends `POST /install` with a wire request payload.
5. Cannon calls `GET /meta` to populate version and update information in the admin UI.
6. During requests, Cannon POSTs JSON wire requests to capability handlers (for example `/page`, `/admin`, `/request`).

## Required and built-in endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/capabilities` | GET | Advertise supported capabilities and admin defaults |
| `/meta` | GET | Name, version, update URL base |
| `/install` | POST | One-time setup (migrations, config, files) |
| `/health` | GET | Optional health check (not used by CMS today) |
| `/help` | GET | JSON list of help article paths |
| `/help/{slug}` | GET | Markdown help article |

The `extension` package registers all of these automatically. You only register the capabilities your extension implements.

## Capabilities

Capabilities map a role to an HTTP path on the extension socket:

| Capability | When Cannon calls it |
|------------|----------------------|
| `request` | Early in the HTTP pipeline; can modify or block requests |
| `page` | When a route is bound to this extension for public page rendering |
| `data` | Automatically for public URLs under `/ext/{route_hash}/…` (no admin route required) |
| `endpoint` | When an admin **Extension Endpoint** route is bound for passthrough HTTP responses |
| `block` | When a template space is bound to this extension |
| `admin` | When the extension admin UI is opened under `/admin/extension-apps/{name}` |
| `help` | Help articles aggregated into `/admin/help` |
| `templates` | Lists embedded extension templates that can be copied into site overrides |

Register handlers with `HandleRequest`, `HandlePage`, `RegisterPage`, `HandleData`, `RegisterEndpoint`, `RegisterBlock`, `HandleAdmin`, `OnHook`, and `OnConfiguration`. Paths default to `/request`, `/page`, `/data`, `/endpoint`, `/block`, `/hooks`, and `/admin` but can be customized.

### Template overrides

Extensions that call `EmbedTemplates(fsys, "templates")` automatically expose a `templates` capability at `/templates`. Cannon uses it to show embedded extension templates in the admin and copy default template source into the site's override folder.

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/templates` | GET | List embedded HTML templates |
| `/templates/{path}` | GET | Return embedded default source for one template |

**GET /templates**

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

**GET /templates/contact/form.html**

```json
{
  "path": "contact/form.html",
  "override_path": "extension/contact/form.html",
  "content": "<form>...</form>"
}
```

- `path` is the extension-local embedded template path.
- `override_path` is the site-template-relative path Cannon writes when the admin chooses Override.
- Override files live under `{template_dir}/extension/...`; for example `contact/form.html` is overridden by `{template_dir}/extension/contact/form.html`.
- Use namespaced local template paths such as `contact/form.html` or `calendar/page.html` to avoid collisions with other extensions.

### Pages

Extension routes use the `page` capability. Extensions expose:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/page` | GET | List page definitions for this extension |
| `/page/{id}` | POST | Render one page with a wire request |

**GET /page**

```json
{
  "pages": [
    {
      "id": "contact",
      "title": "Contact Page",
      "fields": [
        {
          "name": "form_id",
          "label": "Form",
          "type": "select",
          "required": true,
          "options": [{"value": "1", "label": "General inquiry"}]
        }
      ]
    }
  ]
}
```

- `id` — page type rendered at `POST /page/{id}`
- `fields` — optional admin metadata fields stored on the route and sent as `page_data`

**POST /page/{id}** uses the normal wire request plus page context:

```json
{
  "method": "GET",
  "url": "/contact",
  "site_id": "example",
  "page_item": "contact",
  "page_data": {"form_id": 42},
  "user": {"authenticated": false}
}
```

Register pages on the extension server:

```go
s.RegisterPage(extension.PageDefinition{
    ID:    "contact",
    Title: "Contact Page",
    Fields: []extension.PageField{
        {Name: "form_id", Label: "Form", Type: "select", Required: true},
    },
}, func(item string, req extension.WireRequest) extension.WireResponse {
    _ = extension.PageItem(req) // "contact"
    if formID, ok := extension.PageDataInt(req, "form_id"); ok {
        _ = formID
    }
    return extension.HTML(http.StatusOK, "<div>Contact page</div>")
})
```

`HandlePage(path, fn)` remains available for simple extensions: it registers one `default` page at `POST /page/default`.

Cannon loads page definitions when an extension starts. Site admins pick an extension page under **System → Routes** for Extension-type routes; metadata is stored on the route and passed as `page_data`.

### Data routes (`/ext/{route_hash}/…`)

Use the `data` capability for **automatic public data URLs**. Cannon always proxies:

```text
/ext/{route_hash}/{path}  →  POST /data/{path} on the extension socket
```

- `{route_hash}` — stable per extension + site (same value as the socket filename without `.sock`). Available from `GET /meta` as `route_hash`.
- `{path}` — the route you register with `HandleData`, for example `contact/submit`.

No admin route is required. This is the recommended way to wire form posts and other action URLs from extension page HTML.

| Socket | Method | Purpose |
|--------|--------|---------|
| `/data/{path}` | POST | Handle one path-based data route |

**POST /data/contact/submit** receives the normal wire request plus:

```json
{
  "method": "POST",
  "url": "/ext/a1b2…/contact/submit",
  "body": "name=Jane&email=jane%40example.com",
  "site_id": "example",
  "data_path": "contact/submit",
  "user": {"authenticated": false}
}
```

Register data handlers on the extension server:

```go
submitURL := extension.PublicDataURL(s.Info().Name, req.SiteID, "contact/submit")

s.RegisterPage(extension.PageDefinition{
    ID:    "contact",
    Title: "Contact Page",
}, func(item string, req extension.WireRequest) extension.WireResponse {
    action := extension.PublicDataURL(s.Info().Name, req.SiteID, "contact/submit")
    form := fmt.Sprintf(`<form method="post" action="%s">...</form>`, html.EscapeString(action))
    return extension.HTML(http.StatusOK, form)
})

s.HandleData("contact/submit", func(req extension.WireRequest) extension.WireResponse {
    if req.Method != http.MethodPost {
        return extension.Error(http.StatusMethodNotAllowed, "POST required")
    }
    // extension.DataPath(req) == "contact/submit"
    return extension.Redirect(http.StatusSeeOther, "/contact?sent=1")
})
```

Helpers:

```go
hash := extension.RouteHash("cannon-extension-contact", "example")
url := extension.PublicDataURL("cannon-extension-contact", "example", "contact/submit")
path := extension.DataPath(req) // "contact/submit"
```

Use `OnData(fn)` when you prefer to route manually inside one fallback handler.

### Endpoints (admin routes)

Use the `endpoint` capability when you want a **friendly public path** configured in **Admin → Routes** (for example `/contact/submit` instead of `/ext/{route_hash}/contact/submit`). Cannon passes the extension response through to the browser **without** wrapping it in the site layout.

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/endpoint` | GET | List endpoint definitions for this extension |
| `/endpoint/{id}` | POST | Handle one endpoint with a wire request |

**GET /endpoint**

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

- `id` — endpoint type handled at `POST /endpoint/{id}`
- `fields` — optional admin metadata fields stored on the route and sent as `endpoint_data`

**POST /endpoint/{id}** uses the normal wire request plus endpoint context:

```json
{
  "method": "POST",
  "url": "/contact/submit",
  "header": {"Content-Type": ["application/x-www-form-urlencoded"]},
  "body": "name=Jane&email=jane%40example.com",
  "site_id": "example",
  "endpoint_item": "submit",
  "endpoint_data": {},
  "user": {"authenticated": false}
}
```

Register endpoints on the extension server when using admin **Extension Endpoint** routes:

```go
s.RegisterEndpoint(extension.EndpointDefinition{
    ID:    "submit",
    Title: "Submit Contact Form",
}, func(item string, req extension.WireRequest) extension.WireResponse {
    if req.Method != http.MethodPost {
        return extension.Error(http.StatusMethodNotAllowed, "POST required")
    }
    return extension.Redirect(http.StatusSeeOther, "/contact?sent=1")
})
```

Optional **Admin → Routes** entries for friendly URLs:

| Path | Type | Extension | Handler |
|------|------|-----------|---------|
| `/contact` | Extension | `cannon-extension-contact` | Page `contact` |
| `/contact/submit` | Extension Endpoint | `cannon-extension-contact` | Endpoint `submit` |

For most extensions, prefer **`HandleData` + `/ext/{route_hash}/…`** so submit URLs work without creating admin routes. Use **Extension Endpoint** routes when you need a custom public path.

`HandleEndpoint(path, fn)` registers a single `default` endpoint for simple extensions.

Cannon loads endpoint definitions when an extension starts. Site admins pick an extension endpoint under **System → Routes** for **Extension Endpoint** routes; metadata is stored on the route and passed as `endpoint_data`.

### Blocks

Template spaces such as `{{space "footer"}}` render discrete content areas. Extensions with the `block` capability expose:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/block` | GET | List block definitions for this extension |
| `/block/{id}` | POST | Render one block with a wire request |

**GET /block**

```json
{
  "blocks": [
    {
      "id": "contact-form",
      "title": "Contact Form",
      "spaces": ["footer", "sidebar"]
    }
  ]
}
```

- `id` — block type rendered at `POST /block/{id}`
- `spaces` — template spaces this block may fill (for example `footer` for `{{space "footer"}}`)
- omit `spaces` or use an empty list to allow the block in any space

**POST /block/{id}** uses the normal wire request plus block context:

```json
{
  "method": "GET",
  "url": "/about",
  "site_id": "example",
  "block_space": "footer",
  "block_item": "contact-form",
  "user": {"authenticated": false}
}
```

Register blocks on the extension server:

```go
s.RegisterBlock(extension.BlockDefinition{
    ID:     "contact-form",
    Title:  "Contact Form",
    Spaces: []string{"footer"},
}, func(item string, req extension.WireRequest) extension.WireResponse {
    space := extension.BlockSpace(req)
    _ = extension.BlockItem(req) // "contact-form"
    if formID, ok := extension.BlockDataInt(req, "form_id"); ok {
        _ = formID
    }
    return extension.HTML(http.StatusOK, "<div>Form in "+html.EscapeString(space)+"</div>")
})
```

`HandleBlock(path, fn)` remains available for simple extensions: it registers one `default` block that matches any space.

Cannon loads block definitions when an extension starts, matches the template space to a block id, then POSTs to `/block/{id}`.

## Wire protocol

Capability handlers receive a POST with a JSON body:

```json
{
  "method": "GET",
  "url": "/contact",
  "header": {"Accept": ["text/html"]},
  "body": "",
  "user": {
    "authenticated": true,
    "user_id": 1,
    "username": "jane",
    "email": "jane@example.com",
    "given_name": "Jane",
    "family_name": "Doe"
  },
  "csrf": "64-char-hex-session-token",
  "site_id": "default",
  "page_item": "contact",
  "page_data": {"form_id": 42},
  "endpoint_item": "submit",
  "endpoint_data": {},
  "data_path": "contact/submit",
  "block_space": "footer",
  "block_item": "contact-form",
  "block_data": {"form_id": 42}
}
```

Page handlers receive `page_item` (definition id) and optional `page_data` route metadata from the admin Routes UI (for example `form_id`).

Endpoint handlers receive `endpoint_item` (definition id) and optional `endpoint_data` route metadata from **Extension Endpoint** routes. Return `extension.Redirect`, `extension.HTML`, or a custom `WireResponse` with any status code and headers — Cannon writes the response directly to the client.

Data handlers receive `data_path` (for example `contact/submit`) for requests proxied from `/ext/{route_hash}/…`. Use `extension.DataPath(req)` in the handler.

Block handlers also receive `block_space` (template space), `block_item` (definition id), and optional `block_data` placement metadata from the admin Blocks UI (for example `form_id`).

Site admins manage block placements under **System → Blocks**. Each row assigns a native HTML/Markdown block or an extension block to a template space used by `{{space "space"}}`. Only assigned blocks render; extensions are not invoked automatically when a space has no placement.

### User scope

Cannon attaches the current session user to every capability call in the `user` field of the wire request. This is populated from the signed-in Cannon user when a session exists.

| Field | Type | Description |
|-------|------|-------------|
| `authenticated` | bool | `true` when a user is signed in |
| `user_id` | number | Cannon user id |
| `username` | string | Login username |
| `email` | string | User email |
| `given_name` | string | First name |
| `family_name` | string | Last name |

When no user is signed in, Cannon sends `{"authenticated": false}`. Public page and block handlers may see this for anonymous visitors. Admin capability calls are made from the authenticated admin area, so `authenticated` is typically `true` and name fields are available.

Use the helpers in the extension package:

```go
if !extension.UserAuthenticated(req) {
    return extension.Error(http.StatusForbidden, "sign in required")
}
name := extension.UserDisplayName(req) // "Jane Doe", or username, or email
email := extension.UserString(req, "email")
userID, ok := extension.UserID(req)
```

### CSRF protection

Cannon validates CSRF tokens on mutating browser requests (`POST`, `PUT`, `PATCH`, `DELETE`) before admin handlers, controller actions, or extension data/endpoint routes run. Each session receives a token on safe requests; forms must include it as a hidden field named `_csrf`, or clients may send the `X-CSRF-Token` header.

Every capability wire request includes the current session token in the top-level `csrf` field. Use it when rendering HTML forms from extensions:

```go
formAction := extension.PublicDataURL("my-extension", req.SiteID, "contact/submit")
body := fmt.Sprintf(`<form method="post" action="%s">%s<input name="email"></form>`,
    html.EscapeString(formAction),
    extension.CSRFHiddenField(req),
)
```

Helpers:

```go
token := extension.CSRFToken(req)
field := extension.CSRFHiddenField(req) // `<input type="hidden" name="_csrf" ...>`
```

Cannon validates the token on incoming browser POSTs to `/ext/{route_hash}/…` and extension endpoint routes before forwarding to your handler. Extension handlers do not need to re-validate unless they accept requests from other origins directly.

### Event hooks

Cannon dispatches named event hooks during routing, rendering, login/logout, blocks, and content display. See `.info/specs/event_hooks.md` for the full event list and argument reference.

Extensions subscribe with `OnHook`:

```go
s.OnHook("onUserBeforeLogin", func(req extension.HookWireRequest) extension.HookWireResponse {
    username, _ := extension.HookArguments(req)["username"].(string)
    if username == "blocked" {
        return extension.HookAbort("account disabled")
    }
    return extension.HookOK(nil)
})
```

Cannon reads subscriptions from `GET /hooks`:

```json
{"hooks": ["onUserBeforeLogin"]}
```

Hook dispatches use `POST /hooks` with the normal wire fields plus `event` and `arguments`:

```json
{
  "event": "onBeforeRoute",
  "arguments": {"method": "GET", "path": "/contact"},
  "method": "GET",
  "url": "/contact",
  "user": {"authenticated": false},
  "site_id": "example"
}
```

Respond with optional `arguments` updates and `stop: true` to halt further listeners. Use `extension.HookOK`, `extension.HookStop`, and `extension.HookAbort`.

Core code can register in-process listeners with `github.com/rob121/cannon/internal/hooks`.Register and fire them with `hooks.Fire(ctx, event, args)` (context is wired per request by middleware).

`onAfterRender` is the final template-render hook. Cannon fires it after the page/layout has rendered and before writing bytes to the client. Arguments include `layout`, `page`, `headers`, and `body`. Returning `body` replaces the rendered text body. Returning `body_base64` with `body_encoding: "base64"` replaces the body with binary bytes, which lets hook extensions emit compressed payloads such as gzip. Returning `headers` merges those headers into the outgoing response.

#### Admin example: greet the signed-in user

When an admin opens `/admin/extension-apps/my-extension`, Cannon POSTs to your `/admin` handler with the admin user's scope in `req.User`:

```go
s.HandleAdmin("/admin", func(req extension.WireRequest) extension.WireResponse {
    if !extension.UserAuthenticated(req) {
        return extension.Error(http.StatusForbidden, "sign in required")
    }
    name := html.EscapeString(extension.UserDisplayName(req))
    body := fmt.Sprintf("<p>Signed in as <strong>%s</strong> (%s)</p>",
        name, html.EscapeString(extension.UserString(req, "email")))
    return extension.HTML(http.StatusOK, body)
})
```

The wire request `url` field is the original admin URL (for example `/admin/extension-apps/my-extension/settings`), not the extension socket path. Use `req.URL` for admin sub-routes if your handler serves multiple pages.

Respond with JSON:

```json
{
  "status_code": 200,
  "header": {"Content-Type": ["text/html; charset=utf-8"]},
  "body": "<h1>Hello</h1>",
  "stop": false
}
```

- `status_code` — HTTP status for the rendered result (or block response).
- `header` — Response headers for the output.
- `body` — Response body (HTML for page/admin, etc.).
- `updated_request` — Optional modified request for `request` capability chaining.
- `stop` — When true, Cannon stops processing and returns the response immediately.

Use helpers from the package:

```go
extension.HTML(200, "<h1>Hello</h1>")
extension.OK()
extension.Error(500, "something failed")
```

## Using the extension package

### Minimal example

```go
package main

import (
	"log"

	"github.com/rob121/cannon/extension"
)

func main() {
	s := extension.New(extension.Info{
		Name:          "my-extension",
		Version:       "0.1.0",
		UpdateURLBase: "https://github.com/you/my-extension/releases/download",
		AdminMenuName: "My Extension",
	})

	s.HandlePage("/page", func(req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "<h1>Hello</h1>")
	})

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### Help articles

Embed markdown files and register help:

```go
//go:embed help/*.md
var helpFiles embed.FS

s.EmbedHelp(helpFiles, "help")
```

Cannon lists articles from `GET /help` and fetches markdown from `/help/{slug}`.

### Install hook

Run migrations or copy files on first install:

```go
s.OnInstall(func(req extension.WireRequest) error {
	db, site, err := extension.OpenDB()
	if err != nil {
		return err
	}
	return db.AutoMigrate(&MyTable{})
})
```

If you omit `OnInstall`, the package returns success (suitable for extensions with no setup).

### Configuration (JSON Forms)

Extensions can expose settings through `GET` and `POST /configuration` using [JSON Forms](https://jsonforms.io/) schemas. Cannon renders these forms in **System → Configuration** and saves changes back to the extension.

Register a configuration provider on the server:

```go
info := extension.Info{Name: "cannon-extension-contact"}
settingsTable := extension.TableName(info.TablePrefix(), "settings")

s.OnConfiguration(extension.MapConfiguration([]extension.ConfigurationDefinition{
    {
        ID:    "general",
        Title: "General",
        Schema: json.RawMessage(`{
          "type": "object",
          "properties": {
            "recipient_email": {"type": "string", "title": "Notification Email", "format": "email"}
          }
        }`),
        UISchema: json.RawMessage(`{
          "type": "VerticalLayout",
          "elements": [{"type": "Control", "scope": "#/properties/recipient_email"}]
        }`),
    },
}, extension.DBConfigurationStore(db, settingsTable)))
```

`OnConfiguration` registers `/configuration` and advertises the `configuration` capability.

Wire format:

**GET /configuration**

```json
{
  "sections": [
    {
      "id": "general",
      "title": "General",
      "schema": {"type": "object", "properties": {...}},
      "ui_schema": {"type": "VerticalLayout", "elements": [...]},
      "data": {"recipient_email": "admin@example.com"}
    }
  ]
}
```

**POST /configuration**

```json
{
  "section": "general",
  "data": {"recipient_email": "admin@example.com"}
}
```

Use `extension.DBConfigurationStore` with a namespaced table such as `contact_settings`, and create it during install:

```go
s.OnInstall(func(req extension.WireRequest) error {
    db, _, err := extension.OpenDB()
    if err != nil {
        return err
    }
    return extension.MigrateConfigurationStore(db, settingsTable)
})
```

Cannon global settings use the same JSON Forms format via the `internal/settings` package. **Global section schemas are embedded in the Cannon binary** under `internal/settings/definitions/` and ship with the program. Admin-saved values are stored per site in the database `settings` table under scope `global`.

To add or change a global section, add a JSON file in the Cannon repo and rebuild:

```
internal/settings/definitions/mail.json
internal/settings/definitions/general.json
```

Each file contains `title`, `schema`, and `ui_schema`.

### Custom routes

When Cannon starts an extension it sets `CANNON_CONFIG` and passes `--site`. Use:

```go
db, site, err := extension.OpenDB()
cfg, err := extension.SiteConfig()
siteID := extension.SiteID()
```

SQLite connections use WAL mode, busy timeout, foreign keys, and `MaxOpenConns(1)` — the same rules as the CMS. Extensions should own their tables and keep writes short.

### Namespacing database tables

Extensions share the site database with Cannon. **Prefix every extension-owned table** so names stay unique and clearly owned. Do not write to Cannon core tables (`users`, `routes`, `extensions`, etc.) unless you are deliberately integrating with core data through a documented contract.

Use a short slug derived from the extension name. The helpers strip a `cannon-extension-` prefix automatically:

```go
info := extension.Info{Name: "cannon-extension-contact"}

formsTable := extension.TableName(info.TablePrefix(), "forms")
// => "contact_forms"

submissionsTable := extension.TableName(info.TablePrefix(), "submissions")
// => "contact_submissions"
```

Guidelines:

- Use lowercase identifiers with underscores only.
- Keep one logical prefix per extension (`contact_*`, `newsletter_*`, etc.).
- Create tables in `OnInstall` with `AutoMigrate` or explicit SQL.
- Keep transactions short, especially on SQLite where Cannon and extensions share the same file.

Example install hook:

```go
info := extension.Info{Name: "cannon-extension-contact"}

s.OnInstall(func(req extension.WireRequest) error {
    db, _, err := extension.OpenDB()
    if err != nil {
        return err
    }
    table := extension.TableName(info.TablePrefix(), "forms")
    return db.Table(table).AutoMigrate(&ContactForm{})
})
```

### Templates and overrides

Extensions often ship HTML templates embedded in the binary. Sites can override them by placing files in the Cannon template directory using the `extension/` prefix.

| Embedded in extension | Site override path (under `template_dir`) |
|-----------------------|-------------------------------------------|
| `contact/form.html` | `extension/contact/form.html` |
| `contact/admin/list.html` | `extension/contact/admin/list.html` |

Resolution order:

1. `{template_dir}/extension/{local-path}`
2. Embedded template from the extension binary

Register templates on the server:

```go
//go:embed all:templates
var templateFiles embed.FS

s.EmbedTemplates(templateFiles, "templates")
```

Render from a capability handler:

```go
s.HandlePage("/page", func(req extension.WireRequest) extension.WireResponse {
    body, err := s.Templates().Execute("contact/form.html", map[string]any{
        "Title": "Contact us",
    })
    if err != nil {
        return extension.Error(http.StatusInternalServerError, err.Error())
    }
    return extension.HTML(http.StatusOK, body)
})
```

Use a unique first path segment per extension (`contact/...`, `newsletter/...`) so overrides from different extensions do not collide under `extension/`.

Standalone use without the server:

```go
tpl := extension.NewTemplates(templateFiles, "templates")
body, err := tpl.Execute("contact/form.html", data)
```

For tests, pin the template directory explicitly:

```go
tpl := extension.NewTemplates(templateFiles, "templates").WithTemplateDir("/tmp/site-templates")
```

Override path helper:

```go
path, _ := extension.TemplateOverridePath("contact/form.html")
// => "extension/contact/form.html"
```

## Building and deploying

1. Create a Go module for your extension (separate repo or subdirectory).
2. Add a dependency on Cannon:

   ```bash
   go get github.com/rob121/cannon@latest
   ```

   For local development:

   ```go
   replace github.com/rob121/cannon => ../cannon
   ```

3. Build a binary named to match the extension (for example `cannon-extension-contact`).
4. Place the binary in Cannon's extensions directory (configured in `sites.json`).
5. Register the extension in the admin UI or let Cannon sync it from the directory.
6. Set status to active; Cannon starts the process and runs `/install` if needed.

## Testing

Use `Handler()` for unit tests without binding a socket:

```go
s := extension.New(extension.Info{Name: "test", Version: "1"})
s.HandlePage("/page", handler)
req := httptest.NewRequest(http.MethodPost, "/page/default", body)
rec := httptest.NewRecorder()
s.Handler().ServeHTTP(rec, req)
```

See `extension/server_test.go` and `cannon-extension-contact` for examples.

## Example: contact extension

The [cannon-extension-contact](https://github.com/rob121/cannon-extension-contact) repository demonstrates a page + admin + help extension using this package.

## Admin integration

- **Extensions list** — Shows process status, install state, capabilities from `/capabilities`, and meta from `/meta`.
- **Extension edit** — Install, start, stop, restart toolbar.
- **Help** — Aggregates `/help` entries from all extensions at `/admin/help`.
- **Admin menu** — Uses `defaults.admin.menu_name` from `/capabilities` when the extension record has no menu name yet.
- **Configuration** — `/admin/configuration` lists Global sections and extensions that expose `/configuration`. Forms are rendered with [go-jsonforms](https://github.com/TobiEiss/go-jsonforms).

## Reference

- Wire types: `extension/wire.go`
- Server: `extension/server.go`
- Configuration: `extension/configuration.go`
- Settings store: `internal/settings`
- Templates: `extension/templates.go`
- Table naming: `extension/tables.go`
- Spec notes: `.info/specs/extensions.md`, `.info/specs/settings.md`
