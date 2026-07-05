# Extension Authoring Guide

Cannon extensions are separate processes that speak HTTP over a Unix domain socket. The `github.com/rob121/cannon/extension` package handles socket setup, built-in endpoints, and capability registration so you can focus on your extension logic.

This guide covers how to build an extension, which capabilities to implement, and how Cannon wires them into the CMS.

## Project layout

```
my-extension/
  main.go
  go.mod
  templates/          # optional — for EmbedTemplates
  help/               # optional — markdown for EmbedHelp
```

Minimal entry point:

```go
package main

import "github.com/rob121/cannon/extension"

func main() {
    s := extension.New(extension.Info{
        Name:    "my-extension",
        Version: "1.0.0",
        Title:   "My Extension",
    })
    // Register capabilities here…
    s.ListenAndServe()
}
```

Cannon starts each active extension with `--site=<id>` and `--socket=<path>`, sets `CANNON_CONFIG` to the path of `sites.json`, and calls `GET /capabilities` to learn what the extension supports. On first activation, Cannon sends `POST /install` once for migrations, default configuration, or file setup.

Activate extensions under **System → Extensions**, then bind pages, blocks, or endpoints under **System → Routes** as needed.

## Built-in endpoints

The `extension` package registers these automatically:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/capabilities` | GET | Advertise supported capabilities |
| `/meta` | GET | Name, version, `route_hash`, update URL |
| `/install` | POST | One-time setup (`OnInstall`) |
| `/health` | GET | Health check |
| `/help` | GET | List of help article paths |
| `/help/{slug}` | GET | Markdown help article |

Register only the capabilities your extension implements. Cannon ignores missing handlers.

## Capabilities

| Capability | Register with | When Cannon calls it |
|------------|---------------|----------------------|
| Request | `HandleRequest` | Early in the HTTP pipeline; modify or block requests |
| Pages | `RegisterPage` / `HandlePage` | Public routes rendered inside site layouts |
| Data | `HandleData` / `OnData` | Automatic URLs at `/ext/{route_hash}/…` |
| Endpoints | `RegisterEndpoint` / `HandleEndpoint` | Admin-configured friendly paths; raw HTTP response |
| Blocks | `RegisterBlock` / `HandleBlock` | Template spaces (`{{space "footer"}}`) |
| Admin | `HandleAdmin` | UI under **Extensions** at `/admin/extension-apps/{name}` |
| Hooks | `OnHook` | Lifecycle events (content, rendering, search, mail, SEO) |
| Configuration | `OnConfiguration` | JSON Forms settings per site |
| Help | `EmbedHelp` | Markdown articles in **Help → Extensions** |
| Templates | `EmbedTemplates` | Embeddable HTML overridable from the site template dir |
| Captcha | `RegisterCaptcha` | Render/verify widgets; Cannon expands `<captcha>` tags |
| Install | `OnInstall` | One-time setup on first activation |
| Permissions | `RegisterPermissions` | Capabilities synced into the role permission catalog |

You do not need every capability.

## Wire requests

Cannon POSTs JSON to capability handlers. Every wire request includes:

| Field | Meaning |
|-------|---------|
| `method`, `url`, `header`, `body` | Original browser request |
| `site_id` | Current site |
| `csrf` | Session CSRF token for mutating forms |
| `user` | Signed-in user scope (`authenticated`, `username`, `permissions`, …) |

Return values with helpers:

```go
extension.HTML(200, "<p>Hello</p>")
extension.Redirect(http.StatusSeeOther, "/done")
extension.Error(http.StatusBadRequest, "invalid input")
extension.OK()
```

### CSRF

Include the session token in extension-rendered forms that POST through Cannon:

```go
form := `<form method="post" action="` + action + `">` +
    extension.CSRFHiddenField(req) +
    `…</form>`
