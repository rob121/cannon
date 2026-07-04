# Notifications

Cannon delivers event-driven notifications in two layers. Both layers subscribe to the same hook events (see `.info/specs/event_hooks.md`) but differ in **who configures them** and **who receives messages**.

## Goals

- **Layer 1 — Admin destinations:** Site operators configure fixed outbound channels (Slack, SMTP list, webhooks) for ops and monitoring.
- **Layer 2 — User and role subscriptions:** Individual users and roles opt into events they care about; messages go to the subscriber’s email (and later optional in-app inbox).

Layers are complementary. A single hook firing may notify Layer 1 destinations **and** all matching Layer 2 subscribers.

---

## Layer 1 — Admin destinations (implemented, extend)

### Model (today)

| Table | Purpose |
|-------|---------|
| `notifications` | Named destination: `name`, `shoutrr_url`, `status` |
| `notification_events` | Many-to-many: which hook events trigger this destination |

Admin UI: **System → Notifications** (`/admin/notifications`). Permission: `core.admin.notifications.read` (manage requires write permission when split).

### Delivery

1. Hook fires → `hooks.Fire` → in-process listeners (including `internal/notifications`).
2. `DispatchEvent` loads active `Notification` rows subscribed to that event.
3. Each row sends a plain-text message via its Shoutrrr URL (`smtp://`, `slack://`, generic webhook).

Recipients are **not** derived from users or roles. They are encoded in the Shoutrrr URL (e.g. SMTP `to=`, Slack webhook).

### Events available in admin UI (today)

| Event | Typical use |
|-------|-------------|
| `onUserSignup` | New account alert |
| `onUserAfterLogin` | Login monitoring |
| `onUserVerified` | Email verification complete |
| `onUserLocked` | Account lockout alert |

### Layer 1 gaps (to implement)

- Expose additional hooks in the admin event picker (content, comments, editorial — see **Event catalog** below).
- Richer message templates per event (subject/body with placeholders, not one-line plain text).
- Optional global Shoutrrr defaults in Configuration (fallback when per-notification URL omitted).
- Delivery log / last-error column in admin list (debug failed sends).
- Tie to site mail settings (`internal/mail`) for SMTP when Shoutrrr URL uses site defaults.

---

## Layer 2 — User and role subscriptions (implemented)

### Concepts

| Subscriber type | Who configures | Who receives |
|-----------------|----------------|--------------|
| **User** | The user (account/profile UI) or an admin on their behalf | That user’s account email |
| **Role** | Admin (role form or dedicated subscriptions UI) | Every active user assigned that role at send time |

A subscription binds:

- **Subscriber:** `user_id` **or** `role_id` (exactly one; not both on the same row).
- **Event:** hook name (same vocabulary as Layer 1).
- **Channel:** `email` initially; reserve `in_app` for a future inbox.
- **Status:** active / inactive.
- **Optional filters** (phase 2+): e.g. only items in category X, only when `author_id` matches subscriber, only pending-review items.

### Proposed model

```
notification_subscriptions
  subscription_id   PK
  user_id           FK nullable, index
  role_id           FK nullable, index
  event             string(64), not null, index
  channel           string(16), default 'email'
  status            active | inactive
  filters_json      text nullable
  created_at, updated_at

CHECK: (user_id IS NOT NULL AND role_id IS NULL)
    OR (user_id IS NULL AND role_id IS NOT NULL)
UNIQUE (user_id, event, channel) WHERE user_id IS NOT NULL
UNIQUE (role_id, event, channel) WHERE role_id IS NOT NULL
```

Role subscriptions are **defaults for role members**, not a separate mailing list. At dispatch time, Cannon resolves role → current member user IDs, deduplicates emails, and skips users who lack a valid email or are inactive/locked.

### User-facing UI

- **Account → Notifications** (or profile tab): checklist of subscribable events grouped by category (Account, Content, Comments).
- Show only events the user is allowed to know about (e.g. hide admin-only hooks from viewers).
- Per-user overrides: user subscription wins over role defaults for the same event (explicit opt-out on role-granted events).

### Admin UI

