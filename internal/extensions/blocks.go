package extensions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
)

func (m *Manager) fetchBlocks(socketPath, blockBase string) ([]extension.BlockDefinition, error) {
	path := capabilityPath(blockBase, "")
	resp, err := m.do(socketPath, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var payload extension.BlockListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Blocks, nil
}

// SpaceHasExtensionBlock reports whether any active extension could render the space
// when no admin blocks are assigned (without invoking the extension).
func (m *Manager) SpaceHasExtensionBlock(space string) bool {
	space = strings.TrimSpace(space)
	if space == "" {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, rt := range m.runtimes {
		if rt.Capabilities.Block == "" {
			continue
		}
		if _, ok := MatchBlock(rt.Blocks, space); ok {
			return true
		}
		if len(rt.Blocks) == 0 {
			return true
		}
	}
	return false
}

// MatchBlock picks a block id for a template space using extension block definitions.
func MatchBlock(blocks []extension.BlockDefinition, space string) (string, bool) {
	space = strings.TrimSpace(space)
	if space == "" || len(blocks) == 0 {
		return "", false
	}

	var wildcard string
	for _, block := range blocks {
		if block.ID == space {
			return block.ID, true
		}
		if len(block.Spaces) == 0 {
			if wildcard == "" {
				wildcard = block.ID
			}
			continue
		}
		for _, candidate := range block.Spaces {
			if candidate == space {
				return block.ID, true
			}
		}
	}
	if wildcard != "" {
		return wildcard, true
	}
	if len(blocks) == 1 {
		return blocks[0].ID, true
	}
	return "", false
}

// BlockRuntimes returns active extensions that expose /block.
func (m *Manager) BlockRuntimes() []*Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Runtime, 0)
	for _, rt := range m.runtimes {
		if rt.Capabilities.Block != "" {
			out = append(out, rt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Model.Sort < out[j].Model.Sort })
	return out
}
