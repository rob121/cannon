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

### Legacy pack layout

If your site still has a root `template.json` and/or `default/` / `admin/` trees under `template_dir`, **Admin → Templates** shows a one-time **Migrate Legacy Layout** form. That moves files into named theme folders, writes per-theme metadata, and updates the active theme settings. It does not run automatically on startup.

Template and asset edits in admin use atomic saves (temp file + rename) and keep up to three prior versions under `versions/`.

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

## Header templates

Public pages use three cooperating header pieces. Override these partials in a custom theme when you want consistent page chrome without duplicating markup in every controller template.

### Document `<head>` (`layout.html` + `meta-tags`)

`layout.html` sets the HTML document title and includes the `metaTags` partial:

```html
<title>{{if .MetaTitle}}{{.MetaTitle}}{{else if .Title}}{{.Title}}{{else}}{{lang "common.home"}}{{end}}{{if siteName}} · {{siteName}}{{end}}</title>
{{template "metaTags" .}}
<link href="{{themeAsset "site.css"}}" rel="stylesheet">
```

`meta-tags.html` (`{{define "metaTags"}}`) renders SEO and social tags. It falls back to Configuration → SEO defaults via `siteMetaDescription`, `siteOGTitle`, and related functions. Canonical and image URLs use `absoluteURL` so relative paths work behind reverse proxies.

Controllers and extension routes can pass page-level SEO fields on the layout data:

| Field | Used for |
|-------|----------|
| `.MetaTitle` | `<title>`, `og:title`, `twitter:title` |
| `.MetaDescription` | `description`, `og:description`, `twitter:description` |
| `.MetaKeywords` | `keywords` |
| `.CanonicalURL` | `<link rel="canonical">` (passed through `absoluteURL`) |
| `.OGImage` | `og:image`, `twitter:image` |

### Page header block (`page-header`)

Most controller templates open with the shared `pageHeader` partial:

```html
{{template "pageHeader" (dict "Title" (lang "search.title") "Subtitle" (lang "search.subtitle"))}}
```

`page-header.html` respects route configuration through `showRouteTitle`. When a route hides its title in **Admin → Routes**, the header is omitted unless you pass `AlwaysShow`:

| `dict` key | Description |
|------------|-------------|
| `Title` | Main `<h1>` text |
| `Subtitle` | Optional lead paragraph under the title |
| `Eyebrow` | Small uppercase label above the title (errors, extension pages) |
| `AlwaysShow` | Render even when `showRouteTitle` is false (maintenance, errors) |

Example with route-aware visibility:

```html
{{if .ShowCategoryTitle}}{{template "pageHeader" (dict "Title" .Category.Name)}}{{end}}
```

### Navigation active state

Menu partials (`menu-nav.html`) mark the current page with `isCurrentPath`:

```html
<a class="nav-link{{if isCurrentPath $item.Href}} active{{end}}" href="{{$item.Href}}">{{$item.Name}}</a>
```

Use `isRoute "content" "item"` when matching controller/action pairs instead of static paths.

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

Content item detail pages also expose per-item spaces derived from the item slug:

| Space | Placement |
|-------|-----------|
| `content-{slug}-start` | Before the intro |
| `content-{slug}-middle` | Between intro and body |
| `content-{slug}-end` | After the body |

Example for an item with slug `important-dates`: assign blocks to `content-important-dates-start`, `content-important-dates-middle`, or `content-important-dates-end`.

Enable **Debug Template Spaces** under Configuration → General, then append `?tp=1` to public URLs. The same `?tp=1` query also appends each `{{lang "key"}}` lookup key in parentheses (e.g. `Sign in (nav.sign_in)`) to help when editing locale files.

Space preview shows a labeled red dashed outline even for empty spaces. During preview, `lenspace` counts an empty space's outline as one placeholder and `hasSpace` returns true, so conditional space wrappers remain visible. Fallback content in the corresponding `else` branch is replaced by the space outline. Outside preview these helpers report only active blocks. Other conditions, such as homepage-only sections, still apply.

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

**Bootstrap.** Built-in styles use `{{themeAsset "site.css"}}`. Override by placing `site.css` in your theme's `assets/` folder.

**Page headers.** Use `{{template "pageHeader" …}}` for in-page titles; see **Header templates** above.

**Item listings.** Use `userName`, `itemDate`, `itemDateISO`, and `excerpt` in cards and search results — see **Help → Templates → Template Functions**.

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
