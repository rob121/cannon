package hooks

// System hook events.
const (
	OnBeforeRoute = "onBeforeRoute"
	OnAfterRoute  = "onAfterRoute"
	OnBeforeRender = "onBeforeRender"
	OnAfterRender  = "onAfterRender"
)

// User hook events.
const (
	OnUserBeforeLogin  = "onUserBeforeLogin"
	OnUserAfterLogin   = "onUserAfterLogin"
	OnUserLogout       = "onUserLogout"
	OnUserSignup       = "onUserSignup"
	OnUserVerified     = "onUserVerified"
	OnUserLocked       = "onUserLocked"
)

// Block hook events.
const (
	OnRenderBlock      = "onRenderBlock"
	OnAfterRenderBlock = "onAfterRenderBlock"
)

// Content hook events.
const (
	OnContentPrepare       = "onContentPrepare"
	OnContentBeforeDisplay = "onContentBeforeDisplay"
	OnContentAfterDisplay  = "onContentAfterDisplay"
	OnItemBeforeSave       = "onItemBeforeSave"
	OnItemAfterSave        = "onItemAfterSave"
	OnItemBeforeRender     = "onItemBeforeRender"
	OnCommentBeforeSave    = "onCommentBeforeSave"
	OnCommentAfterSave     = "onCommentAfterSave"
)

// AllEvents lists every hook name Cannon may dispatch.
var AllEvents = []string{
	OnBeforeRoute,
	OnAfterRoute,
	OnBeforeRender,
	OnAfterRender,
	OnUserBeforeLogin,
	OnUserAfterLogin,
	OnUserLogout,
	OnUserSignup,
	OnUserVerified,
	OnUserLocked,
	OnRenderBlock,
	OnAfterRenderBlock,
	OnContentPrepare,
	OnContentBeforeDisplay,
	OnContentAfterDisplay,
	OnItemBeforeSave,
	OnItemAfterSave,
	OnItemBeforeRender,
	OnCommentBeforeSave,
	OnCommentAfterSave,
}
