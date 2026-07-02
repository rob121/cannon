# Templates

Cannon templates are organized as **themes**: self-contained folders under each site's `template_dir`. A theme holds HTML templates, static assets, and its own `template.json` metadata.

---

## Concepts

| Term | Meaning |
|------|---------|
| **Theme** | Folder under `{template_dir}/{name}/` with HTML, `assets/`, and `template.json` |
| **Built-in theme** | Embedded `default` (public site) or `admin` (admin UI) — not a folder on disk |
| **Active theme** | Selected in **Configuration → General** (`frontend_theme`, `admin_theme`) |
| **Logical path** | What the engine requests at render time, e.g. `default/layout.html`, `admin/dashboard.html` |

Blocks (database content in named spaces) are **not** templates. Extension template overrides still live at `{template_dir}/extension/{ext}/...`.

---

## Configuration

### Site template directory

Each site in `sites.json`:

```json
{
  "template_dir": "/path/to/data/example/templates"
}
```

### Active themes (global settings)

**Admin → Configuration → General**

| Setting | Default | Description |
|---------|---------|-------------|
| **Frontend Theme** | `default` | Public site template pack |
| **Admin Theme** | `admin` | Admin UI template pack |

Dropdown options include **Built-in (default/admin)** plus each **active** custom theme folder whose `template.json` type allows that role:

- Frontend dropdown: themes with type `frontend` or `full`
- Admin dropdown: themes with type `backend` or `full`

Inactive themes (`status: inactive` in `template.json`) do not appear in the dropdown.

---

## Theme folder layout

```
{template_dir}/
├── versions/                     # auto snapshots on save (global mirror)
│   └── mysite/
│       └── version-{nano}-layout.html
├── extension/                    # extension HTML overrides (unchanged)
│   └── contact/form.html
└── mysite/                       # custom theme
    ├── template.json             # theme metadata (required for discovery)
    ├── layout.html
    ├── page.html
    ├── maintenance.html
    ├── controllers/
    │   └── content/
    │       ├── index.html
    │       ├── category.html
    │       └── item.html
    └── assets/                   # served at /theme/* when this theme is active
        ├── site.css
        ├── app.js
        └── images/logo.png
```

### Reserved names

These are **not** valid theme folder names: `versions`, `extension`.

---

## Theme metadata (`template.json`)

Each theme has its own metadata file at **`{theme}/template.json`** (not at the root of `template_dir`).

Edit via **Admin → Templates → {theme} → Metadata**.

| Field | Description |
|-------|-------------|
| `name` | Display name |
| `author` | Author or team |
| `description` | Short summary |
| `version` | Semantic version string |
| `type` | `frontend`, `backend`, or `full` |
| `status` | `active` or `inactive` |

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

When a theme is **inactive**, it cannot be selected in Configuration and its files are not used even if still on disk.

---

## How templates are loaded

### Resolution order

When the engine renders logical path `default/controllers/content/item.html`:

1. Read **frontend theme** from Configuration → General
2. If built-in `default` → load from embedded binary
3. If custom theme `mysite` → load `{template_dir}/mysite/controllers/content/item.html` when the theme is active
4. Fall back to embedded built-in if the file is missing

Admin templates (`admin/dashboard.html`) follow the same pattern using **admin theme**.

### Admin template set

Admin HTML files in a custom admin theme are parsed into one shared template set (fragments like `_fragments.html` work the same as built-in admin).

### Versioning

Saving a template via the admin editor keeps up to three prior copies under `{template_dir}/versions/{theme}/`.

---

## Static assets

Themes ship static files in an **`assets/`** subdirectory.

| URL prefix | Serves from |
|------------|-------------|
| `/theme/{path}` | Active **frontend** theme `{theme}/assets/{path}` → embed fallback |
| `/admin/assets/{path}` | Active **admin** theme `{theme}/assets/{path}` → embed fallback |
| `/assets/{path}` | Site `assets_dir` (media uploads, etc.) — separate from themes |

Built-in layout references `/theme/site.css`, which resolves to `{theme}/assets/site.css` when using a custom frontend theme.

Reference assets in templates:

```html
<link href="/theme/site.css" rel="stylesheet">
<link href="/theme/images/logo.png" rel="icon">
<script src="/admin/assets/admin.js" defer></script>
```

---

## Request → template selection

Unchanged logical paths; theme selection determines **where files are loaded from**:

| Request type | Layout | Page |
|--------------|--------|------|
| Controller route | `default/layout.html` | `default/controllers/{controller}/{action}.html` |
| Extension page | `default/layout.html` | `default/page.html` |
| Site offline | `default/layout.html` | `default/maintenance.html` |
| Admin page | `admin/layout.html` | `admin/{page}.html` |

### Category override

Category **Template** field still accepts a full theme-relative path, e.g. `mysite/controllers/content/custom-category.html`.

### Render hooks

`OnBeforeRender` can still change `layout` and `page` paths at runtime.

---

## Admin UI

| Route | Purpose |
|-------|---------|
| `/admin/templates` | List theme folders |
| `/admin/templates/new?create=theme` | Create a new theme |
| `/admin/templates/{theme}` | Browse templates in a theme |
| `/admin/templates/{theme}/meta` | Edit `{theme}/template.json` |
| `/admin/templates/edit?path={theme}/layout.html` | Edit raw HTML |
| POST `/admin/templates/override` | Copy built-in → theme folder |

---

## Authoring workflow

1. **Admin → Templates → New Theme** — creates folder, `template.json`, and `assets/`
2. Open the theme, **Override** built-in files (e.g. `layout.html`) into the theme folder
3. Add CSS/JS/images under `{theme}/assets/`
4. **Configuration → General** — set **Frontend Theme** or **Admin Theme** to your theme name
5. Load the site and confirm templates and assets render

See also: **Help → Templates → Frontend Templates** and **Template Functions**.

---

## Package reference

| Package | Role |
|---------|------|
| `internal/themes` | Theme discovery, selection, asset paths, config schema patching |
| `internal/templateengine` | Render pipeline, embed vs theme disk loading |
| `internal/templatemgr` | Admin CRUD, override, versioning |
| `internal/templatemeta` | Per-theme `template.json` |

---

## Migration note

Older layouts that placed `default/` or `admin/` directly under `template_dir` (with root `template.json`) are superseded by the theme-folder model. Move content into a named theme folder, add `{theme}/template.json`, and select the theme in Configuration → General.
