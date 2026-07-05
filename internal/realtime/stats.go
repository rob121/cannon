package realtime

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/centrifugal/centrifuge"
)

// PageStat counts visitors on one path.
type PageStat struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// Stats is the live analytics payload published to admin subscribers.
type Stats struct {
	Online        int        `json:"online"`
	Authenticated int        `json:"authenticated"`
	Pages         []PageStat `json:"pages"`
	UpdatedAt     string     `json:"updated_at"`
}

type presenceMeta struct {
	Page string `json:"page"`
}

func buildStats(result centrifuge.PresenceResult) Stats {
	pageCounts := map[string]int{}
	online := 0
	auth := 0
	for _, info := range result.Presence {
		if info == nil {
			continue
		}
		online++
		if strings.HasPrefix(info.UserID, "user:") {
			auth++
		}
		path := "/"
		if len(info.ChanInfo) > 0 {
			var meta presenceMeta
			if err := json.Unmarshal(info.ChanInfo, &meta); err == nil {
				path = strings.TrimSpace(meta.Page)
				if path == "" {
					path = "/"
				}
			}
		}
		pageCounts[path]++
	}
	pages := make([]PageStat, 0, len(pageCounts))
	for path, count := range pageCounts {
		pages = append(pages, PageStat{Path: path, Count: count})
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Count == pages[j].Count {
			return pages[i].Path < pages[j].Path
		}
		return pages[i].Count > pages[j].Count
	})
	return Stats{
		Online:        online,
		Authenticated: auth,
		Pages:         pages,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
}

func statsPayload(result centrifuge.PresenceResult) ([]byte, error) {
	return json.Marshal(buildStats(result))
}
