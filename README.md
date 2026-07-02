# Cannon

A multisite content management system written in Go.

## Prerequisites

- Go 1.21 or later
- GCC (required for SQLite via `go-sqlite3`)

## Build

Clone the repository and build the binary:

```bash
git clone https://github.com/rob121/cannon.git
cd cannon
go build -o cannon ./cmd/cannon
```

## Run

Copy the example config and start the server:

```bash
cp sites.json.example sites.json
./cannon
```

By default Cannon listens on port **8001**. Change the port with the `--port` flag:

```bash
./cannon --port 9000
```

Or run without building:

```bash
go run ./cmd/cannon --port 8001
```

## First-time setup

On first startup, Cannon redirects all requests to the install wizard:

1. Open [http://localhost:8001/install](http://localhost:8001/install)
2. Configure your site (name, host URL, database)
3. Create the administrator account
4. Sign in at [http://localhost:8001/admin/login](http://localhost:8001/admin/login)

SQLite is the default database and is created automatically during install. Site data is written under `./data/<site-id>/`.

To re-run the installer later, set `"install_enabled": true` in `sites.json`.

## Configuration

Cannon reads `sites.json` from the current directory (or `$HOME/.cannon/sites.json`, or `/etc/cannon/sites.json`). See `sites.json.example` for the schema.

For local development against a site configured with a different host, send the `X-Host` header matching the site URL in `sites.json`.

## Documentation

- [AGENT.MD](AGENT.MD) — project overview and architecture
- [ANSWERS.md](ANSWERS.md) — design decisions
- [specs/](specs/) — feature specifications
