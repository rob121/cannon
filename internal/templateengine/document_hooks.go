package templateengine

import (
	"html/template"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
)

func (e *Engine) applyDocumentHooks(layoutData map[string]any, layout, page string) error {
	if e.hookCtx == nil {
		return nil
	}
	args := map[string]any{
		"layout":  layout,
		"page":    page,
		"context": renderContextForLayout(layout),
	}
	for _, key := range []string{"Title", "controller", "action", "route_name", "route_path"} {
		if v, ok := layoutData[key]; ok {
			args[key] = v
		}
	}

	isAdmin := strings.HasPrefix(layout, "admin/")
	if isAdmin {
		if out, err := hooks.Fire(e.hookCtx, hooks.OnAdminBeforeRender, args); err != nil {
			return err
		} else {
			applyRenderOverrides(out, &layout, &page)
		}
		headOut, err := hooks.Fire(e.hookCtx, hooks.OnAdminPrepareDocumentHead, args)
		if err != nil {
			return err
		}
		bodyOut, err := hooks.Fire(e.hookCtx, hooks.OnAdminPrepareDocumentBody, args)
		if err != nil {
			return err
		}
		layoutData["HeadExtra"] = template.HTML(hooks.HTMLFragment(headOut, "head_html"))
		layoutData["BodyEndExtra"] = template.HTML(hooks.HTMLFragment(bodyOut, "body_html"))
		return nil
	}

	headOut, err := hooks.Fire(e.hookCtx, hooks.OnPrepareDocumentHead, args)
	if err != nil {
		return err
	}
	bodyOut, err := hooks.Fire(e.hookCtx, hooks.OnPrepareDocumentBody, args)
	if err != nil {
		return err
	}
	layoutData["HeadExtra"] = template.HTML(hooks.HTMLFragment(headOut, "head_html"))
	layoutData["BodyEndExtra"] = template.HTML(hooks.HTMLFragment(bodyOut, "body_html"))
	return nil
}

func renderContextForLayout(layout string) string {
	if strings.HasPrefix(layout, "admin/") {
		return "admin"
	}
	return "frontend"
}

func applyRenderOverrides(out map[string]any, layout, page *string) {
	if v, ok := out["layout"].(string); ok && v != "" {
		*layout = v
	}
	if v, ok := out["page"].(string); ok && v != "" {
		*page = v
	}
}
