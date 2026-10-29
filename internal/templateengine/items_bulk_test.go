package templateengine

import (
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/themes"
	"golang.org/x/net/html"
)

func TestItemsBulkFormOwnership(t *testing.T) {
	funcs := TestAdminFuncs()
	funcs["csrfField"] = func() template.HTML { return `<input type="hidden" name="csrf_token" value="test-token">` }
	e := New(&config.SiteConfig{}, themes.Selection{}, nil, nil, funcs)
	tmpl, err := e.parse("admin/items.html")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	err = tmpl.Execute(&out, map[string]any{
		"BasePath": "/admin/items", "ListQuery": "?dir=asc&page_size=25&sort=sort",
		"Page": 1, "PageSize": 25, "Total": int64(0), "Sort": "sort", "Dir": "asc",
		"Rows": []map[string]any{
			{"ItemID": 8, "Title": "First", "Status": "draft", "CanMoveDown": true},
			{"ItemID": 9, "Title": "Second", "Status": "draft", "CanMoveUp": true, "CanMoveDown": true, "Featured": true, "CanFeaturedMoveDown": true},
			{"ItemID": 10, "Title": "Third", "Status": "draft", "CanMoveUp": true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Browsers discard nested form start tags and close the bulk form early.
	z := html.NewTokenizer(strings.NewReader(out.String()))
	var formID, action string
	inForm := false
	var selectedIDs []string
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				t.Fatal(z.Err())
			}
			break
		}
		token := z.Token()
		if kind == html.EndTagToken && token.Data == "form" {
			inForm, formID, action = false, "", ""
			continue
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range token.Attr {
			attrs[attr.Key] = attr.Val
		}
		if token.Data == "form" {
			if inForm {
				t.Fatalf("nested form %q inside %q breaks bulk submissions", attrs["action"], action)
			}
			inForm, formID, action = true, attrs["id"], attrs["action"]
			if strings.Contains(attrs["class"], "admin-bulk-form") {
				if formID != "items-bulk-form" || action != "/admin/items/bulk?dir=asc&page_size=25&sort=sort" {
					t.Errorf("bulk form ID/action = %q / %q", formID, action)
				}
			}
		}
		if attrs["name"] == "item_ids" {
			owner := attrs["form"]
			if owner == "" {
				owner = formID
			}
			if owner != "items-bulk-form" {
				t.Errorf("item %s belongs to form %q, want bulk form", attrs["value"], owner)
			}
			selectedIDs = append(selectedIDs, attrs["value"])
		}
	}
	if strings.Join(selectedIDs, ",") != "8,9,10" {
		t.Fatalf("item checkboxes = %v", selectedIDs)
	}
}
