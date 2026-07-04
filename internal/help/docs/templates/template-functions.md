# Template Functions

Cannon renders HTML with Go **`html/template`**. Every site, admin, and extension template can call **Sprig** helpers plus Cannon-specific functions listed below.

Functions are merged at render time. When a name exists in both Sprig and Cannon, **Cannon wins** for: `add`, `sub`, `mul`, `div`, `min`, `dict`, and `initials`.

## Where functions are available

| Context | Sprig | Cannon core | Frontend extras | Admin extras | CSRF |
|---------|:-----:|:-----------:|:---------------:|:------------:|:----:|
| Public site (`default/` templates) | Yes | Yes | Yes | — | Yes |
| Admin panel (`admin/` templates) | Yes | Yes | — | Yes | Yes |
| Extension embedded HTML (in-process) | Yes | — | — | — | If wired by extension |
| Extension HTML rendered by Cannon | Yes | Yes | Per route | — | Yes |

Extension processes that render their own templates with `extension.Templates` receive **Sprig only** unless the extension passes extra functions via `WithFuncs`. Site overrides under `{template_dir}/extension/…` still use the extension's function map when the extension executes them.

## Cannon core functions

Available on **public site and admin** templates (from `templateengine.FuncMap`).

### Block spaces

| Function | Signature | Description |
|----------|-----------|-------------|
| `space` | `space "name"` | Renders the HTML for a block space. Returns trusted `template.HTML`. No-op when block rendering is unavailable (admin previews). |
| `lenspace` | `lenspace "name"` | Returns the count of active blocks assigned to the space for the current viewer. Use with `if gt (lenspace "header") 0`. |

### Math (integer-friendly)

Cannon replaces Sprig's float-oriented math for pagination and list UI. Arguments accept integers, int64, and other numeric types via internal conversion.

| Function | Example | Result |
|----------|---------|--------|
| `add` | `{{add 1 2}}` | `3` |
| `sub` | `{{sub 5 2}}` | `3` |
| `mul` | `{{mul 3 4}}` | `12` |
| `div` | `{{div 10 3}}` | `4` (ceiling division; `0` if divisor is `0`) |
| `min` | `{{min 3 7}}` | `3` |

### Maps and strings

| Function | Signature | Description |
|----------|-----------|-------------|
| `dict` | `dict "key1" val1 "key2" val2` | Builds a `map[string]any` for passing structured data to partials. Keys must be strings; odd argument count errors at render time. |
| `queryEscape` | `queryEscape "a b"` | URL-query-escapes a string (`url.QueryEscape`). |
| `initials` | `initials "Jane" "Doe"` | First letter of each non-empty argument, uppercased, max two characters. Returns `"?"` when empty. |

## Frontend-only functions

Added when rendering the public site (`internal/server/server.go`).

| Function | Signature | Description |
|----------|-----------|-------------|
| `menu` | `menu "main"` | Returns top-level menu items for the named menu. Each item has `Name`, `Href`, `Class`, `Target`, and optional nested `Children` (same shape). Menus are configured under **Admin → Menus**. |
| `homeURL` | `homeURL` | Path of the site default route (from **Admin → Routes**). |
| `isDefaultRoute` | `isDefaultRoute` | `true` when the current request matched the default route. |
| `routeName` | `routeName` | Display name of the matched route, or empty. |
| `routePath` | `routePath` | Path pattern of the matched route (e.g. `/`, `/content/item/*`). |
| `routeController` | `routeController` | Controller id for controller routes (e.g. `content`, `auth`). |
| `routeAction` | `routeAction` | Controller action id (e.g. `index`, `item`). |
| `richText` | `richText .Category.Description` | Renders stored HTML or Markdown from the admin editor as safe HTML (category descriptions, item intros in listings). |
| `siteName` | `siteName` | Site display name from `sites.json`. |
| `year` | `year` | Current calendar year (for footers). |

Example:

```html
<footer>
  <p>&copy; {{year}} {{siteName}}</p>
  {{range menu "footer"}}
  <a href="{{.Href}}">{{.Name}}</a>
  {{range .Children}}
  <a href="{{.Href}}">{{.Name}}</a>
  {{end}}
  {{end}}
</footer>
```

## Admin-only functions

Added when rendering admin templates (`internal/admin/handler.go`).

### Localization and URLs

