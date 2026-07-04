# Configuration Field Types

**System → Configuration** renders settings with [JSON Forms](https://jsonforms.io/). Each section has a JSON **schema** (field definitions) and **UI schema** (layout and widget hints).

Global sections are defined in the Cannon repo under `internal/settings/definitions/` (for example `general.json`, `content.json`, `mail.json`). Extension settings use the same format via `OnConfiguration`.

## Standard fields

| Schema type | UI hint | Rendered as |
|-------------|---------|-------------|
| `string` | — | Text input |
| `integer` | — | Number input |
| `string` + `enum` | — | Dropdown |
| `integer` + `enum` | — | Dropdown |
| `boolean` | — | Enable/disable toggle |
| `string` | `"options": {"multi": true}` | Textarea |

Example boolean and enum properties:

```json
"allow_login": {
  "type": "boolean",
  "title": "Allow Login",
  "default": true
},
"log_level": {
  "type": "string",
  "title": "Log Level",
  "enum": ["debug", "info", "warning", "error", "none"]
}
```

Example textarea control (SEO section):

```json
{
  "type": "Control",
  "scope": "#/properties/site_meta_description",
  "options": {"multi": true}
}
```

String properties may include a `"default"` value; Cannon uses it as the input placeholder when the saved value is empty.

## Category dropdown

Use a **category dropdown** when a setting should store a category ID from the site taxonomy.

**Schema:**

```json
"listing_category_id": {
  "type": ["integer", "null"],
  "format": "category",
  "title": "Listing Category",
  "description": "Default category for public listings."
}
```

**UI schema** (either schema `format` or this option triggers the widget):

```json
{
  "type": "Control",
  "scope": "#/properties/listing_category_id",
  "options": {"format": "category"}
}
```

Cannon replaces the number input with a `<select>` of active categories. Empty selection stores `null` when the type includes `"null"`, otherwise `0`. If a saved category is inactive, it still appears as `Category #ID`.

## Dynamic dropdowns

Some global fields populate their `enum` at render time from live site data:

- **Frontend Theme** / **Admin Theme** — folders under the site template directory
- **Active Captcha Extension** — installed captcha extensions

Extension configuration can return the same JSON Forms shape from `GET /configuration`; Cannon renders it in **Configuration → Extensions**.

## Content section extras

The **Content** global section also includes admin-only fields injected outside JSON Forms:

- **Author Profiles** — profile schema picker for item author pages. Frontend content permissions are managed under **Users → Roles** (`core.content.frontend.*`).
- **Author Profile** — profile schema dropdown for author pages

These are handled in code (`internal/content/config_form.go`), not in the section JSON file.

## Nullable types

JSON Schema allows `"type": ["integer", "null"]` for optional values. Cannon normalizes this for the form renderer; category fields and save handling respect `null` when present in the type list.

## Adding a global section

1. Add `internal/settings/definitions/{id}.json` with `title`, `schema`, and `ui_schema`.
2. Rebuild Cannon — the section appears under **Configuration → Global**.
3. Saved values are stored per site in the `settings` table (`scope = global`).

See [EXTENSIONS.md](https://github.com/rob121/cannon/blob/main/EXTENSIONS.md) in the repository for the full extension configuration wire format and examples.
