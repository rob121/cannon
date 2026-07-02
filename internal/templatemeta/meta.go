package templatemeta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = "template.json"

// VersionsDirName is the mirrored backup directory excluded from theme listings.
const VersionsDirName = "versions"

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

const (
	TypeFrontend = "frontend"
	TypeBackend  = "backend"
	TypeFull     = "full"
)

// GroupMeta describes metadata for a template group folder.
type GroupMeta struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// PackMeta describes the site template pack stored in template.json.
type PackMeta struct {
	Name        string               `json:"name"`
	Type        string               `json:"type"`
	Author      string               `json:"author"`
	Description string               `json:"description"`
	Version     string               `json:"version"`
	Status      string               `json:"status"`
	Groups      map[string]GroupMeta `json:"groups,omitempty"`
}

// DefaultPackMeta returns baseline metadata for a new template pack.
func DefaultPackMeta() PackMeta {
	return PackMeta{
		Status: StatusActive,
		Type:   TypeFull,
	}
}

// Load reads template.json from the template root.
func Load(root string) (PackMeta, error) {
	meta := DefaultPackMeta()
	root = strings.TrimSpace(root)
	if root == "" {
		return meta, nil
	}
	raw, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return meta, nil
		}
		return meta, err
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, fmt.Errorf("parse %s: %w", FileName, err)
	}
	meta.normalize()
	return meta, nil
}

// Save writes template.json to the template root.
func Save(root string, meta PackMeta) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("template directory is not configured")
	}
	meta.normalize()
	if err := meta.validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, FileName), raw, 0644)
}

func (m *PackMeta) normalize() {
	m.Status = normalizeStatus(m.Status, StatusActive)
	m.Type = normalizeType(m.Type, TypeFull)
	if m.Groups == nil {
		m.Groups = map[string]GroupMeta{}
	}
	for name, group := range m.Groups {
		group.Status = normalizeStatus(group.Status, StatusActive)
		group.Type = normalizeType(group.Type, defaultGroupType(name))
		m.Groups[name] = group
	}
}

func (m PackMeta) validate() error {
	if m.Status != StatusActive && m.Status != StatusInactive {
		return fmt.Errorf("invalid pack status %q", m.Status)
	}
	switch m.Type {
	case TypeFrontend, TypeBackend, TypeFull:
	default:
		return fmt.Errorf("invalid pack type %q", m.Type)
	}
	for name, group := range m.Groups {
		if err := validateGroupName(name); err != nil {
			return err
		}
		if group.Status != StatusActive && group.Status != StatusInactive {
			return fmt.Errorf("invalid status for group %q", name)
		}
		switch group.Type {
		case TypeFrontend, TypeBackend, TypeFull, "":
		default:
			return fmt.Errorf("invalid type for group %q", name)
		}
	}
	return nil
}

// Active reports whether the pack is marked active.
func (m PackMeta) Active() bool {
	return m.Status == StatusActive
}

// GroupStatus returns the effective status for a template group.
func (m PackMeta) GroupStatus(group string) string {
	if !m.Active() {
		return StatusInactive
	}
	if gm, ok := m.Groups[group]; ok && gm.Status != "" {
		return gm.Status
	}
	return StatusActive
}

// GroupType returns the display type for a template group.
func (m PackMeta) GroupType(group string) string {
	if gm, ok := m.Groups[group]; ok && gm.Type != "" {
		return gm.Type
	}
	switch m.Type {
	case TypeFrontend, TypeBackend:
		return m.Type
	default:
		return defaultGroupType(group)
	}
}

// GroupLabel returns a human-friendly label for a template group.
func (m PackMeta) GroupLabel(group string) string {
	if gm, ok := m.Groups[group]; ok && strings.TrimSpace(gm.Label) != "" {
		return gm.Label
	}
	return group
}

// OverridesEnabled reports whether a theme pack should be used at runtime.
func OverridesEnabled(themeRoot string) bool {
	meta, err := Load(themeRoot)
	if err != nil {
		return false
	}
	return meta.Active()
}

func defaultGroupType(group string) string {
	switch group {
	case "admin":
		return TypeBackend
	case "default":
		return TypeFrontend
	default:
		return TypeFull
	}
}

func normalizeStatus(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case StatusActive, StatusInactive:
		return value
	case "":
		return fallback
	default:
		return fallback
	}
}

func normalizeType(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case TypeFrontend, TypeBackend, TypeFull:
		return value
	case "":
		return fallback
	default:
		return fallback
	}
}

func groupFromPath(path string) string {
	path = strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

func validateGroupName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid template group")
	}
	return nil
}
