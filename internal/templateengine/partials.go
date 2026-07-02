package templateengine

import (
	"fmt"
	"html/template"
	"strings"
)

const loginFormPartial = "default/partials/auth/login-form.html"

var contentListTemplates = map[string]struct{}{
	"default/controllers/content/index.html":    {},
	"default/controllers/content/category.html": {},
	"default/controllers/content/tag.html":      {},
	"default/controllers/content/author.html":   {},
	"default/controllers/content/featured.html": {},
	"default/controllers/content/search.html":   {},
}

func builtinPartialsFor(name string) []string {
	var partials []string
	switch name {
	case "default/layout.html":
		partials = append(partials, "default/partials/offline-notice.html")
	case "default/controllers/auth/login.html", "default/partials/blocks/login.html", "default/offline.html":
		partials = append(partials, loginFormPartial)
	case errorFallbackPage:
		partials = append(partials, errorPagePartial, "default/partials/page-header.html")
	default:
		if strings.HasPrefix(name, "default/error/") {
			partials = append(partials, errorPagePartial, "default/partials/page-header.html")
		}
	}

	if usesPageHeaderPartial(name) {
		partials = append(partials, "default/partials/page-header.html")
	}
	if _, ok := contentListTemplates[name]; ok {
		partials = append(partials, "default/partials/content/item-card.html")
	}
	if name == "default/controllers/content/category.html" {
		partials = append(partials, "default/partials/content/pagination.html")
	}
	if name == "default/controllers/content/item.html" {
		partials = append(partials, "default/partials/content/item-media.html")
	}
	if name == "default/controllers/content/edit.html" {
		partials = append(partials, "default/partials/content/custom-field-input.html")
	}
	return partials
}

func usesPageHeaderPartial(name string) bool {
	switch name {
	case "default/page.html", "default/maintenance.html":
		return true
	default:
		return strings.HasPrefix(name, "default/controllers/")
	}
}

func (e *Engine) attachPartials(base *template.Template, name string) error {
	for _, partial := range builtinPartialsFor(name) {
		content, _, err := e.readTemplate(partial)
		if err != nil {
			continue
		}
		if _, err := base.Parse(content); err != nil {
			return fmt.Errorf("parse %s partial for %s: %w", partial, name, err)
		}
	}
	return nil
}
