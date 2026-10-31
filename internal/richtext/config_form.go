package richtext

import (
	"fmt"
	"html"
	"strings"
)

const (
	formKeyToolbarIntro = "#/properties/toolbar_intro"
	formKeyToolbarBody  = "#/properties/toolbar_body"
)

// InjectConfigurationToolbarBuilder replaces comma-separated toolbar fields with a visual builder.
func InjectConfigurationToolbarBuilder(formHTML string, cfg Settings) string {
	block := renderToolbarConfigBlock(cfg)
	if idx := configurationSubmitIndex(formHTML); idx >= 0 {
		return formHTML[:idx] + block + formHTML[idx:]
	}
	end := strings.LastIndex(formHTML, "</form>")
	if end < 0 {
		return formHTML + block
	}
	return formHTML[:end] + block + formHTML[end:]
}

func configurationSubmitIndex(formHTML string) int {
	marker := `class="btn btn-admin-primary" type="submit"`
	idx := strings.Index(formHTML, marker)
	if idx < 0 {
		return strings.Index(formHTML, `type="submit"`)
	}
	if start := strings.LastIndex(formHTML[:idx], "<button"); start >= 0 {
		return start
	}
	return idx
}

func renderToolbarConfigBlock(cfg Settings) string {
	var b strings.Builder
	b.WriteString(`<div class="col-12"><div class="admin-form-divider"></div></div>`)
	b.WriteString(`<div class="col-12"><h3 class="admin-form-section-title">Toolbar buttons</h3>`)
	b.WriteString(`<p class="admin-form-section-desc">Intro and body fields share the same button pool. Build each toolbar by adding controls, then drag or use arrows to reorder. Media library inserts images from your media folders; upload/insert image embeds Base64 in HTML.</p></div>`)
	b.WriteString(renderToolbarBuilder("intro", "Intro & short fields", formKeyToolbarIntro, IntroToolbarOptions(), cfg.IntroToolbar))
	b.WriteString(renderToolbarBuilder("body", "Body fields", formKeyToolbarBody, BodyToolbarOptions(), cfg.BodyToolbar))
	return b.String()
}

func renderToolbarBuilder(kind, title, formKey string, options []ToolbarOption, active []string) string {
	labelByID := map[string]string{}
	for _, o := range options {
		labelByID[o.ID] = o.Label
	}
	activeSet := map[string]bool{}
	for _, id := range active {
		activeSet[id] = true
	}

	var b strings.Builder
	b.WriteString(`<div class="col-12 admin-richtext-toolbar-builder" data-toolbar-kind="`)
	b.WriteString(html.EscapeString(kind))
	b.WriteString(`"><div class="admin-richtext-toolbar-builder-head"><h4 class="admin-form-subsection-title">`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</h4></div>`)

	b.WriteString(`<p class="admin-form-label mb-1">Active toolbar</p>`)
	b.WriteString(`<ul class="admin-richtext-toolbar-active" data-toolbar-active aria-label="`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(` toolbar">`)
	for _, id := range active {
		b.WriteString(renderActivePill(id, labelByID[id]))
	}
	b.WriteString(`</ul>`)
	b.WriteString(`<p class="admin-form-help">Drag items to reorder, or use the arrow buttons.</p>`)

	b.WriteString(`<p class="admin-form-label mb-1 mt-3">Add button</p>`)
	b.WriteString(`<div class="admin-richtext-toolbar-pool" data-toolbar-pool>`)
	for _, o := range options {
		if o.ID != "|" && activeSet[o.ID] {
			continue
		}
		b.WriteString(renderPoolPill(o.ID, o.Label))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<input type="hidden" name="`)
	b.WriteString(html.EscapeString(formKey))
	b.WriteString(`" data-toolbar-input value="`)
	b.WriteString(html.EscapeString(FormatToolbar(active)))
	b.WriteString(`"></div>`)

	return b.String()
}

func renderActivePill(id, label string) string {
	if label == "" {
		label = id
	}
	sepClass := ""
	if id == "|" {
		sepClass = " is-separator"
	}
	return fmt.Sprintf(
		`<li class="admin-richtext-toolbar-pill is-active%s" draggable="true" data-item-id="%s" data-item-label="%s">`+
			`<span class="admin-richtext-toolbar-pill-grip" aria-hidden="true">⋮⋮</span>`+
			`<span class="admin-richtext-toolbar-pill-label">%s</span>`+
			`<span class="admin-richtext-toolbar-pill-actions">`+
			`<button type="button" class="admin-richtext-toolbar-move" data-move="up" aria-label="Move left">↑</button>`+
			`<button type="button" class="admin-richtext-toolbar-move" data-move="down" aria-label="Move right">↓</button>`+
			`<button type="button" class="admin-richtext-toolbar-remove" aria-label="Remove">×</button>`+
			`</span></li>`,
		sepClass, html.EscapeString(id), html.EscapeString(label), html.EscapeString(label),
	)
}

func renderPoolPill(id, label string) string {
	sepClass := ""
	if id == "|" {
		sepClass = " is-separator"
	}
	return fmt.Sprintf(
		`<button type="button" class="admin-richtext-toolbar-pill admin-richtext-toolbar-add%s" data-item-id="%s" data-item-label="%s">+%s</button>`,
		sepClass, html.EscapeString(id), html.EscapeString(label), html.EscapeString(label),
	)
}
