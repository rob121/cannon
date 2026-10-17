package extension

import (
	"strconv"
	"strings"
	"time"
)

// Common settings keys passed on WireRequest.Settings.
const (
	SettingDateFormat      = "date_format"
	SettingSiteName        = "site_name"
	SettingDevelopmentMode = "development_mode"
)

// DefaultDateFormat is the Go layout used when date_format is unset.
const DefaultDateFormat = "01/02/2006"

// Setting returns an allowlisted site setting from the wire request.
func Setting(req WireRequest, key string) string {
	if req.Settings == nil {
		return ""
	}
	return strings.TrimSpace(req.Settings[strings.TrimSpace(key)])
}

// DevelopmentMode reports whether the site has development_mode enabled.
func DevelopmentMode(req WireRequest) bool {
	raw := Setting(req, SettingDevelopmentMode)
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return strings.EqualFold(raw, "on") || strings.EqualFold(raw, "yes")
	}
	return v
}

// DateFormat returns the site's Go date layout from wire settings.
func DateFormat(req WireRequest) string {
	layout := Setting(req, SettingDateFormat)
	if layout == "" {
		return DefaultDateFormat
	}
	return layout
}

// FormatDate formats t using the site date_format setting (date only).
// A zero time returns an empty string.
func FormatDate(req WireRequest, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateFormat(req))
}

// FormatDatePtr formats a nullable time using the site date_format setting.
func FormatDatePtr(req WireRequest, t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return FormatDate(req, *t)
}
