package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func (m *Manager) runInstall(ctx context.Context, socketPath string) error {
	wire := WireRequest{
		Method: http.MethodPost,
		URL:    "/install",
		SiteID: m.site.ID,
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		return err
	}

	resp, err := m.do(socketPath, http.MethodPost, "/install", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var out WireResponse
	if err := json.Unmarshal(body, &out); err == nil {
		status := out.StatusCode
		if status == 0 {
			status = resp.StatusCode
		}
		if status >= 300 || out.Stop {
			msg := out.Body
			if msg == "" {
				msg = fmt.Sprintf("status %d", status)
			}
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}

func (m *Manager) markInstalled(ctx context.Context, name string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	return db.Model(&models.Extension{}).Where("name = ?", name).Update("installed", true).Error
}
