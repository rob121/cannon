package hooks

// System hook events.
const (
	OnBeforeRoute              = "onBeforeRoute"
	OnAfterRoute               = "onAfterRoute"
	OnBeforeRender             = "onBeforeRender"
	OnAfterRender              = "onAfterRender"
	OnPrepareDocumentHead      = "onPrepareDocumentHead"
	OnPrepareDocumentBody      = "onPrepareDocumentBody"
	OnAdminBeforeRender        = "onAdminBeforeRender"
	OnAdminPrepareDocumentHead = "onAdminPrepareDocumentHead"
	OnAdminPrepareDocumentBody = "onAdminPrepareDocumentBody"
	OnSettingsSave             = "onSettingsSave"
	OnSitemapGenerate          = "onSitemapGenerate"
	OnRobotsGenerate           = "onRobotsGenerate"
)

// User hook events.
const (
	OnUserBeforeLogin = "onUserBeforeLogin"
	OnUserAfterLogin  = "onUserAfterLogin"
	OnUserLogout      = "onUserLogout"
	OnUserSignup      = "onUserSignup"
	OnUserVerified    = "onUserVerified"
	OnUserLocked      = "onUserLocked"
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
	OnItemBeforeDelete     = "onItemBeforeDelete"
	OnItemAfterDelete      = "onItemAfterDelete"
	OnItemTrash            = "onItemTrash"
	OnItemRestore          = "onItemRestore"
	OnItemBeforeRender     = "onItemBeforeRender"
	OnCategoryBeforeSave   = "onCategoryBeforeSave"
	OnCategoryAfterSave    = "onCategoryAfterSave"
	OnCategoryBeforeDelete = "onCategoryBeforeDelete"
	OnMediaUpload          = "onMediaUpload"
	OnMediaDelete          = "onMediaDelete"
	OnRevisionRestore      = "onRevisionRestore"
	OnBeforeSearch         = "onBeforeSearch"
	OnAfterSearch          = "onAfterSearch"
	OnCommentBeforeSave    = "onCommentBeforeSave"
	OnCommentAfterSave     = "onCommentAfterSave"
)

// Mail hook events.
const (
	OnBeforeMailSend = "onBeforeMailSend"
)

// Extension / contribution hook events.
const (
	// OnSubmissionAfterCreate fires when the submission extension accepts a new tip.
	OnSubmissionAfterCreate = "onSubmissionAfterCreate"
)

// EmittableEvents lists hook names extensions may request via WireResponse.emit_events.
// These are after-the-fact notify-safe events only (not login/abort hooks).
var EmittableEvents = []string{
	OnSubmissionAfterCreate,
}

// IsEmittable reports whether an extension may ask the host to fire this event.
func IsEmittable(event string) bool {
	for _, name := range EmittableEvents {
		if name == event {
			return true
		}
	}
	return false
}

// AllEvents lists every hook name Cannon may dispatch.
var AllEvents = []string{
	OnBeforeRoute,
	OnAfterRoute,
	OnBeforeRender,
	OnAfterRender,
	OnPrepareDocumentHead,
	OnPrepareDocumentBody,
	OnAdminBeforeRender,
	OnAdminPrepareDocumentHead,
	OnAdminPrepareDocumentBody,
	OnSettingsSave,
	OnSitemapGenerate,
	OnRobotsGenerate,
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
	OnItemBeforeDelete,
	OnItemAfterDelete,
	OnItemTrash,
	OnItemRestore,
	OnItemBeforeRender,
	OnCategoryBeforeSave,
	OnCategoryAfterSave,
	OnCategoryBeforeDelete,
	OnMediaUpload,
	OnMediaDelete,
	OnRevisionRestore,
	OnBeforeSearch,
	OnAfterSearch,
	OnCommentBeforeSave,
	OnCommentAfterSave,
	OnBeforeMailSend,
	OnSubmissionAfterCreate,
}
