package extension

import (
	"strings"
	"unicode"
)

// TablePrefixFromName derives a database table prefix from an extension binary name.
// For example "cannon-extension-contact" becomes "contact".
func TablePrefixFromName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "cannon-extension-")
	return sanitizeTableIdent(name)
}

// TablePrefix returns the table prefix for this extension's Info.Name.
func (i Info) TablePrefix() string {
	return TablePrefixFromName(i.Name)
}

// TableName joins a prefix and suffix with a single underscore.
// Example: TableName("contact", "forms") => "contact_forms".
func TableName(prefix, suffix string) string {
	prefix = sanitizeTableIdent(prefix)
	suffix = sanitizeTableIdent(suffix)
	if prefix == "" {
		return suffix
	}
	if suffix == "" {
		return prefix
	}
	return prefix + "_" + suffix
}

func sanitizeTableIdent(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
