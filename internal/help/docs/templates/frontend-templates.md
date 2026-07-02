# Frontend Templates

Cannon renders the public site with **Go `html/template`** files. Templates live in your site's template directory (configured in `sites.json` as `template_dir`) and can override built-in files shipped with Cannon.

For a full list of **Cannon template functions** and a **Sprig reference**, see **Help → Templates → Template Functions**.

Use **Admin → Templates** to browse groups, override built-ins, and edit HTML source. Saving keeps up to three timestamped backups under a `versions/` mirror of your template tree.

## Layout and page

Most public pages use two templates:

| File | Role |
|------|------|
| `default/layout.html` | Outer document — `<html>`, shared chrome, navigation shell |
| `default/page.html` | Inner page body rendered into the layout |

Cannon renders the page template first, then injects the result into the layout as **`.Main`**. Fields you pass to the page (for example `.Title`) are also available on the layout.

Built-in layout excerpt:

```html
<main class="container">
  {{.Main}}
</main>
```

Built-in page excerpt:

```html
<div class="py-4">
  {{space "header"}}
  <h1 class="display-6">{{.Title}}</h1>
  {{if .Content}}<div class="content">{{.Content}}</div>{{end}}
</div>
```

Override either file from **Templates → default** when you want site-wide changes without editing Cannon itself.

## Template lookup order

For a path like `default/page.html`, Cannon loads the first match:

1. `{template_dir}/default/page.html` — your site override
2. Built-in copy embedded in the Cannon binary

The same rule applies under `admin/` for admin panel overrides (see the **admin** template group in the template manager).

Extension templates use an `extension/` prefix in your template directory (for example `extension/contact/form.html`) to override HTML returned by extensions. See **Help → Getting Started → Extensions** and `EXTENSIONS.md` in the repo for that layout.

## Block spaces

**Block spaces** are named regions in a template. Extensions with a `block` capability can render HTML into a space when the template calls `space` (not `block` — that name is reserved by Go templates):

```html
{{space "footer"}}
```

Use **`lenspace`** to count blocks in a space before rendering a wrapper:

```html
{{if gt (lenspace "header") 0}}
<div class="site-banner">{{space "header"}}</div>
{{end}}
```

`lenspace` counts active admin-assigned blocks visible to the current viewer.

The string is an arbitrary space name (`header`, `footer`, `sidebar`, and so on). Cannon asks installed extensions for a matching block definition and renders the result inline.

### Debug outlines

Enable **Debug Template Spaces** under **Admin → System → Configuration → General**, then add `?tp=1` to any public URL. Each `{{space "…"}}` region is wrapped in a dashed red `#FF3300` border with the space name shown as a label. Useful when placing blocks and checking that spaces match your template calls.

Use block spaces for reusable areas — contact forms, newsletter signups, or other extension-provided fragments — without hard-coding extension HTML in your layout.

## Menus

Load admin-managed navigation with the **`menu`** template function:

```html
<ul class="nav">
  {{range menu "main"}}
  <li class="nav-item">
    <a class="nav-link {{.Class}}" href="{{.Href}}" target="{{.Target}}">{{.Name}}</a>
  </li>
  {{end}}
</ul>
```

Create the menu under **Admin → Menus**, set its **Menu name** to the same string you pass to `menu` (for example `main`), and attach menu items to routes.

Each item is a map with:

| Key | Description |
|-----|-------------|
| `Name` | Link label |
| `Href` | URL from the linked route |
| `Class` | Optional CSS class |
| `Target` | Optional link target (`_blank`, etc.) |

## Extension routes

When a route's type is **Extension**, Cannon calls the extension's page handler and wraps the returned HTML in `default/layout.html` and `default/page.html`.

Template data for those routes includes:

| Field | Description |
|-------|-------------|
| `.Title` | Route name from **Admin → Routes** |
| `.Content` | HTML fragment returned by the extension |
| `.Main` | Rendered page body (available on the layout, same as other pages) |

Customize `default/page.html` if you need a different wrapper around extension output.

## Authoring tips

**Use valid HTML paths.** Template paths must end with `.html` (for example `default/layout.html`, not `default/layout`).

**Auto-escaping.** `html/template` escapes values in `{{.Field}}` by default. Use `{{.Content}}` only for trusted HTML (such as extension output). Prefer plain text fields for user-supplied data.

**Conditionals and loops.** Standard Go template syntax applies:

```html
{{if .Subtitle}}<p class="lead">{{.Subtitle}}</p>{{end}}

{{range .Items}}
  <article>{{.Title}}</article>
{{end}}
```

**Bootstrap.** Built-in `default` templates use Cannon theme styles at `/theme/site.css` (emerald palette, Inter + Source Serif 4). Override `default/layout.html` or link your own assets from your site template directory.

**Site name.** Layout templates can call `{{siteName}}` for the site label from `sites.json`, and `{{year}}` for the current year in footers.

## Template functions

Every template can call **Sprig** helpers plus Cannon functions such as `space`, `lenspace`, `menu`, `siteName`, `year`, and `csrfField`.

```html
{{upper .Title}}
{{default "Untitled" .Title}}
{{if gt (lenspace "sidebar") 0}}{{space "sidebar"}}{{end}}
```

Cannon overrides Sprig when names collide (`add`, `sub`, `mul`, `div`, `min`, `dict`, `initials`) so pagination and admin lists behave predictably with integers.

See **Help → Templates → Template Functions** for the complete Cannon and Sprig reference.

**Local assets.** Routes of type **Local File** serve files from the site `assets_dir`. Reference them with normal URLs in your templates (for example `/files/brochure.pdf` when configured in routes).

## Workflow summary

1. Open **Admin → Templates → default** and review built-in files.
2. Click **Override** on `layout.html` or `page.html` to copy the built-in source into your site template directory.
3. Edit the HTML and save. Prior versions are kept under `versions/` automatically.
4. Add `{{space "…"}}` and `{{menu "…"}}` where you need extension blocks and navigation.
5. Load the public site and confirm the route or extension page renders with your layout.

For admin UI templates, use the **admin** group instead of **default**. Frontend and admin overrides are separate trees under the same `template_dir`.
