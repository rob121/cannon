# Frontend Templates

Cannon renders the public site with **Go `html/template`** files organized as **themes** — self-contained folders under your site's `template_dir`.

For **Cannon template functions** and **Sprig**, see **Help → Templates → Template Functions**.

---

## Themes

A theme is a folder like `{template_dir}/mysite/` containing:

- HTML templates (`layout.html`, `controllers/...`)
- Static assets in `assets/` (CSS, JS, images)
- `template.json` metadata

**Built-in frontend theme** (`default`) ships embedded in Cannon. To customize, create a custom theme and select it under **Configuration → General → Frontend Theme**.

### Create a theme

1. **Admin → Templates → New Theme**
2. Enter folder name (e.g. `mysite`), type (`frontend` or `full`), and metadata
3. Browse the theme, click **Override** on `layout.html` or other built-ins to copy source into your theme folder
4. Set **Frontend Theme** to `mysite` in **Configuration → General**

---

## Layout and page

Most public pages use:

| File (inside theme) | Role |
|---------------------|------|
| `layout.html` | Outer document — chrome, nav, `{{.Main}}` |
| `page.html` | Inner body for extension routes |
| `controllers/{controller}/{action}.html` | Controller page bodies |

Cannon still requests logical paths like `default/layout.html`; when your theme is active, files are loaded from `{theme}/layout.html` instead of the embed.

Built-in layout excerpt (include these hook placeholders in custom `layout.html` overrides):

```html
<head>
  ...
  <link href="/theme/site.css" rel="stylesheet">
  {{if .HeadExtra}}{{.HeadExtra}}{{end}}
</head>
<body>
  ...
  <main>{{.Main}}</main>
  <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"></script>
  {{if .BodyEndExtra}}{{.BodyEndExtra}}{{end}}
</body>
```

### Document hooks in layouts

Cannon merges extension and built-in markup into two layout fields:

| Field | Hook | Typical content |
|-------|------|-----------------|
| `{{.HeadExtra}}` | `onPrepareDocumentHead` | Extra meta, link, or script tags in `<head>` |
| `{{.BodyEndExtra}}` | `onPrepareDocumentBody` | Deferred scripts before `</body>` |

**Include both placeholders once** in `layout.html`. You do not add analytics, captcha, or extension scripts manually — Cannon injects them when enabled (for example **Configuration → Analytics → Enable Real-Time Analytics** loads `/theme/site-analytics.js` via `BodyEndExtra`).

SEO **Additional Head Markup** from Configuration → SEO uses the `{{siteHeadExtra}}` partial inside `meta-tags.html`, separate from `HeadExtra`.

---

## Theme assets

Place CSS, JS, fonts, and images in **`{theme}/assets/`**. They are served at **`/theme/{path}`** when the theme is active:

```
mysite/assets/site.css      →  /theme/site.css
mysite/assets/js/app.js     →  /theme/js/app.js
mysite/assets/logo.png      →  /theme/logo.png
```

When **Frontend Theme** is `default` (built-in), `/theme/site.css`, `/theme/site-mfa.js`, and `/theme/site-analytics.js` serve embedded default files. Custom themes override with on-disk files first (under `assets/` or the theme root for built-in parity).

Site media uploads remain under **`/assets/`** from `assets_dir` — separate from theme assets.

---

## Template lookup order

For logical path `default/controllers/content/item.html` with **Frontend Theme** = `mysite`:

1. `{template_dir}/mysite/controllers/content/item.html` (if theme active)
2. Built-in embedded template

With **Frontend Theme** = `default`, only the embedded copy is used unless you later add a custom theme.

Extension HTML overrides remain at `{template_dir}/extension/{name}/...`.

---

## Theme metadata (`template.json`)

Stored at **`{theme}/template.json`**, edited via **Admin → Templates → {theme} → Metadata**.

| Field | Description |
|-------|-------------|
| `name` | Display name |
| `type` | `frontend`, `backend`, or `full` |
| `status` | `active` or `inactive` — inactive themes cannot be selected |

Example:

```json
{
  "name": "Acme Public Theme",
  "author": "Acme Web Team",
  "type": "frontend",
  "status": "active",
  "version": "1.0.0"
}
```

---

## Admin theme

Admin UI templates work the same way with **Admin Theme** in Configuration → General. A theme with `type: backend` or `full` can be selected for admin. Admin assets live in `{theme}/assets/` and are served at `/admin/assets/`.

---

## Block spaces

Named regions in templates for extension blocks:

```html
{{space "footer"}}
{{if gt (lenspace "header") 0}}<div>{{space "header"}}</div>{{end}}
```

Enable **Debug Template Spaces** under Configuration → General, then append `?tp=1` to public URLs. The same `?tp=1` query also appends each `{{lang "key"}}` lookup key in parentheses (e.g. `Sign in (nav.sign_in)`) to help when editing locale files.

---

## Menus

```html
<ul class="nav">
  {{range menu "main"}}
  <li><a href="{{.Href}}">{{.Name}}</a></li>
  {{end}}
</ul>
```

Create the menu under **Admin → Menus** with matching **Menu name**.

---

## Extension routes

Extension page routes wrap handler HTML in `layout.html` + `page.html`. Data includes `.Title`, `.Content`, and `.Main` on the layout.

---

## Authoring tips

**Valid paths.** Template storage paths must end with `.html` and include the theme folder: `mysite/layout.html`.

**Auto-escaping.** `html/template` escapes `{{.Field}}` by default. Use trusted HTML only for extension output.

**Bootstrap.** Built-in styles use `/theme/site.css`. Override by placing `site.css` in your theme's `assets/` folder.

**Site name.** `{{siteName}}` and `{{year}}` are available in layouts.

**Category templates.** Set **Template override** on a category to a path like `mysite/controllers/content/custom-list.html`.

---

## Workflow summary

1. **Templates → New Theme** — create `mysite` with `assets/` and `template.json`
2. Override `layout.html`, `page.html`, or controller templates into the theme
3. Add `assets/site.css` and reference `/theme/site.css` in layout
4. **Configuration → General → Frontend Theme** — select `mysite`
5. Confirm public routes render with your layout and assets

For admin customization, repeat with an admin-capable theme and **Admin Theme**.

See **Help → Templates → Template Functions** for the full function reference.
