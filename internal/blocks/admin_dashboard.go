package blocks

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/rob121/cannon/internal/appupdate"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/realtime"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"github.com/rob121/cannon/internal/version"
)

type adminDashboardContextKey struct{}

// AdminDashboardContext carries request-scoped deps for admin dashboard widgets.
type AdminDashboardContext struct {
	Updates  *appupdate.Manager
	Hub      *realtime.Hub
	SiteName string
	SiteURL  string
}

// WithAdminDashboard attaches dashboard render deps to ctx.
func WithAdminDashboard(ctx context.Context, deps *AdminDashboardContext) context.Context {
	if deps == nil {
		return ctx
	}
	return context.WithValue(ctx, adminDashboardContextKey{}, deps)
}

// AdminDashboardFromContext returns deps when present.
func AdminDashboardFromContext(ctx context.Context) *AdminDashboardContext {
	deps, _ := ctx.Value(adminDashboardContextKey{}).(*AdminDashboardContext)
	return deps
}

// IsAdminDashboardBlockType reports whether type is a built-in admin dashboard widget.
func IsAdminDashboardBlockType(t models.BlockType) bool {
	switch t {
	case models.BlockTypeAdminLiveAnalytics,
		models.BlockTypeAdminReviewQueue,
		models.BlockTypeAdminUpdateCheck,
		models.BlockTypeAdminLatestSignups,
		models.BlockTypeAdminQuickStart,
		models.BlockTypeAdminSystemReload:
		return true
	default:
		return false
	}
}

func renderAdminDashboardBlock(ctx context.Context, row models.Block, r *http.Request, render FragmentRenderer) (string, error) {
	if render == nil {
		return "", nil
	}
	switch row.Type {
	case models.BlockTypeAdminLiveAnalytics:
		return renderAdminLiveAnalytics(ctx, r, render)
	case models.BlockTypeAdminReviewQueue:
		return renderAdminReviewQueue(ctx, render)
	case models.BlockTypeAdminUpdateCheck:
		return renderAdminUpdateCheck(ctx, render)
	case models.BlockTypeAdminLatestSignups:
		return renderAdminLatestSignups(ctx, render)
	case models.BlockTypeAdminQuickStart:
		return renderAdminQuickStart(ctx, render)
	case models.BlockTypeAdminSystemReload:
		return renderAdminSystemReload(render)
	default:
		return "", nil
	}
}

func renderAdminLiveAnalytics(ctx context.Context, r *http.Request, render FragmentRenderer) (string, error) {
	deps := AdminDashboardFromContext(ctx)
	if deps == nil || deps.Hub == nil {
		return "", nil
	}
	enabled, err := settings.AnalyticsEnabled(ctx)
	if err != nil || !enabled {
		return "", nil
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		return "", nil
	}
	data := map[string]any{}
	scheme := "http"
	if r != nil && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") {
		scheme = "https"
	}
	host := ""
	if r != nil {
		host = r.Host
	}
	if raw, err := realtime.ConfigJSON(site.ID, scheme, host, true); err == nil {
		data["RealtimeConfigJSON"] = template.JS(raw)
	}
	if stats, err := deps.Hub.StatsForSite(site.ID); err == nil {
		data["VisitorsOnline"] = stats.Online
		data["VisitorsAuthenticated"] = stats.Authenticated
		data["AnalyticsPages"] = stats.Pages
	}
	return render(AdminLiveAnalyticsTemplate, data)
}

func renderAdminReviewQueue(ctx context.Context, render FragmentRenderer) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", nil
	}
	var pendingItems []models.Item
	_ = db.Where("status = ?", models.ItemStatusPending).
		Order("updated_at desc, item_id desc").
		Limit(5).
		Find(&pendingItems).Error
	type reviewItem struct {
		ItemID uint
		Title  string
		Slug   string
	}
	rows := make([]reviewItem, 0, len(pendingItems))
	for _, item := range pendingItems {
		rows = append(rows, reviewItem{ItemID: item.ItemID, Title: item.Title, Slug: item.Slug})
	}
	var pendingItemCount, pendingCommentCount int64
	_ = db.Model(&models.Item{}).Where("status = ?", models.ItemStatusPending).Count(&pendingItemCount).Error
	_ = db.Model(&models.Comment{}).Where("approved = ?", false).Count(&pendingCommentCount).Error
	return render(AdminReviewQueueTemplate, map[string]any{
		"ReviewItems":         rows,
		"PendingItemCount":    pendingItemCount,
		"PendingCommentCount": pendingCommentCount,
	})
}

func renderAdminUpdateCheck(ctx context.Context, render FragmentRenderer) (string, error) {
	data := map[string]any{
		"UpdateCurrentVersion": version.Version,
	}
	if deps := AdminDashboardFromContext(ctx); deps != nil && deps.Updates != nil {
		status := deps.Updates.Status()
		data["UpdateCurrentVersion"] = status.CurrentVersion
		if data["UpdateCurrentVersion"] == "" {
			data["UpdateCurrentVersion"] = version.Version
		}
		data["UpdateLatestVersion"] = status.State.LatestVersion
		data["UpdateAvailable"] = status.State.UpdateAvailable
		data["UpdateCheckedAt"] = status.State.UpdateCheckedAt
		data["UpdateError"] = status.State.UpdateError
		data["UpdateHasStaged"] = status.HasStaged
		data["UpdateStagedVersion"] = status.StagedVersion
		data["UpdateRestartRequired"] = status.State.RestartRequired
	}
	return render(AdminUpdateCheckTemplate, data)
}

func renderAdminLatestSignups(ctx context.Context, render FragmentRenderer) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", nil
	}
	var signups []models.User
	_ = db.Order("created_at desc, user_id desc").Limit(5).Find(&signups).Error
	type row struct {
		UserID    uint
		Username  string
		Name      string
		CreatedAt time.Time
	}
	out := make([]row, 0, len(signups))
	for _, u := range signups {
		out = append(out, row{
			UserID:    u.UserID,
			Username:  u.Username,
			Name:      user.DisplayName(&u),
			CreatedAt: u.CreatedAt,
		})
	}
	return render(AdminLatestSignupsTemplate, map[string]any{"LatestSignups": out})
}

func renderAdminQuickStart(ctx context.Context, render FragmentRenderer) (string, error) {
	data := map[string]any{}
	if deps := AdminDashboardFromContext(ctx); deps != nil {
		data["SiteName"] = deps.SiteName
		data["SiteURL"] = deps.SiteURL
	}
	return render(AdminQuickStartTemplate, data)
}

func renderAdminSystemReload(render FragmentRenderer) (string, error) {
	return render(AdminSystemReloadTemplate, map[string]any{})
}

// AdminDashboardPlainMeta is the default metadata for seeded dashboard widgets.
func AdminDashboardPlainMeta() string {
	raw, _ := EncodeMetadata(Metadata{
		TemplateWrapper: PlainWrapperTemplate,
		ShowName:        false,
	})
	return raw
}