```

Cannon validates CSRF on mutating browser requests before proxying to data or endpoint routes. Clients may also send `X-CSRF-Token`.

### Permissions

Register permissions at startup. Cannon prefixes them with your extension name unless the id already includes it (`my-extension.manage`).

```go
s.RegisterPermissions([]extension.PermissionDef{
    {ID: "manage", DisplayName: "Manage Extension"},
})

s.HandleAdmin("/admin", func(req extension.WireRequest) extension.WireResponse {
    if !extension.UserCan(req, "my-extension.manage") {
        return extension.Error(http.StatusForbidden, "forbidden")
    }
    return extension.HTML(200, "<p>Admin</p>")
})
```

Signed-in requests include `user.permissions` and, when present, `user.denied_permissions`. `UserCan` honors explicit denies over allows and supports wildcards (`*`, `my-extension.*`).

## Pages

Extension **Page** routes render HTML inside the site layout. Expose page types with `RegisterPage`:

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
    return extension.HTML(200, "<div>Contact page</div>")
})
```

Site admins create an **Extension** route under **System → Routes**, pick your extension and page id, and fill metadata fields. Cannon stores field values on the route and passes them as `page_data` on each request.

`HandlePage` is a shortcut that registers a single `default` page.

## Data routes (recommended for form posts)

Use **data** for automatic public action URLs. Cannon proxies:

```text
/ext/{route_hash}/{path}  →  POST /data/{path}
```

- `{route_hash}` — stable per extension + site (from `GET /meta` or `extension.RouteHash`)
- `{path}` — the path you register with `HandleData`

No admin route is required. Build form actions with `extension.PublicDataURL`:

```go
action := extension.PublicDataURL(s.Info().Name, req.SiteID, "contact/submit")

s.HandleData("contact/submit", func(req extension.WireRequest) extension.WireResponse {
    if req.Method != http.MethodPost {
        return extension.Error(http.StatusMethodNotAllowed, "POST required")
    }
    // extension.DataPath(req) == "contact/submit"
    return extension.Redirect(http.StatusSeeOther, "/contact?sent=1")
})
```

Prefer **data** for extension-owned form posts. Use **endpoints** when you need a custom friendly URL configured in admin.

## Endpoints

**Endpoint** routes write the extension response directly to the browser (no site layout). Site admins create **Extension Endpoint** routes with friendly paths such as `/contact/submit`.

```go
s.RegisterEndpoint(extension.EndpointDefinition{
    ID:    "submit",
    Title: "Submit Contact Form",
}, func(item string, req extension.WireRequest) extension.WireResponse {
    return extension.Redirect(http.StatusSeeOther, "/contact?sent=1")
})
```

Cannon passes `endpoint_item` and optional `endpoint_data` from route metadata, same pattern as pages.

## Blocks

Blocks fill template spaces such as `{{space "sidebar"}}`. Register block types and allowed spaces:

```go
s.RegisterBlock(extension.BlockDefinition{
    ID:     "greeting",
    Title:  "Greeting",
    Spaces: []string{"footer", "sidebar"},
}, func(item string, req extension.WireRequest) extension.WireResponse {
    return extension.HTML(200, "<p>Hello from a block.</p>")
})
```

Create a block of type **Extension** in **System → Blocks**, choose your extension and block id, and assign it to a space. Wire requests include `block_space`, `block_item`, and optional `block_data` from block metadata (same pattern as pages).

`HandleBlock` registers a single `default` block for simple extensions.

## Request middleware

`HandleRequest` runs early in Cannon's HTTP pipeline. Return `updated_request` to modify the incoming request or `stop: true` with an error response to block the request. Use sparingly — prefer hooks or data routes when possible.

## Admin UI

`HandleAdmin` serves Turbo-framed HTML under `/admin/extension-apps/{extension-name}`. Return fragments that use Cannon admin classes (`admin-form-control`, `admin-data-card`, `btn-admin-primary`, etc.) for a consistent look.

