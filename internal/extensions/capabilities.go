package extensions

import "strings"

// CapabilityItem describes one reported extension capability.
type CapabilityItem struct {
	Key         string
	Label       string
	Handler     string
	Description string
	Enabled     bool
}

// CapabilitiesSummary is formatted capability data for admin display.
type CapabilitiesSummary struct {
	Available            bool
	Items                []CapabilityItem
	DefaultAdminMenuName string
}

var capabilityCatalog = []struct {
	key   string
	label string
	desc  string
	get   func(Capabilities) string
}{
	{
		key:   "request",
		label: "Request",
		desc:  "Intercepts web requests in middleware to alter, block, or short-circuit routing.",
		get:   func(c Capabilities) string { return c.Request },
	},
	{
		key:   "page",
		label: "Page",
		desc:  "Renders page content when a route is associated with this extension.",
		get:   func(c Capabilities) string { return c.Page },
	},
	{
		key:   "endpoint",
		label: "Endpoint",
		desc:  "Handles admin-configured data routes with full HTTP passthrough.",
		get:   func(c Capabilities) string { return c.Endpoint },
	},
	{
		key:   "data",
		label: "Data",
		desc:  "Handles path-based data routes at /ext/{route_hash}/… automatically.",
		get:   func(c Capabilities) string { return c.Data },
	},
	{
		key:   "block",
		label: "Block",
		desc:  "Renders template spaces assigned to this extension.",
		get:   func(c Capabilities) string { return c.Block },
	},
	{
		key:   "admin",
		label: "Admin",
		desc:  "Provides an admin UI surfaced under Extensions in the Cannon admin.",
		get:   func(c Capabilities) string { return c.Admin },
	},
	{
		key:   "help",
		label: "Help",
		desc:  "Publishes help articles included in the admin Help area.",
		get:   func(c Capabilities) string { return c.Help },
	},
	{
		key:   "hooks",
		label: "Hooks",
		desc:  "Subscribes to Cannon event hooks (onBeforeRoute, onUserBeforeLogin, etc.).",
		get:   func(c Capabilities) string { return c.Hooks },
	},
	{
		key:   "templates",
		label: "Templates",
		desc:  "Lists embedded templates that can be copied into the site template override folder.",
		get:   func(c Capabilities) string { return c.Templates },
	},
	{
		key:   "captcha",
		label: "Captcha",
		desc:  "Renders and verifies captcha widgets for protected forms. Only one extension is active per site.",
		get:   func(c Capabilities) string { return c.Captcha },
	},
}

// CapabilitiesSummary returns capability details for a running extension.
func (m *Manager) CapabilitiesSummary(name string) CapabilitiesSummary {
	rt, ok := m.runtime(name)
	if !ok || !m.IsRunning(name) {
		return CapabilitiesSummary{Available: false}
	}

	items := capabilityItems(rt.Capabilities)
	return CapabilitiesSummary{
		Available:            true,
		Items:                items,
		DefaultAdminMenuName: rt.DefaultAdminMenuName,
	}
}

func capabilityItems(caps Capabilities) []CapabilityItem {
	items := make([]CapabilityItem, 0, len(capabilityCatalog))
	for _, spec := range capabilityCatalog {
		handler := strings.TrimPrefix(strings.TrimSpace(spec.get(caps)), "/")
		items = append(items, CapabilityItem{
			Key:         spec.key,
			Label:       spec.label,
			Handler:     handler,
			Description: spec.desc,
			Enabled:     handler != "",
		})
	}
	return items
}
