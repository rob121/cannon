# Extension Authoring Guide

Cannon extensions are standalone Go programs that speak a simple HTTP wire protocol over a Unix socket. This guide covers how to build one and register the capabilities your extension supports.

## Project layout

```
my-extension/
  main.go
  go.mod
```

Use `github.com/rob121/cannon/extension` as your dependency. Call `extension.New` with metadata and register only the capabilities you implement.

## Capabilities

| Capability | Register with | Purpose |
|------------|---------------|---------|
| Pages | `RegisterPage` / `HandlePage` | Frontend routes rendered inside Cannon layouts |
| Blocks | `RegisterBlock` / `HandleBlock` | Content for `{{space "…"}}` template regions |
| Endpoints | `RegisterEndpoint` / `HandleEndpoint` | Form posts and AJAX actions |
| Data | `HandleData` / `OnData` | JSON APIs for admin or frontend |
| Admin | `HandleAdmin` | Screens under **Extensions** in admin |
| Hooks | `OnHook` | React to Cannon lifecycle events |
| Configuration | `OnConfiguration` | JSON Forms settings stored per site |
| Help | `EmbedHelp` | Markdown docs under **Help → Extensions** |
| Install | `OnInstall` | One-time setup when activated |

You do not need every capability. Register only what your extension provides; Cannon ignores missing handlers.

## Minimal block extension

```go
package main

import "github.com/rob121/cannon/extension"

func main() {
    ext := extension.New(extension.Info{
        Name:    "hello",
        Version: "1.0.0",
        Title:   "Hello Block",
    })
    ext.HandleBlock("greeting", func(ctx extension.BlockContext) (string, error) {
        return "<p>Hello from an extension block.</p>", nil
    })
    ext.ListenAndServe()
}
```

Add the socket path to **Admin → Extensions**, activate the extension, then create a block of type **Extension** pointing at your block id.

## Admin UI extensions

`HandleAdmin` receives Turbo-framed requests. Return HTML fragments that use Cannon admin CSS classes (`admin-form-control`, `admin-data-card`, etc.) for a consistent look.

## Configuration

`OnConfiguration` accepts JSON Schema and UI Schema (JSON Forms). Saved values are available on each request via the extension wire protocol.

## Hooks

Use `OnHook` to listen for Cannon events such as `onUserAfterLogin` or `onItemBeforeSave`. Hook arguments are passed as JSON maps; return modified arguments or abort with an error.

## Testing locally

1. Build: `go build -o my-extension .`
2. Point `sites.json` extensions dir at your binary or socket.
3. Activate under **Admin → Extensions** and reload the site.

See [EXTENSIONS.md](/EXTENSIONS.md) in the repository root for the full wire protocol reference.
