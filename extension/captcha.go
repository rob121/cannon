package extension

import "strings"

const (
	// ProviderAny means use the site's configured active captcha extension.
	ProviderAny = "any"

	// CaptchaContextLogin is rendered on frontend and admin login forms.
	CaptchaContextLogin = "login"
	// CaptchaContextRegister is rendered on self-registration forms.
	CaptchaContextRegister = "register"
	// CaptchaContextComment is rendered on item comment forms.
	CaptchaContextComment = "comment"
	// CaptchaContextForm is a generic protected form context.
	CaptchaContextForm = "form"
)

// CaptchaProviderInfo describes a captcha extension for Cannon and admin UI.
// GET /captcha returns this document. Only public client values belong in Client
// (for example a Turnstile or reCAPTCHA site key). Secrets stay in extension
// configuration and are never returned here.
type CaptchaProviderInfo struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Contexts []string          `json:"contexts,omitempty"`
	Client   map[string]string `json:"client,omitempty"`
}

// CaptchaRenderResult is returned from POST /captcha/render.
type CaptchaRenderResult struct {
	HTML      string `json:"html"`
	HeadHTML  string `json:"head_html,omitempty"`
	FieldName string `json:"field_name"`
}

// CaptchaVerifyResult is returned from POST /captcha/verify.
type CaptchaVerifyResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// CaptchaRegistration wires captcha handlers for the captcha capability.
type CaptchaRegistration struct {
	// ProviderInfo serves GET /captcha. When nil, RegisterCaptcha synthesizes
	// a minimal response from extension Info.
	ProviderInfo func(req WireRequest) (CaptchaProviderInfo, error)
	// Render serves POST /captcha/render.
	Render func(req WireRequest) (CaptchaRenderResult, error)
	// Verify serves POST /captcha/verify.
	Verify func(req WireRequest) (CaptchaVerifyResult, error)
}

// CaptchaContext returns the protected form context from a wire request.
func CaptchaContext(req WireRequest) string {
	return strings.TrimSpace(req.CaptchaContext)
}

// CaptchaAction returns an optional provider-specific action label (for example
// reCAPTCHA v3 actions).
func CaptchaAction(req WireRequest) string {
	return strings.TrimSpace(req.CaptchaAction)
}

// CaptchaToken returns the submitted captcha response token.
func CaptchaToken(req WireRequest) string {
	return strings.TrimSpace(req.CaptchaToken)
}

// CaptchaRemoteIP returns the client IP Cannon observed for the protected request.
func CaptchaRemoteIP(req WireRequest) string {
	return strings.TrimSpace(req.CaptchaRemoteIP)
}