- **Roles → Edit → Notifications:** default subscriptions for everyone with that role (e.g. Editors → `onItemAfterSave` when status becomes `pending`).
- Optional: **Users → Edit → Notifications** for admin-managed overrides (support desk).

### Dispatch (Layer 1 + 2)

```
On hook event:
  1. Build normalized payload from hook arguments (event, site, timestamp, typed fields).
  2. Layer 1: send to each active Notification shoutrr_url for this event.
  3. Layer 2: collect recipients:
       a. Active user subscriptions for event + channel
       b. Active role subscriptions → expand to member users
       c. Deduplicate by email; apply per-user opt-out
  4. For each recipient: send via site mail (preferred) or Shoutrrr SMTP URL from mail config.
  5. Log failures; do not block the hook chain on send errors.
```

Layer 2 must not require a per-user Shoutrrr URL. Use `internal/mail` with the site From address once mail configuration is complete.

### RBAC

| Permission | Purpose |
|------------|---------|
| `core.admin.notifications.read` | View Layer 1 destinations (exists) |
| `core.admin.notifications.write` | Create/edit/delete Layer 1 destinations |
| `core.admin.notification-subscriptions.read` | View role default subscriptions |
| `core.admin.notification-subscriptions.write` | Edit role default subscriptions |
| (none) | Users manage their own Layer 2 subscriptions via account UI |

### Relationship to groups

- **Groups** = visibility and membership (frontend/backend). Do not use groups for notification routing.
- **Roles** = capability + **default notification subscriptions** (Layer 2).

---

## Event catalog

Hooks that should eventually appear in Layer 1 picker and/or Layer 2 subscription UI.

### User lifecycle (Layer 1 + 2)

| Event | Fired when | Key arguments |
|-------|------------|---------------|
| `onUserSignup` | Public registration creates account | `user_id`, `username`, `email` |
| `onUserAfterLogin` | Successful login | `user_id`, `username`, `email`, `context` |
| `onUserVerified` | Email verification completes | `user_id`, `username`, `email` |
| `onUserLocked` | Admin locks account | `user_id`, `username`, `email` |
| `onUserLogout` | Session cleared | `user_id`, `username`, `email`, `context` |

### Content / editorial (Layer 2 priority; Layer 1 optional)

| Event | Fired when | Key arguments |
|-------|------------|---------------|
| `onItemAfterSave` | Item saved (admin or frontend) | `item_id`, `item`, `is_new` |
| `onItemBeforeSave` | Before item persist | `item`, `is_new` |
| `onCommentAfterSave` | Comment saved | `item_id`, `comment_id`, `approved` |

Suggested role defaults:

- **Editor / Publisher:** `onItemAfterSave` when status → `pending` (filter).
- **Author (own content):** user subscription when their item is approved or commented on (filter `author_id`).

### System (Layer 1 only by default)

Route/render hooks (`onBeforeRoute`, `onAfterRender`, …) remain extension-oriented; omit from default subscription UI unless an admin explicitly wants them.

---

## Implementation phases

### Phase A — Layer 1 polish (small)

- [x] `Notification` + `NotificationEvent` models
- [x] Admin CRUD + Shoutrrr send for four user hooks
- [ ] Expand Layer 1 event picker to content/comment hooks
- [ ] Message templates per event
- [ ] Update `.info/TODO.md` status (done when shipped)

### Phase B — Layer 2 core

- [x] `notification_subscriptions` model + migration
- [x] Extend `DispatchEvent` to resolve user + role recipients
- [x] Send via site mail to subscriber emails
- [x] Account UI: per-user event checklist
- [x] Role admin UI: default subscriptions per role
- [x] Dedup + opt-out rules (user overrides role)

### Phase C — Layer 2 filters & polish

- [ ] `filters_json` (category, status transition, author=self)
- [ ] Admin view of effective subscriptions on user form
- [ ] Delivery log / retry (optional)
- [ ] In-app notification inbox (`channel: in_app`)

---

## Related docs

- `.info/specs/event_hooks.md` — hook names, arguments, extension protocol
- `.info/specs/user_group_role.md` — roles vs groups; role admin UI
- `internal/notifications/` — current Layer 1 implementation
- `.info/TODO.md` — tracked checklist
