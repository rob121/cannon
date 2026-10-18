# Cannon

Cannon is a multisite CMS written in Go. It serves multiple sites from one process, runs an admin console, and extends itself through separate extension binaries over Unix sockets—so features can ship and update without rebuilding the core.

## Features

- **Multisite** — Host-based site resolution (plus `X-Host` for local development), with per-site databases, templates, assets, and language packs
- **Admin console** — Content, media, routes, menus, users, configuration, extensions, and system tools in one UI
- **Content model** — Items, categories, tags, custom fields, blocks, comments, and media library
- **Dynamic routing & templates** — Configurable routes, controller actions, theme templates, and Sprig-powered helpers
- **Process extensions** — Precompiled binaries started as child processes; capabilities cover pages, middleware, admin UI, captcha, settings, help docs, and more ([EXTENSIONS.md](EXTENSIONS.md))
- **Auth & access control** — Users, groups, roles, permissions, authenticators, WebAuthn, TOTP, and OAuth providers
- **i18n** — Editable language packs for the public site and admin
- **APIs & realtime** — Content API credentials, live analytics hooks, and Centrifuge-backed updates where enabled
- **Updates** — In-admin checks for Cannon binary updates (staged `cannon.latest` / rollback via `cannon.previous`) and per-extension releases
- **Databases** — SQLite by default; MySQL and PostgreSQL via GORM
- **Install wizard** — First-run setup for site, database, and administrator account

## Prerequisites

- Go **1.26+** (see `go.mod`)
- A C compiler (GCC/Clang) for SQLite via `go-sqlite3`

## Quick start

```bash
git clone https://github.com/rob121/cannon.git
cd cannon
cp sites.json.example sites.json
go run ./cmd/cannon --port 8001
```

Or build a binary:

```bash
go build -o cannon ./cmd/cannon
./cannon --port 8001
```

Default listen port is **8001**. Useful flags:

| Flag | Description |
|------|-------------|
| `--port` | HTTP listen port (default `8001`) |
| `--version` | Print version and exit |

## First-time setup

On a fresh install (no sites configured), Cannon redirects to the install wizard:

1. Open [http://localhost:8001/install](http://localhost:8001/install)
2. Configure the site (name, host URL, database)
3. Create the administrator account
4. Sign in at [http://localhost:8001/admin/login](http://localhost:8001/admin/login)

SQLite is the default and is created during install. Site data lives under `./data/<site-id>/` unless you change `data_root`.

To re-run the installer later, set `"install_enabled": true` in `sites.json`.

## Configuration

Cannon loads `sites.json` from the first path that exists:

1. `$CANNON_CONFIG` (if set)
2. `./sites.json`
3. `sites.json` beside the Cannon binary
4. `$HOME/.cannon/sites.json`
5. `/etc/cannon/sites.json`

See [`sites.json.example`](sites.json.example) for the schema. Typical top-level keys:

- `data_root` — shared data directory
- `extensions.dir` / `extensions.sockets_dir` — extension binaries and sockets
- `update.url_base` — Cannon release download base (GitHub Releases by default)
- `sites[]` — per-site host, database, template/asset/language paths

For local development against a site whose `host` is not `localhost`, send an `X-Host` header matching that site URL.

## Templating

Public pages use **Go `html/template`** files organized as **themes** under each site's `template_dir`. The built-in frontend theme is `default` (embedded in the binary). Customize by creating a theme folder, overriding the files you need, then selecting it under **Configuration → General → Frontend Theme**.

### Create a theme

**From admin (recommended):**

1. **Admin → Templates → New Theme**
2. Choose a folder name (e.g. `mysite`), type (`frontend` or `full`), and metadata
3. Open the theme browser and use **Override** on `layout.html` (and any controller templates) to copy built-in sources into your theme
4. Set **Frontend Theme** to your theme name in **Configuration → General**

**On disk**, a theme looks like:

```text
{template_dir}/mysite/
├── template.json          # name, type (frontend|backend|full), status, version
├── layout.html            # outer document; must include {{.Main}}, {{.HeadExtra}}, {{.BodyEndExtra}}
├── page.html              # inner body for many extension routes
├── controllers/
│   └── content/
│       ├── index.html
│       ├── category.html
│       └── item.html
└── assets/                # served at /theme/* when this theme is active
    └── site.css
```

`template.json` example:

```json
{
  "name": "My Site Theme",
  "author": "Acme",
  "type": "frontend",
  "status": "active",
  "version": "1.0.0"
}
```

Reserved folder names (not themes): `versions`, `extension`.

### Layout essentials

Custom `layout.html` should keep Cannon's document hooks so extensions (analytics, captcha, etc.) can inject markup:

```html
<head>
  <title>{{if .Title}}{{.Title}}{{else}}{{lang "common.home"}}{{end}}</title>
  {{template "metaTags" .}}
  <link href="{{themeAsset "site.css"}}" rel="stylesheet">
  {{if .HeadExtra}}{{.HeadExtra}}{{end}}
</head>
<body>
  <main>{{.Main}}</main>
  {{if .BodyEndExtra}}{{.BodyEndExtra}}{{end}}
</body>
```

| Piece | Role |
|-------|------|
| `layout.html` | Chrome, nav, `{{.Main}}` |
| `controllers/{controller}/{action}.html` | Page bodies for built-in controllers |
| `assets/` → `/theme/...` | Theme CSS/JS/images (separate from media uploads under `/assets/`) |
| `{template_dir}/extension/{name}/...` | Overrides for extension-rendered HTML |

Templates get **Sprig** plus Cannon helpers such as `lang`, `themeAsset`, `absoluteURL`, and site settings accessors. Details:

| Doc | Contents |
|-----|----------|
| [Frontend templates](internal/help/docs/templates/frontend-templates.md) | Themes, layout hooks, headers, assets, resolution order |
| [Template functions](internal/help/docs/templates/template-functions.md) | Sprig + Cannon function reference |
| [Templates spec](.info/specs/templates.md) | Theme metadata, loading rules, admin themes |

In a running site, the same material is under **Admin → Help → Templates**.

## Production notes

A sample systemd unit is in [`deploy/cannon.service`](deploy/cannon.service). For in-admin binary updates to work:

- Use `Restart=always` (the process exits after staging/rollback so systemd can bring it back)
- Allow writes to the binary directory (for `cannon.latest` / `cannon.previous`) and to your data paths

Reload Application in admin refreshes config and extensions; it does **not** replace the Cannon binary.

## Extensions

Extensions are discovered from the configured extensions directory, registered in the admin UI, and started with `--site` and `--socket`. They advertise capabilities over HTTP on a Unix domain socket.

- Authoring protocol and capability reference: [EXTENSIONS.md](EXTENSIONS.md)
- Official / example extensions live as separate `cannon-ext-*` repositories

## Documentation

| Doc | Contents |
|-----|----------|
| [EXTENSIONS.md](EXTENSIONS.md) | Extension wire protocol, capabilities, updates |
| [Frontend templates](internal/help/docs/templates/frontend-templates.md) | Creating themes and layout conventions |
| [Template functions](internal/help/docs/templates/template-functions.md) | Available template helpers |
| [.info/specs/templates.md](.info/specs/templates.md) | Template / theme specification |
| [.info/AGENT.MD](.info/AGENT.MD) | Project overview and architecture notes |
| [.info/ANSWERS.md](.info/ANSWERS.md) | Design decisions |
| [.info/specs/](.info/specs/) | Feature specifications |
| [`deploy/cannon.service`](deploy/cannon.service) | Example systemd unit |

In-app help is also available under **Admin → Help** once Cannon is running.
