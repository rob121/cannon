package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/sites"
)

func seedPublicGroup(t *testing.T, ctx context.Context) models.Group {
	t.Helper()
	db, err := sites.DB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var g models.Group
	if err := db.Where("name = ?", "public").First(&g).Error; err != nil {
		g = models.Group{Name: "public", Status: models.StatusActive, Kind: models.GroupKindFrontend}
		if err := db.Create(&g).Error; err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func TestCategoriesTreeAndItems(t *testing.T) {
	ctx := apiTestCtx(t)
	db, err := sites.DB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parent := models.Category{Name: "News", Slug: "news", Status: models.StatusActive}
	child := models.Category{Name: "Local", Slug: "news/local", Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child.ParentID = &parent.CategoryID
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	item := models.Item{
		Title: "Headline", Slug: "headline", Status: models.ItemStatusPublished,
		CategoryID: &child.CategoryID,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := cms.CategoryTree(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tree := buildCategoryTreeJSON(ctx, rows, false)
	if len(tree) == 0 {
		t.Fatal("expected category tree")
	}
	var foundChild bool
	for _, root := range tree {
		if root["slug"] == "news" {
			children, _ := root["children"].([]map[string]any)
			if len(children) == 1 && children[0]["slug"] == "news/local" {
				foundChild = true
			}
		}
	}
	if !foundChild {
		t.Fatalf("expected nested child in tree: %#v", tree)
	}

	viewerGroups, _ := groups.ViewerGroupIDs(ctx)
	opts := cms.ListOptions{Page: 1, Limit: 10}
	if err := setCategoryDescendantFilter(ctx, parent.CategoryID, &opts); err != nil {
		t.Fatal(err)
	}
	items, total, err := cms.ListItems(ctx, viewerGroups, opts)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].Slug != "headline" {
		t.Fatalf("expected headline in category descendants, got total=%d items=%d", total, len(items))
	}
}

func TestMenusForViewer(t *testing.T) {
	ctx := apiTestCtx(t)
	db, err := sites.DB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	route := models.Route{Name: "Home", Path: "/", Type: models.RouteTypeURL, Status: models.StatusActive}
	if err := db.Create(&route).Error; err != nil {
		t.Fatal(err)
	}
	routeID := route.RouteID
	item := models.MenuItem{MenuID: menu.MenuID, Name: "Home", RouteID: &routeID, Sort: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	public := seedPublicGroup(t, ctx)
	if err := db.Model(&item).Association("Groups").Append(&public); err != nil {
		t.Fatal(err)
	}
	viewerGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	items, err := router.MenuDataForViewer(ctx, "main", viewerGroups)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["Name"] != "Home" || items[0]["Href"] != "/" {
		t.Fatalf("unexpected menu data: %#v", items)
	}
}

func TestServeCategoriesAndMenusHTTP(t *testing.T) {
	ctx := apiTestCtx(t)
	seedPublicGroup(t, ctx)
	db, err := sites.DB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Category{Name: "Root", Slug: "root", Status: models.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Menu{MenuName: "footer", Status: models.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	_, token, err := IssueCredential(ctx, "HTTP test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories?roots=1", nil)
	req.Header.Set("X-Cannon-API-Key", token)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("categories status %d body %s", rec.Code, rec.Body.String())
	}
	var catResp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catResp); err != nil {
		t.Fatal(err)
	}
	if len(catResp.Data) == 0 {
		t.Fatal("expected root categories")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/menus", nil)
	req.Header.Set("X-Cannon-API-Key", token)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("menus status %d", rec.Code)
	}
}
