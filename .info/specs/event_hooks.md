# Event Hooks

Cannon dispatches Joomla-style event hooks during routing, rendering, authentication, block output, and content display. Hooks can be handled by:

1. **In-process listeners** registered with `hooks.Register` from any Cannon package.
2. **Extensions** that expose the `hooks` capability and subscribe with `OnHook`.

## Available events

### System

| Event | When | Common arguments |
|-------|------|------------------|
| `onBeforeRoute` | After `/ext/…` handling, before DB route matching | `method`, `path`, `query` |
| `onAfterRoute` | After a route is matched, before dispatch | above + `route_id`, `route_name`, `route_type`, `route_path` |
| `onBeforeRender` | Before a layout/page template renders | `layout`, `page` |
| `onAfterRender` | After layout render completes, before bytes are written | `layout`, `page`, `headers`, `body` (mutable), `body_base64` + `body_encoding: "base64"` for binary bodies |

### User

| Event | When | Common arguments |
|-------|------|------------------|
| `onUserBeforeLogin` | Before credential check (frontend + admin login) | `username`, `context` (`frontend` or `admin`), request fields |
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
| `onItemBeforeRender` | Before item page render | `item`, viewer context |
| `onCommentBeforeSave` | Before comment persist | `item_id`, `body`, … |
| `onCommentAfterSave` | After comment persist | `item_id`, `comment_id`, `approved`, … |

## Notifications

Cannon’s notification system listens to hooks in-process and delivers messages via configured channels. See `.info/specs/notifications.md`.

- **Layer 1 (today):** Admin-defined Shoutrrr destinations subscribed to hook events (`internal/notifications`, System → Notifications).
- **Layer 2 (planned):** Per-user and per-role email subscriptions with optional event filters.

In-process listeners register with `hooks.Register` the same way as notifications; dispatch runs after local listeners, before extension hooks.

## In-process registration

```go
import "github.com/rob121/cannon/internal/hooks"

hooks.Register(hooks.OnBeforeRoute, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
    // read or mutate e.Arguments
    return &hooks.Result{Arguments: map[string]any{"seen": true}}, nil
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
s.OnHook("onUserBeforeLogin", func(req extension.HookWireRequest) extension.HookWireResponse {
    username, _ := extension.HookArguments(req)["username"].(string)
    if username == "blocked" {
        return extension.HookAbort("account disabled")
    }
    return extension.HookOK(nil)
})

s.OnHook("onAfterRenderBlock", func(req extension.HookWireRequest) extension.HookWireResponse {
    args := extension.HookArguments(req)
    html, _ := args["html"].(string)
    return extension.HookOK(map[string]any{"html": html})
})
```

Cannon discovers subscriptions via `GET /hooks`:

```json
{"hooks": ["onUserBeforeLogin", "onAfterRenderBlock"]}
```

When an event fires, Cannon `POST`s to `/hooks` with the normal wire fields plus:

```json
{
  "event": "onUserBeforeLogin",
  "arguments": {"username": "jane", "context": "frontend"},
  "method": "POST",
  "url": "/login",
  "user": {"authenticated": false},
  "csrf": "...",
  "site_id": "example"
}
```

Respond with:

```json
{
  "status_code": 200,
  "arguments": {"allowed": false, "error": "reason"},
  "stop": true
}
```

Helpers: `extension.HookOK`, `extension.HookStop`, `extension.HookAbort`, `extension.HookEvent`, `extension.HookArguments`.

Extensions are invoked in extension sort order after in-process listeners. `stop: true` ends the chain.

## Related docs

- `EXTENSIONS.md` — wire protocol and capability overview
- `.info/specs/extensions.md` — `/hooks` capability endpoint
- `.info/specs/notifications.md` — Layer 1 admin destinations and Layer 2 user/role subscriptions
