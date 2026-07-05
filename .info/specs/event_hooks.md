# Event Hooks

Cannon dispatches Joomla-style event hooks during routing, rendering, authentication, block output, content display, and site generation. Hooks can be handled by:

1. **In-process listeners** registered with `hooks.Register` from any Cannon package.
2. **Extensions** that expose the `hooks` capability and subscribe with `OnHook`.

## Available events

### System

| Event | When | Common arguments | Returns |
|-------|------|------------------|---------|
| `onBeforeRoute` | After `/ext/…` handling, before DB route matching | `method`, `path`, `query` | — |
| `onAfterRoute` | After a route is matched, before dispatch | above + `route_id`, `route_name`, `route_type`, `route_path` | — |
| `onBeforeRender` | Before a layout/page template renders | `layout`, `page`, `context` (`frontend` or `admin`) | `layout`, `page` |
| `onAfterRender` | After layout render completes, before bytes are written | `layout`, `page`, `headers`, `body` (mutable), `body_base64` + `body_encoding: "base64"` for binary bodies | `body`, `headers` |
| `onPrepareDocumentHead` | Before frontend layout `<head>` closes | `layout`, `page`, `context: "frontend"`, route fields | `head_html` (appended) |
| `onPrepareDocumentBody` | Before frontend layout `</body>` | same | `body_html` (appended) |
| `onAdminBeforeRender` | Before admin layout/page renders | `layout`, `page`, `context: "admin"` | `layout`, `page` |
| `onAdminPrepareDocumentHead` | Before admin layout `<head>` closes | same | `head_html` (appended) |
| `onAdminPrepareDocumentBody` | Before admin layout `</body>` | same | `body_html` (appended) |
| `onSettingsSave` | After global configuration section saved | `scope`, `section`, `data` | — |
| `onSitemapGenerate` | While building `/sitemap.xml` | `base_url`, `sitemap_urls` (seed list) | `sitemap_urls` (appended entries with `loc`, optional `lastmod`) |
| `onRobotsGenerate` | While building `/robots.txt` | `body` (mutable) | `body`, `robots_append` (appended lines) |

Document hooks are the preferred way to inject analytics scripts, fonts, or structured data. Cannon merges `head_html` and `body_html` from every listener into `{{.HeadExtra}}` and `{{.BodyEndExtra}}` in the layout. Admin and frontend layouts use separate hook names so extensions can target one surface without affecting the other.

Static site-wide head markup remains available through **SEO & Robots → Additional head markup** (`site_head_extra`).

### User

| Event | When | Common arguments |
|-------|------|------------------|
| `onUserBeforeLogin` | Before credential check (frontend, admin, API) | `username`, `context` (`frontend`, `admin`, or `api`), request fields |
| `onUserAfterLogin` | After successful auth, before session commit | `user_id`, `username`, `email`, `context` |
| `onUserLogout` | Before session is cleared | `user_id`, `username`, `email`, `context` |
| `onUserSignup` | After public registration creates a user | `user_id`, `username`, `email` |
| `onUserVerified` | After email verification completes | `user_id`, `username`, `email` |
| `onUserLocked` | After admin locks an account | `user_id`, `username`, `email` |

Set `allowed: false` and optional `error` in returned arguments (or use `extension.HookAbort`) to block login.

### Block

| Event | When | Common arguments |
|-------|------|------------------|
| `onRenderBlock` | Before a block renders | `block_id`, `block_type`, `space`, `extension`, `block_item` |
| `onAfterRenderBlock` | After block HTML is produced | above + `html` (mutable) |

### Content

| Event | When | Common arguments |
|-------|------|------------------|
| `onContentPrepare` | Before markdown/HTML/extension content is processed | `content`, `content_type`, context fields |
| `onContentBeforeDisplay` | Before content is wrapped in layout / controller view | `title`, `content`, `controller`, `action`, `data`, route fields |
| `onContentAfterDisplay` | After page body template executes, before layout wrap | `layout`, `page`, `body` (mutable) |
| `onItemBeforeSave` | Before item persist (admin or frontend) | `item`, `is_new`, `form` |
| `onItemAfterSave` | After item persist | `item_id`, `item` |
| `onItemBeforeDelete` | Before permanent item delete | `item_id`, `item` |
| `onItemAfterDelete` | After permanent item delete | `item_id`, `item` |
| `onItemTrash` | After item moved to trash | `item_id`, `item` |
| `onItemRestore` | After item restored from trash | `item_id`, `item` |
| `onItemBeforeRender` | Before item page render | `item`, viewer context |
| `onCategoryBeforeSave` | Before category persist | `category`, `is_new`, `form` |
| `onCategoryAfterSave` | After category persist | `category`, `is_new` |
| `onCategoryBeforeDelete` | Before category delete | `category_id`, `category` |
| `onMediaUpload` | After media asset saved | `asset`, `folder` |
| `onMediaDelete` | Before media asset deleted | `asset_id`, `asset` |
| `onRevisionRestore` | Before item revision rollback applied | `item_id`, `revision_id`, `item` |
| `onBeforeSearch` | Before search query executes | `query`, `category_id`, `tag_id`, `author_id`, `sort`, `page`, `field_filters` |
| `onAfterSearch` | After search results loaded | `query`, `total`, `page`, `items` |
| `onCommentBeforeSave` | Before comment persist | `item_id`, `body`, … |
| `onCommentAfterSave` | After comment persist | `item_id`, `comment_id`, `approved`, … |

