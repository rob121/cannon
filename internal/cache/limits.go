package cache

// Soft caps for process-local caches. Tuned for small multi-site VPS hosts:
// keep hot data in RAM without unbounded growth from users/spaces/sections.
//
// These are package vars so tests can lower them; production uses the defaults.
var (
	// MaxViewerGroupsPerSite caps cached authenticated viewer→group lookups.
	MaxViewerGroupsPerSite = 512

	// MaxBlockSpacesPerSite caps cached template spaces per site.
	MaxBlockSpacesPerSite = 128

	// MaxSettingsEntriesPerSite caps cached scope+section settings blobs.
	MaxSettingsEntriesPerSite = 64

	// MaxRoutesSites caps how many sites keep a full active-routes snapshot.
	MaxRoutesSites = 32
)
