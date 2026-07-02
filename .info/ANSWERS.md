Below are questions you had, my answers all start with "A-" 


Scope and priorities
What is the MVP? Should the first deliverable be install + single-site admin + basic routing/templating, or do you want extensions, multisite, and OAuth wired in from day one?

A - All of the above, if there is more information needed,ask before starting each section.  

Content model is undefined. The table list covers users, routes, menus, and extensions, but there are no Pages, Posts, Blocks, or similar tables. What content types should Cannon manage in v1, and how do they relate to routes and templates?

A - I want to establisha  framework first, and then we'll develop the content model with further instruction

"Extensions and Plugins" — are plugins a separate concept from extensions, or is that the same thing with two names?

A - That is a mistake, there are just extensions currently

Configuration and multisite
sites.json structure — the example in AGENT.MD looks like draft JSON (missing commas, inconsistent quoting). Should I treat this as the intended schema and formalize it, or do you have a corrected version?

A - This is a draft schema, make necessary corrections and suggest improvements if you find them.

Site resolution — is matching purely by Host header (e.g. example.com → site2), or should path-based multisite (e.g. /site1/...) also be supported?

A - Match by host header, however, to support development, Also check X-Host header and if set use that 

Per-site vs shared resources:

One database per site, or a shared database with a site_id column?
Users global across sites, or scoped per site?
Extensions: one process per (extension, site) pair (implied by the socket hash), or one global process that receives --site per request?
host in sites.json — should this be a full URL (https://example.com), hostname only (example.com), or support multiple hosts per site?

A - One database per site, same for users, extensions are one per server but the socket and site id cli calls scope them to each site.   Hosts shold be the full url so that we know if it supports https or not

Installation
Install wizard step 1 — for a fresh install, should we configure exactly one site, or allow defining multiple sites upfront?

A - Configure just one site

Minimum fields for step 1 — site name, host, database URL, template/assets/tmp/language dirs, and extensions dir: is that the full set, or are some optional with sensible defaults (e.g. SQLite file in the working directory)?

A - Add database type with a drop down for the specific fields needed, default to sqlite and calculate the sqlite db name from the site name field (remove spaces, normalize, etc)

After install, is /install permanently disabled, or should it remain accessible for adding sites later (via admin instead)?

A - Disabled in sites.json, able to be re-ran by flipping the boolean value

Database and schema
Migrations — auto-migrate via GORM on startup, or explicit migration files (e.g. golang-migrate)?

A- Use Gorm migration

Missing FKs and scope — should tables like Routes, Menu, Extensions, and Users include a site_id? The current schema doesn't show one.

A - One site per database, site_id not necessary

Profiles system — is this for end-user custom fields (like Drupal profile fields), admin-only metadata, or both? What field types should ProfileFields.type support in v1 (text, textarea, select, date, etc.)?

A - Both, start with textarea, input, select, date, additionally specific types of input like email, password,etc. 

Extensions
HTTP over Unix socket — confirm the parent CMS dials the extension's socket and sends normal HTTP requests (method, path, headers, body). Is that the contract?  Yes the parent dials the socket and passes the request information though follow the specs/extensions.md for the paths to call.

A - Extension discovery — how do extensions get registered?  They are manually added right now, at some point we will allow for pulling them into the server via a url.

Scan the extensions directory for binaries? 
Upload via admin?
Manual DB entry pointing to a binary path?

A - For now scan the extensions directory for binaries and also support manual db entry

Request handler contract — what can an extension return from /request_handler?

Modify headers/body and continue?
Short-circuit with a response (redirect, 403, custom HTML)?
Is there a defined request/response JSON schema?

A - Request Handler Contract, the return can either be an updated request body http.Request and or a response like a redirect 
 The response should ideally be serialized version of http.Request/response so we can convert and send on the wire.

Page handler contract — what does /page_handler return? Full HTML document, a fragment for embedding in a layout, or JSON with template data?

A-A fragment to be integrated with ther est of the template.


Block positions — what should the template syntax look like? e.g. {{block "sidebar"}}, {{.Blocks.sidebar}}, or a custom template func?

A -  {{block "sidebar"}}

Extension execution order — when multiple extensions expose request capability, do they all run in sort order? Can one extension stop the chain?

A - They all run in sort order, when a response that is ending like a redirect occurs then the chain should be stopped

Extensions table — should it store binary_path, sort, and site_id in addition to name, socket, and status?

Site id is inferred from the current scop, the path should be a combination of the socket hash and the extensions dir in the sites.json so not necessary to store it, just calculate it.

Routing
Route types — please clarify each:

Url — external redirect?  A - or can be a existing url on the site ie https://example.com/foo/bar
Extension — delegates to an extension's page handler? - A - Correct
Local File — serve a static file from assets_dir? - Yes - Correct ie foo.pdf in the assets dir
Controller — built-in Go handler (which controllers exist in v1)? - 

A - Yes, however, these will be front end controllers and we'll adress that more fully after this initial MVP.

Route matching precedence — if /contact, /contact/*, and a regex route could all match, which wins?

A - The most specific match wins.

Menus — are menus purely navigational (admin-managed links to routes), or do they drive routing too?

A - Naviational we will need a custom template function to get menu idata so that it can be rendered.

User system and auth
Session storage — file-based sessions under each site's tmp_dir? Cookie name/session format preferences?

A - File based, have sane defaults in a sites.json config area for default key/tokens etc.

Shared session for extensions — the spec says extensions may import the user package with request context via a shared session location in tmp. Should extensions read session files directly, or should the CMS pass user context in extension HTTP calls?

A - Pass in user context

Goth providers — which OAuth providers for v1 (Google, GitHub, etc.)? One authenticator row per provider?

All of the providers should be added as inactive in the table so that they can be turned on dynamically and configured. 

Admin access — is any authenticated user allowed in /admin, or do we need roles/permissions beyond the profile system?

A - We will need user groups and roles, 

Admin panel
Initial admin screens — which CRUD areas are in v1?

Users, Routes, Menus, Extensions, Authenticators, Profiles, Languages — all of them, or a subset?
"Button cased" in admin_design.md — did you mean Title Case (e.g. "Save Changes", "Add User")?

A - Title Casing ie Save Change etc.
A - All of the above areas should be added.  

Turbo usage — Turbo Drive for full-page navigation, Turbo Frames for inline table/form updates, or both?

A - Both

Language editing in admin — inline editing of .ini key/value pairs per locale, or file upload/download?

A - Inline so that a admin could quickly update the language file.

Templating and i18n
Template layout — is there a base layout with blocks/partials (e.g. default/layout.html wrapping default/page.html), or flat templates per route?

A - Use a base template and then tags for page info ie .Main and blocks for block rendering ie {{block "header"}} etc. Additionally a {{template path/to/file.html}} that follows  a precendence of the current template and then the default template file location match.

Locale selection — cookie, URL prefix (/en-US/...), Accept-Language, or admin-configured default per site?

A - Cookie/Accept-Language with Cookie overriding Accept Language

lang.Fmt placeholders — the example uses named keys ("Username","Dave"). Should this support positional args, multiple placeholders, and pluralization, or keep it simple for v1?

A - Keep the dynamic key/values

en-US-*.ini files — what is the suffix convention? e.g. en-US-admin.ini for admin strings vs en-US-site.ini for frontend?

A - Correct

Technical choices (unspecified but needed)
HTTP router/framework — preference among stdlib net/http, Chi, Echo, Gin, or no preference?

A - stdlib, use middlewares we will probably extend support here later.

Go module path — should this be github.com/rob121/cannon?

A - yes

Deployment target — single binary on a VPS, Docker, or both? That affects default paths in sites.json.

A - single binary, however, the sites.json directories should be separate (created at /install if not exist with warning if they do) A separate pacakge e.g github.com/rob121/cannon-template-clean  may hold a template that can be loaded into the template dir


Contradictions to resolve
Topic	Tension
Extensions dir - Global in sites.json, but socket hash uses site name — per-site instances?

A - Extensions dir is globabl, because we should calculate a path based on site name and then hash for the socket 

Extensions table  - No site_id or binary_path columns

A - yes, because those are inferred/calcualated

Content CMS features listed, but no content tables defined

A - we will work on that feature separately at a different time.

Install vs multisite - Install creates one config; multisite implies many

A - one site, then configurable in the admin to add more / switch sites.

