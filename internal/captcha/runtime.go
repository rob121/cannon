package captcha

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/httpreq"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/user"
)

var (
	// ErrVerificationFailed is returned when captcha verification fails.
	ErrVerificationFailed = errors.New("captcha verification failed")
	// ErrRequired is returned when captcha was required but missing.
	ErrRequired = errors.New("captcha is required")

	captchaTagRE = regexp.MustCompile(`(?i)<captcha([^>]*?)(?:/>|></captcha>)`)
	attrRE       = regexp.MustCompile(`(?i)([a-z_]+)\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'>]+))`)
)

const sessionFieldPrefix = "captcha:field:"

// ExpandHTML replaces <captcha> placeholders in rendered HTML.
func ExpandHTML(ctx context.Context, body string) (string, error) {
	if !captchaTagRE.MatchString(body) {
		return body, nil
	}
	enabled, err := settings.CaptchaEnabled(ctx)
	if err != nil || !enabled {
		return stripPlaceholders(body), nil
	}
	if skip, err := shouldSkipAuthenticated(ctx); err != nil {
		return body, err
	} else if skip {
		return stripPlaceholders(body), nil
	}
	mgr, ok := extensions.FromContext(ctx)
	if !ok || mgr == nil {
		return stripPlaceholders(body), nil
	}
	active, err := settings.CaptchaActiveExtension(ctx)
	if err != nil {
		return body, err
	}
	if name := strings.TrimSpace(active); name != "" && !mgr.CaptchaExtensionAvailable(name) {
		return stripPlaceholders(body), nil
	}
	r, ok := httpreq.FromContext(ctx)
	if !ok || r == nil {
		return body, fmt.Errorf("captcha expand: request missing from context")
	}
	userCtx, _ := user.RequestUser(ctx)
	var heads []string
	out := captchaTagRE.ReplaceAllStringFunc(body, func(tag string) string {
		attrs := parseTagAttrs(tag)
		contextName := strings.TrimSpace(attrs["context"])
		if contextName == "" {
			contextName = CaptchaContextForm
		}
		provider := attrs["provider"]
		if provider == "" {
			provider = attrs["type"]
		}
		extName, err := mgr.ResolveCaptchaExtension(ctx, provider)
		if err != nil {
			return ""
		}
		render, err := mgr.InvokeCaptchaRender(ctx, extName, contextName, attrs["action"], r, userCtx)
		if err != nil {
			return `<p class="text-danger small mb-0">Captcha could not be loaded.</p>`
		}
		if strings.TrimSpace(render.HeadHTML) != "" {
			heads = append(heads, render.HeadHTML)
		}
		rememberFieldName(ctx, contextName, render.FieldName)
		return wrapWidget(contextName, render)
	})
	out = injectHeadHTML(out, heads)
	return out, nil
}

func wrapWidget(contextName string, render extension.CaptchaRenderResult) string {
	widget := strings.TrimSpace(render.HTML)
	hidden := fmt.Sprintf(`<input type="hidden" name="captcha_context" value="%s">`, html.EscapeString(contextName))
	return widget + hidden
}

func injectHeadHTML(body string, heads []string) string {
	if len(heads) == 0 {
		return body
	}
	joined := strings.Join(heads, "\n")
	idx := strings.Index(strings.ToLower(body), "</head>")
	if idx < 0 {
		return joined + body
	}
	return body[:idx] + joined + "\n" + body[idx:]
}

func stripPlaceholders(body string) string {
	return captchaTagRE.ReplaceAllString(body, "")
}

func parseTagAttrs(tag string) map[string]string {
	attrs := map[string]string{}
	m := captchaTagRE.FindStringSubmatch(tag)
	if len(m) < 2 {
		return attrs
	}
	for _, match := range attrRE.FindAllStringSubmatch(m[1], -1) {
		if len(match) < 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(match[1]))
		val := match[3]
		if val == "" {
			val = match[4]
		}
		if val == "" {
			val = match[5]
		}
		attrs[key] = strings.TrimSpace(val)
	}
	return attrs
}

func shouldSkipAuthenticated(ctx context.Context) (bool, error) {
	skip, err := settings.CaptchaSkipAuthenticated(ctx)
	if err != nil || !skip {
		return false, err
	}
	svc, err := user.FromContext(ctx)
	if err != nil {
		return false, nil
	}
	_, ok := svc.CurrentID()
	return ok, nil
}

func rememberFieldName(ctx context.Context, contextName, fieldName string) {
	svc, err := user.FromContext(ctx)
	if err != nil {
		return
	}
	_ = svc.SetSessionValue(sessionFieldPrefix+contextName, fieldName)
}

