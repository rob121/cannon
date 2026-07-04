package content

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func (c *Controller) handlePreview(ctx *controllers.Context) controllers.Result {
	token := strings.Trim(ctx.PathSuffix(), "/")
	if token == "" {
		return controllers.Error(http.StatusNotFound, "preview not found")
	}
	item, err := cms.ItemByPreviewToken(ctx.GoContext(), token)
	if err != nil {
		if errors.Is(err, cms.ErrPreviewInvalid) {
			return controllers.Error(http.StatusNotFound, "preview link is invalid or expired")
		}
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if err := db.Preload("Tags").Preload("Category").Preload("Author").First(item, item.ItemID).Error; err != nil {
		return controllers.Error(http.StatusNotFound, "item not found")
	}
	fieldDisplays, _ := cms.ItemFieldDisplays(ctx.GoContext(), item)
	settings, _ := cms.LoadSettings(ctx.GoContext())
	bodyHTML, _ := cms.RichTextToHTML(item.Body)
	introHTML, _ := cms.RichTextToHTML(item.Intro)
	gallery := cms.ParseGalleryJSON(item.GalleryJSON)
	embeds := cms.ParseEmbedsJSON(item.EmbedJSON)
	attachments := cms.ParseAttachmentsJSON(item.AttachmentsJSON)
	renderArgs := map[string]any{
		"item":           item,
		"field_displays": fieldDisplays,
		"comments":       []models.Comment{},
		"BodyHTML":       bodyHTML,
		"IntroHTML":      introHTML,
	}
	if _, err := hooks.Fire(ctx.GoContext(), hooks.OnItemBeforeRender, renderArgs); err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	var authorProfile *cms.AuthorProfile
	if settings.ShowAuthorBio && item.AuthorID != nil {
		authorProfile, _ = cms.LoadAuthorProfile(ctx.GoContext(), *item.AuthorID)
	}
	data := map[string]any{
		"Item":            item,
		"FieldDisplays":   fieldDisplays,
		"Comments":        []models.Comment{},
		"CommentCount":    0,
		"CommentSettings": settings,
		"ContentSettings": settings,
		"AuthorProfile":   authorProfile,
		"CanEdit":         false,
		"PreviewMode":     true,
		"BodyHTML":        template.HTML(bodyHTML),
		"IntroHTML":       template.HTML(introHTML),
		"Gallery":         gallery,
		"Embeds":          embeds,
		"Attachments":     attachments,
	}
	return controllers.HTMLPage(item.Title+" (Preview)", "default/controllers/content/item.html", data)
}
