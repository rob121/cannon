package admin

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func parseSort(r *http.Request, allowed map[string]string, defaultCol string) (col, dir, orderBy string) {
	col = strings.TrimSpace(r.URL.Query().Get("sort"))
	dir = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dir")))
	if dir != "desc" {
		dir = "asc"
	}
	dbCol, ok := allowed[col]
	if !ok {
		col = defaultCol
		dbCol = allowed[defaultCol]
		dir = "asc"
	}
	return col, dir, dbCol + " " + dir
}

func applyListSort(r *http.Request, data map[string]any, allowed map[string]string, defaultCol string) string {
	col, dir, orderBy := parseSort(r, allowed, defaultCol)
	data["Sort"] = col
	data["Dir"] = dir
	return orderBy
}

func listQuery(page int, sort, dir string) string {
	return listQueryExtra(page, sort, dir, nil)
}

func listQueryExtra(page int, sort, dir string, extra url.Values) string {
	v := listQueryValues(page, sort, dir)
	for k, vals := range extra {
		if k == "page" || k == "sort" || k == "dir" {
			continue
		}
		for _, val := range vals {
			v.Set(k, val)
		}
	}
	if s := v.Encode(); s != "" {
		return "?" + s
	}
	return ""
}

// listQueryAmp appends list query params to an existing query string.
func listQueryAmp(page int, sort, dir string) string {
	return listQueryAmpExtra(page, sort, dir, nil)
}

func listQueryAmpExtra(page int, sort, dir string, extra url.Values) string {
	if s := strings.TrimPrefix(listQueryExtra(page, sort, dir, extra), "?"); s != "" {
		return "&" + s
	}
	return ""
}

func listQueryValues(page int, sort, dir string) url.Values {
	v := url.Values{}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	if sort != "" {
		v.Set("sort", sort)
		v.Set("dir", dir)
	}
	return v
}

func sortLink(basePath string, page int, currentSort, currentDir, col string) string {
	return sortLinkExtra(basePath, page, currentSort, currentDir, col, nil)
}

func sortLinkExtra(basePath string, page int, currentSort, currentDir, col string, extra url.Values) string {
	dir := "asc"
	if currentSort == col && currentDir == "asc" {
		dir = "desc"
	}
	return basePath + listQueryExtra(page, col, dir, extra)
}

func sortLess(a, b string, dir string) bool {
	if dir == "desc" {
		return a > b
	}
	return a < b
}

func sortLessInt(a, b int, dir string) bool {
	if dir == "desc" {
		return a > b
	}
	return a < b
}

func sortLessInt64(a, b int64, dir string) bool {
	if dir == "desc" {
		return a > b
	}
	return a < b
}

func sortLessBool(a, b bool, dir string) bool {
	if dir == "desc" {
		return boolRank(a) > boolRank(b)
	}
	return boolRank(a) < boolRank(b)
}

func boolRank(v bool) int {
	if v {
		return 1
	}
	return 0
}

func sortStrings(items []string, dir string) {
	sort.Slice(items, func(i, j int) bool {
		return sortLess(items[i], items[j], dir)
	})
}