The wire `url` field is the original admin URL (for example `/admin/extension-apps/my-extension/settings`), not the socket path.

## Configuration

`OnConfiguration` exposes JSON Schema and UI Schema (JSON Forms). Cannon renders settings under the extension's admin area and stores values per site.

```go
s.OnConfiguration(extension.ConfigurationProvider{
    Schema:   []byte(`{"type":"object","properties":{"api_key":{"type":"string"}}}`),
    UISchema: []byte(`{"type":"VerticalLayout","elements":[{"type":"Control","scope":"#/properties/api_key"}]}`),
})
```

### Configuration field types

| Kind | Declaration | Notes |
|------|-------------|-------|
| Boolean | `"type": "boolean"` | Toggle; unchecked saves as `false` |
| Enum | `"enum": [...]` on `string` or `integer` | Dropdown |
| Textarea | UI `"options": {"multi": true}` | Multi-line text |
| Category | `"format": "category"` and/or UI `"options": {"format": "category"}` | Active site categories by ID; use `"type": ["integer", "null"]` for optional |
| Dynamic enum | Patched at render time | e.g. captcha extensions, theme names |

See [Configuration fields](/admin/help/admin/configuration-fields) for global section examples. Extension configuration uses the same `schema` / `ui_schema` / `data` shape from `GET /configuration`.

## Templates

Call `EmbedTemplates(fsys, "templates")` to expose a `templates` capability. Cannon lists embedded templates in admin and can copy defaults into `{template_dir}/extension/…` for overrides.

Use namespaced paths (`contact/form.html`, not `form.html`) so multiple extensions do not collide.

## Hooks

Subscribe with `OnHook(event, fn)`. Cannon discovers hooks via `GET /hooks` and dispatches with `POST /hooks` including `event` and `arguments`.

```go
s.OnHook("onPrepareDocumentHead", func(req extension.HookWireRequest) extension.HookWireResponse {
    return extension.HookOK(map[string]any{
        "head_html": `<script src="https://example.com/analytics.js" defer></script>`,
    })
})

s.OnHook("onItemTrash", func(req extension.HookWireRequest) extension.HookWireResponse {
    // purge external cache using extension.HookArguments(req)
    return extension.HookOK(nil)
})
```

Respond with `extension.HookOK`, `extension.HookStop`, or `extension.HookAbort` (blocks login and similar guarded operations).

### Common events

| Area | Events |
|------|--------|
| Routing & render | `onBeforeRoute`, `onAfterRoute`, `onBeforeRender`, `onAfterRender` |
| Document head/body | `onPrepareDocumentHead`, `onPrepareDocumentBody` (frontend); `onAdminPrepareDocumentHead`, `onAdminPrepareDocumentBody`, `onAdminBeforeRender` (admin) |
| User | `onUserBeforeLogin`, `onUserAfterLogin`, `onUserLogout`, `onUserSignup`, `onUserVerified`, `onUserLocked` |
| Content | `onItemBeforeSave`, `onItemAfterSave`, `onItemTrash`, `onItemRestore`, `onItemBeforeDelete`, `onItemAfterDelete`, `onCategoryBeforeSave`, `onCategoryAfterSave`, `onCategoryBeforeDelete`, `onMediaUpload`, `onMediaDelete`, `onRevisionRestore` |
| Display | `onContentPrepare`, `onContentBeforeDisplay`, `onContentAfterDisplay`, `onItemBeforeRender`, `onRenderBlock`, `onAfterRenderBlock` |
| Search & SEO | `onBeforeSearch`, `onAfterSearch`, `onSitemapGenerate`, `onRobotsGenerate` |
| Mail & settings | `onBeforeMailSend`, `onSettingsSave` |
| Comments | `onCommentBeforeSave`, `onCommentAfterSave` |

### Fragment arguments

These keys are **appended** across listeners (not replaced):

