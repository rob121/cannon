package templateengine

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/rob121/cannon/internal/captcha"
)

// FuncMap returns html/template functions for Cannon templates.
// Sprig helpers are included; Cannon-specific helpers override name collisions.
func FuncMap(blocks BlockRenderer, blockLen BlockLenRenderer) template.FuncMap {
	funcs := sprig.FuncMap()
	for name, fn := range cannonFuncMap(blocks, blockLen) {
		funcs[name] = fn
	}
	return funcs
}

func cannonFuncMap(blocks BlockRenderer, blockLen BlockLenRenderer) template.FuncMap {
	return template.FuncMap{
		"space": func(name string) (template.HTML, error) {
			if blocks == nil {
				return "", nil
			}
			return blocks(name)
		},
		"lenspace": func(name string) (int, error) {
			if blockLen == nil {
				return 0, nil
			}
			return blockLen(name)
		},
		// Keep Cannon math helpers so admin pagination works with int64 totals.
		"add": func(a, b any) int { return asInt(a) + asInt(b) },
		"sub": func(a, b any) int { return asInt(a) - asInt(b) },
		"mul": func(a, b any) int { return asInt(a) * asInt(b) },
		"div": func(a, b any) int {
			bv := asInt(b)
			if bv == 0 {
				return 0
			}
			av := asInt(a)
			return (av + bv - 1) / bv
		},
		"intRange": func(n any) []int {
			count := asInt(n)
			if count <= 0 {
				return nil
			}
			out := make([]int, count)
			for i := range out {
				out[i] = i
			}
			return out
		},
		"min": func(a, b any) int {
			av, bv := asInt(a), asInt(b)
			if av < bv {
				return av
			}
			return bv
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: odd number of arguments")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key must be string")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
		"queryEscape": url.QueryEscape,
		"initials": func(parts ...string) string {
			var out string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					out += strings.ToUpper(p[:1])
				}
			}
			if out == "" {
				return "?"
			}
			if len(out) > 2 {
				return out[:2]
			}
			return out
		},
		"inUint": func(list []uint, id uint) bool {
			for _, v := range list {
				if v == id {
					return true
				}
			}
			return false
		},
		"inString": func(list []string, v string) bool {
			for _, item := range list {
				if item == v {
					return true
				}
			}
			return false
		},
		"showRouteTitle": func() bool { return true },
		"siteName":            func() string { return "" },
		"siteMetaDescription": func() string { return "" },
		"siteMetaKeywords":    func() string { return "" },
		"siteOGTitle":         func() string { return "" },
		"siteOGImage":         func() string { return "" },
		"siteTwitterCard":     func() string { return "summary_large_image" },
		"siteTwitterSite":     func() string { return "" },
		"siteTwitterCreator":  func() string { return "" },
		"siteHeadExtra":       func() template.HTML { return "" },
		// captcha emits a <captcha> placeholder expanded by Cannon before onAfterRender.
		// Usage: {{captcha "login"}} or {{captcha "comment" "any"}}
		"captcha": func(args ...string) template.HTML {
			context := "form"
			provider := captcha.ProviderAny
			if len(args) > 0 {
				context = strings.TrimSpace(args[0])
			}
			if len(args) > 1 {
				provider = strings.TrimSpace(args[1])
			}
			return template.HTML(captcha.PlaceholderMarkup(context, provider))
		},
	}
}