func tokenFieldName(ctx context.Context, r *http.Request, contextName string) string {
	if svc, err := user.FromContext(ctx); err == nil {
		if raw, ok := svc.SessionValue(sessionFieldPrefix + contextName); ok {
			if name, ok := raw.(string); ok && strings.TrimSpace(name) != "" {
				return name
			}
		}
	}
	if token := strings.TrimSpace(r.FormValue("captcha_token")); token != "" {
		return "captcha_token"
	}
	return ""
}

// AppliesToSubmit reports whether the current request should run captcha
// verification for this context. This lets generic extension data routes opt in
// by rendering a captcha placeholder without forcing every extension POST to use
// captcha.
func AppliesToSubmit(ctx context.Context, r *http.Request, formContext string) (bool, error) {
	required, err := Required(ctx, formContext)
	if err != nil || !required {
		return false, err
	}
	if rememberedFieldName(ctx, formContext) != "" {
		return true, nil
	}
	if err := parseFormPreserveBody(r); err != nil {
		return true, err
	}
	if strings.TrimSpace(r.FormValue("captcha_context")) != "" {
		return true, nil
	}
	if strings.TrimSpace(r.FormValue("captcha_token")) != "" {
		return true, nil
	}
	return false, nil
}

func rememberedFieldName(ctx context.Context, contextName string) string {
	if svc, err := user.FromContext(ctx); err == nil {
		if raw, ok := svc.SessionValue(sessionFieldPrefix + contextName); ok {
			if name, ok := raw.(string); ok {
				return strings.TrimSpace(name)
			}
		}
	}
	return ""
}

func clearRememberedFieldName(ctx context.Context, contextName string) {
	if svc, err := user.FromContext(ctx); err == nil {
		_ = svc.ClearSessionValue(sessionFieldPrefix + contextName)
	}
}

// Required reports whether captcha verification should run for a form context.
func Required(ctx context.Context, formContext string) (bool, error) {
	enabled, err := settings.CaptchaEnabled(ctx)
	if err != nil || !enabled {
		return false, err
	}
	if skip, err := shouldSkipAuthenticated(ctx); err != nil {
		return false, err
	} else if skip {
		return false, nil
	}
	name, err := settings.CaptchaActiveExtension(ctx)
	if err != nil {
		return false, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	mgr, ok := extensions.FromContext(ctx)
	if !ok || mgr == nil {
		return false, nil
	}
	return mgr.CaptchaExtensionAvailable(name), nil
}

// VerifySubmit validates captcha for a protected form submission.
func VerifySubmit(ctx context.Context, r *http.Request, formContext string) error {
	required, err := Required(ctx, formContext)
	if err != nil {
		return err
	}
	if !required {
		return nil
	}
	if err := parseFormPreserveBody(r); err != nil {
		return err
	}
	contextName := strings.TrimSpace(r.FormValue("captcha_context"))
	if contextName == "" {
		contextName = formContext
	}
	mgr, ok := extensions.FromContext(ctx)
	if !ok || mgr == nil {
		return fmt.Errorf("captcha extension manager unavailable")
	}
	extName, err := mgr.ResolveCaptchaExtension(ctx, ProviderAny)
	if err != nil {
		return err
	}
	fieldName := tokenFieldName(ctx, r, contextName)
	if fieldName == "" {
		return ErrRequired
	}
	token := strings.TrimSpace(r.FormValue(fieldName))
	if token == "" {
		return ErrRequired
	}
	userCtx, _ := user.RequestUser(ctx)
	_, err = mgr.InvokeCaptchaVerify(ctx, extName, contextName, token, clientIP(r), r, userCtx)
	if err != nil {
		return ErrVerificationFailed
	}
	clearRememberedFieldName(ctx, contextName)
	return nil
}

func parseFormPreserveBody(r *http.Request) error {
	if r == nil {
		return fmt.Errorf("captcha request is nil")
	}
	if r.Body == nil {
		return r.ParseForm()
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	err = r.ParseForm()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return err
}

// UserFacingError returns a safe message for form errors.
func UserFacingError(err error) string {
	switch {
	case errors.Is(err, ErrRequired):
		return "Please complete the captcha."
	case errors.Is(err, ErrVerificationFailed):
		return "Captcha verification failed. Please try again."
	default:
		return "Captcha verification failed. Please try again."
	}
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if host := strings.TrimSpace(r.RemoteAddr); host != "" {
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			return host[:idx]
		}
		return host
	}
	return ""
}
