package version

// ReleaseVersion is the default version for GitHub releases when no tag or
// workflow_dispatch input is provided. Bump this before running a release.
const ReleaseVersion = "0.1.2"

// Version and Commit are set at build time via -ldflags.
// Local/dev builds leave Version as "dev" unless ldflags override it.
var (
	Version = "dev"
	Commit  = ""
)