| Function | Signature | Description |
|----------|-----------|-------------|
| `lang` | `lang "key"` or `lang "key" "name" "value" …` | Looks up a translation from the locale manager. Returns the key when no translation exists. Optional key/value pairs substitute placeholders. Append `?tp=1` to the URL (or pass `"tp" "1"` as the first pair) to show the key in parentheses after the translated text, e.g. `Sign in (nav.sign_in)`. |
| `internalHelpURL` | `internalHelpURL "templates" "frontend-templates"` | Admin URL for a built-in help article (`/admin/help/{folder}/{slug}`). |
| `helpURL` | `helpURL "extension-name" "article/path"` | Admin URL for an extension-provided help article. |
| `siteURL` | `siteURL .Host` | Full frontend URL for a site host entry (from site configuration). |
| `siteAdminURL` | `siteAdminURL .Host` | Admin URL for a site host (`{frontend}/admin`). |
| `siteHostLabel` | `siteHostLabel .Host` | Hostname label extracted from a site URL string. |

### List and sort helpers

Used by admin list pages for pagination and column sorting links.

| Function | Signature | Description |
|----------|-----------|-------------|
| `sortLink` | `sortLink "/admin/users" .Page .Sort .Dir "username"` | Builds a list URL with toggled sort direction for column `username`. |
| `listQuery` | `listQuery .Page .Sort .Dir` | Query string (`?page=2&sort=name&dir=asc`) preserving list state. Empty when defaults only. |
| `listQueryAmp` | `listQueryAmp .Page .Sort .Dir` | Same as `listQuery` but prefixed with `&` for appending to an existing query. |

When a list page sets a space filter, extra query parameters (for example `space=footer`) are preserved automatically.

### Model display helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `containsUint` | `containsUint .SelectedIDs 3` | Returns whether `uint` slice contains the id. |
| `uintPtrEq` | `uintPtrEq .ParentID 5` | Compares a `*uint` to a `uint`. |
| `joinRoleNames` | `joinRoleNames .Roles` | Comma-separated role names from a `[]models.Role`. |
| `joinGroupNames` | `joinGroupNames .Groups` | Comma-separated group display names. |
| `groupName` | `groupName .Name` | Admin label for a group (`Public` for the public group). |

## CSRF helpers

Available on authenticated admin and frontend forms when a session is present.

| Function | Description |
|----------|-------------|
| `csrfField` | Renders a hidden `<input name="_csrf" …>` for HTML forms. |
| `csrfToken` | Raw token string (for JavaScript or custom markup). |

Example:

```html
<form method="post">
  {{csrfField}}
  …
</form>
```

## Sprig function reference

