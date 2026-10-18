package extensions

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

const binaryMetaProbeTimeout = 5 * time.Second

// BinaryMeta is metadata read from an extension binary via --data=all.
type BinaryMeta struct {
	Name          string
	Title         string
	Description   string
	Version       string
	UpdateURLBase string
	AdminMenuName string
}

// ProbeBinaryMeta runs `<binary> --data=all` and parses key=value metadata.
// The extension does not need to be installed or running.
func (m *Manager) ProbeBinaryMeta(name string) (BinaryMeta, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return BinaryMeta{}, fmt.Errorf("extension name is required")
	}
	path := m.binaryPath(name)
	if _, err := os.Stat(path); err != nil {
		return BinaryMeta{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), binaryMetaProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--data=all")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return BinaryMeta{}, fmt.Errorf("probe %s: %s", name, msg)
	}
	meta, err := parseBinaryMetaOutput(stdout.String())
	if err != nil {
		return BinaryMeta{}, err
	}
	if strings.TrimSpace(meta.Name) == "" {
		meta.Name = name
	}
	return meta, nil
}

func parseBinaryMetaOutput(raw string) (BinaryMeta, error) {
	meta := BinaryMeta{}
	found := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		found = true
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "extensionname", "name":
			meta.Name = strings.TrimSpace(value)
		case "extensiontitle", "title":
			meta.Title = strings.TrimSpace(value)
		case "extensiondescription", "description":
			meta.Description = strings.TrimSpace(value)
		case "extensionversion", "version":
			meta.Version = strings.TrimSpace(value)
		case "updateurlbase", "update_url_base":
			meta.UpdateURLBase = strings.TrimSpace(value)
		case "defaultadminmenu", "adminmenuname", "admin_menu_name":
			meta.AdminMenuName = strings.TrimSpace(value)
		}
	}
	if !found {
		return BinaryMeta{}, fmt.Errorf("no key=value metadata in --data=all output")
	}
	return meta, nil
}

// enrichBinaryMeta syncs registry fields from each on-disk binary via --data=all.
// Always probe when the binary exists so relocated updateURLBase values and version
// bumps overwrite stale registry rows (missing-only sync left wrong URLs stuck).
func (m *Manager) enrichBinaryMeta(ctx context.Context, db *gorm.DB) error {
	var rows []models.Extension
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		row := &rows[i]
		if _, err := os.Stat(m.binaryPath(row.Name)); err != nil {
			continue
		}
		probed, err := m.ProbeBinaryMeta(row.Name)
		if err != nil {
			continue
		}
		if err := m.applyCachedMeta(ctx, row, Meta{
			Name:          probed.Name,
			Title:         probed.Title,
			Description:   probed.Description,
			Version:       probed.Version,
			UpdateURLBase: probed.UpdateURLBase,
		}); err != nil {
			return err
		}
		if err := m.applyDefaultMenuName(ctx, row, probed.AdminMenuName); err != nil {
			return err
		}
	}
	return nil
}
