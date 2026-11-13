package admin

// extensionAdminCSP limits what extension admin markup can do in the sandboxed iframe.
const extensionAdminCSP = "default-src 'none'; script-src 'unsafe-inline' 'self'; style-src 'unsafe-inline' 'self'; connect-src 'self' ws: wss:; img-src 'self' data:; frame-src 'self'; child-src 'self'; base-uri 'none'; form-action 'self'"