| Key | Used by | Purpose |
|-----|---------|---------|
| `head_html` | Document head hooks | Scripts, styles, meta tags before `</head>` |
| `body_html` | Document body hooks | Deferred scripts before `</body>` |
| `robots_append` | `onRobotsGenerate` | Extra `robots.txt` lines |
| `sitemap_urls` | `onSitemapGenerate` | Extra entries: `[{"loc":"/path","lastmod":"2026-07-04"}]` |

Helpers: `extension.HookArguments`, `extension.HookHeadHTML`, `extension.HookBodyHTML`, `extension.HookRobotsAppend`, `extension.HookSitemapURLs`.

Static site-wide head markup is also available through **SEO & Robots → Additional head markup** in global configuration.

## Captcha

Sites may install multiple captcha extensions, but Cannon calls **only the active provider** selected in **Configuration → General → Captcha**.

Register with `RegisterCaptcha`. Implement `GET /captcha`, `POST /captcha/render`, and `POST /captcha/verify`.

Authors place widgets with a literal placeholder — extensions should **not** scan HTML via `onAfterRender`:

```html
<captcha context="login" provider="any"></captcha>
<captcha context="form" provider="cloudflare"></captcha>
```

In site templates you can use the equivalent helper:

```html
{{captcha "login"}}
{{captcha "comment" "any"}}
```

| Attribute | Required | Meaning |
|-----------|----------|---------|
| `context` | yes | `login`, `register`, `comment`, or `form` |
| `provider` | no | `any` (site default), extension name, or alias such as `cloudflare` |
| `type` | no | Legacy alias for `provider` |

Cannon expands placeholders after layout render, injects `head_html` before `</head>`, and verifies tokens on submit for login, comments, and `/ext/…` data routes. If captcha is disabled or skipped for signed-in users, placeholders are removed. If the active captcha extension is stopped or verification fails, Cannon **fails closed** and rejects the protected action.

## Help articles

Embed markdown with `EmbedHelp(fsys, "help")`. Articles appear under **Help → Extensions** when the extension is running.

```go
//go:embed help/*
var helpFS embed.FS

s.EmbedHelp(helpFS, "help")
```

List paths from `GET /help`; fetch markdown from `GET /help/{slug}`.

## Install handler

Run one-time setup with `OnInstall` — create tables, seed configuration, copy assets:

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

Cannon calls `POST /install` only while the extension's `installed` flag is false in the database.

## Database access

When Cannon starts an extension it sets `CANNON_CONFIG` and passes `--site`. Use:

```go
db, site, err := extension.OpenDB()
cfg, err := extension.SiteConfig()
siteID := extension.SiteID()
```

SQLite connections use WAL mode, busy timeout, foreign keys, and `MaxOpenConns(1)` — the same rules as the CMS.

### Table namespacing

Extensions share the site database with Cannon. **Prefix every extension-owned table** so names stay unique. Do not write to Cannon core tables (`users`, `routes`, `extensions`, etc.) unless you are deliberately integrating with core data.

```go
info := extension.Info{Name: "cannon-extension-contact"}

formsTable := extension.TableName(info.TablePrefix(), "forms")
// => "contact_forms"
```

The helpers strip a `cannon-extension-` prefix from the binary name automatically. Create tables in `OnInstall`, keep transactions short (especially on SQLite), and prefer Cannon-owned wire contracts for write-heavy or core CMS workflows.

## Testing locally

1. `go build -o my-extension .`
2. Place the binary in the extensions directory configured in `sites.json`, or point Cannon at your built socket path.
3. Activate under **System → Extensions** and restart the site process if needed.
4. Bind routes/blocks under **System → Routes** and **System → Blocks**.
5. Check **Help → Extensions** if you embedded help docs.

## Further reading

The repository root `EXTENSIONS.md` file is the full wire protocol reference (request/response shapes, captcha contract, hook dispatch details, and configuration examples). Use it alongside this guide when implementing non-trivial extensions.
