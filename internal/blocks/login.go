package blocks

import (
	"context"
	"html"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/auth"
)

// RenderLoginBlock renders a sign-in form block for the current request.
func RenderLoginBlock(ctx context.Context, r *http.Request, row BlockRow, meta Metadata, render FragmentRenderer) (string, error) {
	data, err := auth.BuildLoginFormData(ctx, r, auth.LoginFormOptions{
		BlockID: row.BlockID,
		Title:   meta.LoginTitle,
	})
	if err != nil {
		return "", err
	}
	if render != nil {
		html, err := render(LoginBlockTemplate, map[string]any{"Login": data})
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(html), nil
	}
	return renderLoginBlockFallback(data), nil
}

// BlockRow carries the block fields needed for login rendering.
type BlockRow struct {
	BlockID uint
	Name    string
}

func renderLoginBlockFallback(data auth.LoginFormData) string {
	var b strings.Builder
	b.WriteString(`<div class="site-login-block">`)
	if data.Title != "" {
		b.WriteString(`<h3 class="h5 mb-3">`)
		b.WriteString(htmlEscape(data.Title))
		b.WriteString(`</h3>`)
	}
	if data.Disabled {
		b.WriteString(`<div class="alert alert-info mb-0" role="alert">`)
		b.WriteString(htmlEscape(data.DisabledMessage))
		b.WriteString(`</div></div>`)
		return b.String()
	}
	if data.Authenticated {
		b.WriteString(`<p class="mb-3">Signed in as <strong>`)
		b.WriteString(htmlEscape(data.Username))
		b.WriteString(`</strong></p><a class="btn btn-outline-secondary btn-sm" href="`)
		b.WriteString(htmlEscape(data.LogoutURL))
		b.WriteString(`">Sign out</a></div>`)
		return b.String()
	}
	if data.Error != "" {
		b.WriteString(`<div class="alert alert-danger" role="alert">`)
		b.WriteString(htmlEscape(data.Error))
		b.WriteString(`</div>`)
	}
	if data.LocalEnabled {
		b.WriteString(`<form method="post" action="`)
		b.WriteString(htmlEscape(data.LoginAction))
		b.WriteString(`">`)
		b.WriteString(string(data.CSRF))
		if data.Return != "" {
			b.WriteString(`<input type="hidden" name="return" value="`)
			b.WriteString(htmlEscape(data.Return))
			b.WriteString(`">`)
		}
		b.WriteString(`<div class="mb-3"><label class="form-label">Username</label><input class="form-control" name="username" required autocomplete="username"></div>`)
		b.WriteString(`<div class="mb-3"><label class="form-label">Password</label><input class="form-control" name="password" type="password" required autocomplete="current-password"></div>`)
		b.WriteString(`<div class="d-flex flex-wrap gap-2 align-items-center"><button type="submit" class="btn btn-primary">Sign In</button></div></form>`)
	}
	if len(data.Providers) > 0 {
		if data.LocalEnabled {
			b.WriteString(`<div class="text-center text-muted small my-3">or continue with</div>`)
		}
		b.WriteString(`<div class="d-grid gap-2">`)
		for _, provider := range data.Providers {
			b.WriteString(`<a class="btn btn-outline-primary" href="`)
			b.WriteString(htmlEscape(provider.URL))
			b.WriteString(`">Sign in with `)
			b.WriteString(htmlEscape(provider.Label))
			b.WriteString(`</a>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func htmlEscape(s string) string {
	return html.EscapeString(s)
}