### Mail

| Event | When | Common arguments |
|-------|------|------------------|
| `onBeforeMailSend` | Before SMTP send | `to`, `subject`, `text`, `html` (all mutable) |

## Fragment merging

Several hooks append rather than replace prior listener output:

- `head_html`, `body_html`, `robots_append` — concatenated with newlines
- `sitemap_urls` — arrays merged in order

Use helpers `extension.HookHeadHTML`, `extension.HookBodyHTML`, `extension.HookRobotsAppend`, and `extension.HookSitemapURLs` when handling wire responses.

## Notifications

Cannon’s notification system listens to hooks in-process and delivers messages via configured channels. See `.info/specs/notifications.md`.

- **Layer 1 (today):** Admin-defined Shoutrrr destinations subscribed to hook events (`internal/notifications`, System → Notifications).
- **Layer 2 (planned):** Per-user and per-role email subscriptions with optional event filters.

In-process listeners register with `hooks.Register` the same way as notifications; dispatch runs in-process listeners first, then extension hooks.

## In-process registration

```go
import "github.com/rob121/cannon/internal/hooks"

hooks.Register(hooks.OnPrepareDocumentHead, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
    return &hooks.Result{Arguments: map[string]any{
        "head_html": `<script src="/analytics.js" defer></script>`,
    }}, nil
})
```

Within request handlers, dispatch using context (wired automatically by middleware):

```go
args, err := hooks.Fire(r.Context(), hooks.OnContentPrepare, map[string]any{
    "content": raw,
    "content_type": "block_markdown",
})
```

## Extension hooks capability

Register handlers on the extension server:

```go
s.OnHook("onPrepareDocumentHead", func(req extension.HookWireRequest) extension.HookWireResponse {
    return extension.HookOK(map[string]any{
        "head_html": `<link rel="stylesheet" href="/ext/my-ext/theme.css">`,
    })
})

s.OnHook("onItemTrash", func(req extension.HookWireRequest) extension.HookWireResponse {
    itemID := extension.HookArguments(req)["item_id"]
    // purge external cache...
    return extension.HookOK(nil)
})

s.OnHook("onSitemapGenerate", func(req extension.HookWireRequest) extension.HookWireResponse {
    return extension.HookOK(map[string]any{
        "sitemap_urls": []map[string]any{
            {"loc": "https://example.com/custom-page", "lastmod": "2026-07-04"},
        },
    })
})
```

Cannon discovers subscriptions via `GET /hooks`:

```json
{"hooks": ["onPrepareDocumentHead", "onItemTrash"]}
```

When an event fires, Cannon `POST`s to `/hooks` with the normal wire fields plus:

```json
{
  "event": "onPrepareDocumentHead",
  "arguments": {"layout": "default/layout.html", "page": "default/controllers/content/index.html", "context": "frontend"},
  "method": "GET",
  "url": "/",
  "user": {"authenticated": false},
  "csrf": "...",
  "site_id": "example"
}
```

Respond with:

```json
{
  "status_code": 200,
  "arguments": {"head_html": "<script>...</script>"},
  "stop": false
}
```

Helpers: `extension.HookOK`, `extension.HookStop`, `extension.HookAbort`, `extension.HookEvent`, `extension.HookArguments`, `extension.HookHeadHTML`, `extension.HookBodyHTML`, `extension.HookRobotsAppend`, `extension.HookSitemapURLs`.

Extensions are invoked in extension sort order after in-process listeners. `stop: true` ends the chain.

## Related docs

- `EXTENSIONS.md` — wire protocol and capability overview
- `.info/specs/extensions.md` — `/hooks` capability endpoint
- `.info/specs/notifications.md` — Layer 1 admin destinations and Layer 2 user/role subscriptions
