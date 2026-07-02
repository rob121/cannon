package content_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
)

func TestItemMediaFromForm(t *testing.T) {
	form := url.Values{}
	form.Add("gallery_url", "https://example.com/a.jpg")
	form.Add("gallery_alt", "Photo A")
	form.Add("embed_url", "https://example.com/video.mp4")
	form.Add("embed_kind", "video")
	form.Add("attachment_url", "https://example.com/file.pdf")
	form.Add("attachment_label", "Spec sheet")

	req := &http.Request{Method: http.MethodPost, Form: form}
	gallery, embeds, attachments := content.ItemMediaFromForm(req)

	if !strings.Contains(gallery, "Photo A") {
		t.Fatalf("gallery json: %s", gallery)
	}
	if !strings.Contains(embeds, "video") {
		t.Fatalf("embeds json: %s", embeds)
	}
	if !strings.Contains(attachments, "Spec sheet") {
		t.Fatalf("attachments json: %s", attachments)
	}

	images := content.ParseGalleryJSON(gallery)
	if len(images) != 1 || images[0].URL != "https://example.com/a.jpg" {
		t.Fatalf("parsed gallery: %+v", images)
	}
}

func TestValidateRequiredCustomFields(t *testing.T) {
	fields := []models.ContentField{
		{FieldID: 1, Name: "sku", Label: "SKU", Type: "text", Required: true},
	}
	req := &http.Request{Method: http.MethodPost, Form: url.Values{}}
	if err := content.ValidateRequiredCustomFields(fields, req); err == nil {
		t.Fatal("expected required field error")
	}

	req.Form.Set("field_1", "ABC-123")
	if err := content.ValidateRequiredCustomFields(fields, req); err != nil {
		t.Fatal(err)
	}
}

func TestCategoryDescendantIDs(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parent := models.Category{Name: "Parent", Slug: "parent-descendants", Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-descendants", ParentID: &parent.CategoryID, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	grand := models.Category{Name: "Grand", Slug: "grand-descendants", ParentID: &child.CategoryID, Status: models.StatusActive}
	if err := db.Create(&grand).Error; err != nil {
		t.Fatal(err)
	}

	ids, err := content.CategoryDescendantIDs(ctx, parent.CategoryID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 {
		t.Fatalf("descendant ids = %v, want 3", ids)
	}
}

func TestFieldGroupForCategoryInheritsFromParent(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	groupID := uint(42)
	parent := models.Category{Name: "Parent", Slug: "parent-fields", FieldGroupID: &groupID, InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-fields", ParentID: &parent.CategoryID, InheritSettings: true, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}

	got, err := content.FieldGroupForCategory(ctx, &child)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != groupID {
		t.Fatalf("field group = %v, want %d", got, groupID)
	}
}
