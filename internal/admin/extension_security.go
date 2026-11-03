package admin

// extensionAdminCSP limits what extension admin markup can do in the sandboxed iframe.
const extensionAdminCSP = "default-src 'none'; frame-src 'self'; child-src 'self'; base-uri 'none'; form-action 'self'"
