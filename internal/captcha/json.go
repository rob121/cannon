package captcha

import (
	"context"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/httpreq"
	"github.com/rob121/cannon/internal/settings"
)

// VerifyJSON validates captcha_token from a JSON API request body.
func VerifyJSON(ctx context.Context, r *http.Request, formContext, token string) error {
	required, err := Required(ctx, formContext)
	if err != nil {
		return err
	}
	if !required {
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrRequired
	}
	mgr, ok := extensions.FromContext(ctx)
	if !ok || mgr == nil {
		return ErrVerificationFailed
	}
	extName, err := mgr.ResolveCaptchaExtension(ctx, ProviderAny)
	if err != nil {
		return err
	}
	req := r
	if req == nil {
		if attached, ok := httpreq.FromContext(ctx); ok {
			req = attached
		}
	}
	ip := ""
	if req != nil {
		ip = clientIPFromRequest(req)
	}
	userCtx, _ := settingsFromSkip(ctx)
	_, err = mgr.InvokeCaptchaVerify(ctx, extName, formContext, token, ip, req, userCtx)
	if err != nil {
		return ErrVerificationFailed
	}
	return nil
}

func settingsFromSkip(ctx context.Context) (map[string]any, bool) {
	skip, err := settings.CaptchaSkipAuthenticated(ctx)
	if err != nil || !skip {
		return nil, false
	}
	return nil, false
}

func clientIPFromRequest(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}