Cannon ships [Sprig v3](https://github.com/Masterminds/sprig) — a large helper library for Go templates. Use Sprig in frontend, admin, and extension HTML.

Official interactive docs: [masterminds.github.io/sprig](https://masterminds.github.io/sprig/)

### Quick examples

```html
{{upper .Title}}
{{default "Untitled" .Title}}
{{if empty .Items}}<p>No items.</p>{{end}}
{{date "January 2, 2006" now}}
{{join ", " .Tags}}
{{toJson .Data}}
```

### String functions

| Function | Description |
|----------|-------------|
| `trim`, `trimAll`, `trimPrefix`, `trimSuffix`, `trimall` | Remove whitespace or cut prefixes/suffixes. |
| `upper`, `lower`, `title`, `untitle`, `swapcase` | Change letter case. |
| `repeat`, `substr`, `nospace` | Repeat, slice, or strip spaces from strings. |
| `replace`, `plural` | Replace substrings; simple English pluralization. |
| `snakecase`, `camelcase`, `kebabcase` | Case conversions for identifiers. |
| `wrap`, `wrapWith`, `indent`, `nindent` | Wrap lines or indent blocks of text. |
| `abbrev`, `abbrevboth`, `trunc`, `initial` | Shorten strings (note: `initials` is Cannon's helper, not Sprig's `initial`). |
| `contains`, `hasPrefix`, `hasSuffix`, `quote`, `squote` | Test or quote strings. |
| `cat`, `split`, `splitList`, `splitn`, `join` | Concatenate or split strings. |
| `sha1sum`, `sha256sum`, `sha512sum`, `adler32sum` | Hash strings. |

### List functions

| Function | Description |
|----------|-------------|
| `list`, `append`, `prepend`, `push`, `mustAppend`, `mustPrepend`, `mustPush` | Build and extend lists. |
| `first`, `last`, `rest`, `initial`, `mustFirst`, `mustLast`, `mustRest`, `mustInitial` | Access list ends and tails. |
| `compact`, `mustCompact`, `uniq`, `mustUniq`, `without`, `mustWithout` | Filter list entries. |
| `chunk`, `mustChunk`, `slice`, `mustSlice`, `reverse`, `mustReverse` | Reshape lists. |
| `sortAlpha`, `shuffle` | Order or randomize. |
| `concat` | Concatenate multiple lists. |

### Dictionary functions

| Function | Description |
|----------|-------------|
| `get`, `set`, `unset`, `hasKey`, `pluck`, `dig`, `keys`, `values`, `pick`, `omit` | Read and write map keys. |
| `merge`, `mergeOverwrite`, `mustMerge`, `mustMergeOverwrite` | Combine maps (Sprig `dict` is overridden by Cannon's `dict`). |
| `deepCopy`, `mustDeepCopy`, `deepEqual` | Clone or compare nested structures. |

### Math and logic

| Function | Description |
|----------|-------------|
| `add1`, `add1f`, `addf`, `subf`, `mulf`, `divf`, `max`, `maxf`, `minf`, `mod`, `ceil`, `floor`, `round`, `biggest` | Float and integer math (`add`, `sub`, `mul`, `div`, `min` use Cannon's versions). |
| `float64`, `int`, `int64`, `toDecimal` | Type conversion. |
| `default`, `empty`, `coalesce`, `all`, `any`, `ternary` | Defaults and conditionals. |
| `fail` | Abort template execution with an error message. |

### Date and time

| Function | Description |
|----------|-------------|
| `now` | Current time. |
| `date`, `dateInZone`, `date_in_zone` | Format a timestamp with a Go layout string. |
| `dateModify`, `date_modify`, `mustDateModify`, `must_date_modify` | Add duration to a date. |
| `ago`, `until`, `untilStep` | Relative time helpers. |
| `htmlDate`, `htmlDateInZone` | Format for HTML date inputs. |
| `toDate`, `mustToDate`, `unixEpoch`, `duration`, `durationRound` | Parse or convert times. |

### Encoding and serialization

| Function | Description |
|----------|-------------|
| `toJson`, `toPrettyJson`, `toRawJson`, `mustToJson`, `mustToPrettyJson`, `mustToRawJson` | Encode values as JSON. |
| `fromJson`, `mustFromJson` | Parse JSON strings. |
| `toString`, `toStrings` | String conversion. |
| `b64enc`, `b64dec`, `b32enc`, `b32dec` | Base32/Base64. |
| `atoi` | Parse integer from string. |

### Regular expressions

| Function | Description |
|----------|-------------|
| `regexMatch`, `regexFind`, `regexFindAll`, `regexSplit` | Match and extract. |
| `regexReplaceAll`, `regexReplaceAllLiteral` | Replace matches. |
| `regexQuoteMeta` | Escape regex metacharacters. |
| `mustRegex*` variants | Same as above but fail the template on error. |

### Paths, files, and environment

| Function | Description |
|----------|-------------|
| `base`, `dir`, `ext`, `clean`, `isAbs` | Path manipulation (Sprig names). |
| `osBase`, `osDir`, `osExt`, `osClean`, `osIsAbs` | OS-specific path helpers. |
| `env`, `expandenv` | Read environment variables. |

### Network, crypto, and misc

| Function | Description |
|----------|-------------|
| `uuidv4` | Random UUID string. |
| `randInt`, `randAlpha`, `randAlphaNum`, `randAscii`, `randBytes`, `randNumeric` | Random values. |
| `semver`, `semverCompare` | Semantic version parsing and comparison. |
| `urlParse`, `urlJoin` | URL building (use Cannon's `queryEscape` for query encoding in admin templates). |
| `getHostByName` | DNS lookup. |
| `bcrypt`, `htpasswd`, `encryptAES`, `decryptAES`, `derivePassword` | Password and crypto utilities. |
| `genCA`, `genPrivateKey`, `genSelfSignedCert`, `genSignedCert`, and `*WithKey` variants | TLS certificate generation. |
| `buildCustomCert` | Build a cert from PEM material. |
| `seq`, `tuple` | Generate sequences or fixed tuples. |
| `typeOf`, `kindOf`, `typeIs`, `typeIsLike`, `kindIs` | Reflection helpers. |
| `hello` | Sprig demo function (returns `"Hello!"`). |

## Standard Go template syntax

Sprig and Cannon functions sit on top of Go's built-in template actions:

```html
{{if .Show}}…{{else}}…{{end}}
{{range .Items}}…{{end}}
{{with .User}}…{{end}}
{{template "partial" .}}
{{define "partial"}}…{{end}}
```

Pipeline syntax passes the left value as the last argument to the right function:

```html
{{.Title | upper}}
{{.Count | add 1}}
```

## Related topics

- **Help → Templates → Frontend Templates** — layout, block spaces, menus, and override workflow.
- **Admin → Templates** — browse and override built-in `default/` and `admin/` HTML.
- **EXTENSIONS.md** (repository) — extension template overrides under `extension/`.
