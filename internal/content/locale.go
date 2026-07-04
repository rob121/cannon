package content

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type localeKey struct{}

const defaultContentLocale = "en-US"

// LocaleConfig returns multilingual settings for the current site.
func LocaleConfig(ctx context.Context) (enabled bool, defaultLocale string, locales []string, err error) {
	defaultLocale = defaultContentLocale
	locales = []string{defaultContentLocale}
	enabled, err = settings.GlobalBoolDefault(ctx, settings.SectionGeneral, "content_multilingual", false)
	if err != nil {
		return false, defaultLocale, locales, err
	}
	if v, err := settings.GlobalString(ctx, settings.SectionGeneral, "content_default_locale"); err == nil && strings.TrimSpace(v) != "" {
		defaultLocale = strings.TrimSpace(v)
	}
	if raw, err := settings.GlobalStringSlice(ctx, settings.SectionGeneral, "content_locales"); err == nil && len(raw) > 0 {
		locales = normalizeLocales(raw)
	} else {
		site, siteErr := sites.FromContext(ctx)
		if siteErr == nil {
			if installed := lang.InstalledLocales(site.LanguageDir); len(installed) > 0 {
				locales = installed
			}
		}
	}
	if !containsLocale(locales, defaultLocale) {
		locales = append([]string{defaultLocale}, locales...)
	}
	if !enabled {
		locales = []string{defaultLocale}
	}
	return enabled, defaultLocale, locales, nil
}

// WithLocale stores the active content locale on a context.
func WithLocale(ctx context.Context, locale string) context.Context {
	if strings.TrimSpace(locale) == "" {
		return ctx
	}
	return context.WithValue(ctx, localeKey{}, strings.TrimSpace(locale))
}

// LocaleFromContext returns the active content locale, or the site default.
func LocaleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(localeKey{}).(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	_, defaultLocale, _, err := LocaleConfig(ctx)
	if err != nil || defaultLocale == "" {
		return defaultContentLocale
	}
	return defaultLocale
}

// ResolveContentLocale picks a content locale from URL prefix, UI locale, cookie, or Accept-Language.
func ResolveContentLocale(ctx context.Context, path, uiLocale, cookie, acceptLanguage string) (locale string, strippedPath string) {
	enabled, defaultLocale, locales, err := LocaleConfig(ctx)
	strippedPath = path
	if err != nil || !enabled {
		return defaultLocale, strippedPath
	}
	if prefix, loc, ok := StripLocalePrefix(path, locales, defaultLocale); ok {
		return loc, prefix
	}
	for _, candidate := range []string{uiLocale, cookie, lang.ResolveLocale("", acceptLanguage)} {
		if containsLocale(locales, candidate) {
			return candidate, strippedPath
		}
	}
	return defaultLocale, strippedPath
}

// StripLocalePrefix removes a leading /{locale} segment when it matches an enabled locale.
func StripLocalePrefix(path string, locales []string, defaultLocale string) (stripped string, locale string, ok bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/", defaultLocale, false
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	rest := strings.TrimPrefix(path, "/")
	if rest == "" {
		return path, defaultLocale, false
	}
	segment, after, found := strings.Cut(rest, "/")
	if !found && !strings.Contains(rest, "/") {
		after = ""
		segment = rest
	}
	if segment == "" || !containsLocale(locales, segment) {
		return path, defaultLocale, false
	}
	if after == "" {
		stripped = "/"
	} else {
		stripped = "/" + after
	}
	return stripped, segment, true
}

// LocalePrefix returns a URL prefix for non-default locales (e.g. "/fr-FR" or "").
func LocalePrefix(ctx context.Context, locale string) string {
	enabled, defaultLocale, _, err := LocaleConfig(ctx)
	if err != nil || !enabled || strings.TrimSpace(locale) == "" || locale == defaultLocale {
		return ""
	}
	return "/" + locale
}

// LocalizedPath prefixes a site path with the locale segment when needed.
func LocalizedPath(ctx context.Context, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	prefix := LocalePrefix(ctx, LocaleFromContext(ctx))
	if prefix == "" {
		return path
	}
	if path == "/" {
		return prefix
	}
	return prefix + path
}

// NormalizeItemLocale fills an empty item locale from context/settings.
func NormalizeItemLocale(ctx context.Context, row *models.Item) {
	if row == nil {
		return
	}
	if strings.TrimSpace(row.Locale) != "" {
		row.Locale = strings.TrimSpace(row.Locale)
		return
	}
	row.Locale = LocaleFromContext(ctx)
}

// NormalizeCategoryLocale fills an empty category locale from context/settings.
func NormalizeCategoryLocale(ctx context.Context, row *models.Category) {
	if row == nil {
		return
	}
	if strings.TrimSpace(row.Locale) != "" {
		row.Locale = strings.TrimSpace(row.Locale)
		return
	}
	row.Locale = LocaleFromContext(ctx)
}

// ItemTranslations returns other items in the same translation group.
func ItemTranslations(ctx context.Context, item *models.Item) ([]models.Item, error) {
	if item == nil || item.TranslationGroupID == nil || *item.TranslationGroupID == 0 {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Item
	err = db.Where("translation_group_id = ? AND item_id <> ?", *item.TranslationGroupID, item.ItemID).
		Order("locale ASC, title ASC").Find(&rows).Error
	return rows, err
}

// LinkTranslation assigns a shared translation group between two items.
func LinkTranslation(ctx context.Context, itemID, otherID uint) error {
	if itemID == 0 || otherID == 0 || itemID == otherID {
		return nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var item, other models.Item
	if err := db.First(&item, itemID).Error; err != nil {
		return err
	}
	if err := db.First(&other, otherID).Error; err != nil {
		return err
	}
	groupID := itemID
	if item.TranslationGroupID != nil && *item.TranslationGroupID > 0 {
		groupID = *item.TranslationGroupID
	} else if other.TranslationGroupID != nil && *other.TranslationGroupID > 0 {
		groupID = *other.TranslationGroupID
	}
	return db.Model(&models.Item{}).
		Where("item_id IN ?", []uint{itemID, otherID}).
		Update("translation_group_id", groupID).Error
}

// CategoriesByLocale returns active categories for a specific locale.
func CategoriesByLocale(ctx context.Context, locale string) ([]models.Category, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(locale) == "" {
		locale = LocaleFromContext(ctx)
	}
	var rows []models.Category
	err = db.Where("status = ? AND locale = ?", models.StatusActive, locale).
		Order("sort ASC, name ASC").Find(&rows).Error
	return rows, err
}

func applyLocaleScope(q *gorm.DB, ctx context.Context, table string) *gorm.DB {
	locale := LocaleFromContext(ctx)
	if locale == "" || strings.TrimSpace(table) == "" {
		return q
	}
	return q.Where(table+".locale = ?", locale)
}

func normalizeLocales(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		loc := strings.TrimSpace(raw)
		if loc == "" {
			continue
		}
		if _, ok := seen[loc]; ok {
			continue
		}
		seen[loc] = struct{}{}
		out = append(out, loc)
	}
	return out
}

func containsLocale(locales []string, locale string) bool {
	locale = strings.TrimSpace(locale)
	for _, item := range locales {
		if item == locale {
			return true
		}
	}
	return false
}
