package hooks

// System hook events.
const (
	OnBeforeRoute            = "onBeforeRoute"
	OnAfterRoute             = "onAfterRoute"
	OnBeforeRender           = "onBeforeRender"
	OnAfterRender            = "onAfterRender"
	OnPrepareDocumentHead    = "onPrepareDocumentHead"
	OnPrepareDocumentBody    = "onPrepareDocumentBody"
	OnAdminBeforeRender      = "onAdminBeforeRender"
	OnAdminPrepareDocumentHead = "onAdminPrepareDocumentHead"
	OnAdminPrepareDocumentBody = "onAdminPrepareDocumentBody"
	OnSettingsSave           = "onSettingsSave"
	OnSitemapGenerate        = "onSitemapGenerate"
	OnRobotsGenerate         = "onRobotsGenerate"
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
}
