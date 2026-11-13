package admin

// extensionAdminCSP documents a stricter policy for sandboxed extension markup if we ever
// attach CSP via iframe srcdoc meta tags. Do not send this on the outer admin HTTP response
// (it breaks Bootstrap, Turbo, and fonts loaded from the admin layout).
const extensionAdminCSP = "default-src 'none'; script-src 'unsafe-inline' 'self'; style-src 'unsafe-inline' 'self'; connect-src 'self' ws: wss:; img-src 'self' data:; frame-src 'self'; child-src 'self'; base-uri 'none'; form-action 'self'"
