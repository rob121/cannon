package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

type authorJSON struct {
	UserID      uint   `json:"user_id,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

type tagJSON struct {
	TagID uint   `json:"tag_id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
}

type categoryJSON struct {
	CategoryID uint   `json:"category_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
}

type itemJSON struct {
	ItemID             uint           `json:"item_id"`
	Locale             string         `json:"locale"`
	TranslationGroupID *uint          `json:"translation_group_id,omitempty"`
	Title              string         `json:"title"`
	Slug               string         `json:"slug"`
	Intro              string         `json:"intro,omitempty"`
	Body               string         `json:"body,omitempty"`
	Featured           bool           `json:"featured"`
	PublishStart       *time.Time     `json:"publish_start,omitempty"`
	PublishEnd         *time.Time     `json:"publish_end,omitempty"`
	Author             *authorJSON    `json:"author,omitempty"`
	Category           *categoryJSON  `json:"category,omitempty"`
	Tags               []tagJSON      `json:"tags,omitempty"`
	Image              string         `json:"image,omitempty"`
	Gallery            []any          `json:"gallery,omitempty"`
	Embeds             []any          `json:"embeds,omitempty"`
	Attachments        []any          `json:"attachments,omitempty"`
	Meta               map[string]any `json:"meta,omitempty"`
	CustomFields       map[string]any `json:"custom_fields,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func itemToJSON(ctx context.Context, item *models.Item) (itemJSON, error) {
	out := itemJSON{
		ItemID:             item.ItemID,
		Locale:             item.Locale,
		TranslationGroupID: item.TranslationGroupID,
		Title:              item.Title,
		Slug:               item.Slug,
		Intro:              item.Intro,
		Body:               item.Body,
		Featured:           item.Featured,
		PublishStart:       item.PublishStart,
		PublishEnd:         item.PublishEnd,
		Image:              item.Image,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
	if item.Author != nil && item.Author.UserID > 0 {
		avatar, _ := cms.ResolveUserAvatar(ctx, item.Author.UserID)
		out.Author = &authorJSON{
			UserID:      item.Author.UserID,
			Username:    item.Author.Username,
			DisplayName: user.DisplayName(item.Author),
			AvatarURL:   avatar,
		}
	}
	if item.Category != nil && item.Category.CategoryID > 0 {
		out.Category = &categoryJSON{
			CategoryID: item.Category.CategoryID,
			Name:       item.Category.Name,
			Slug:       item.Category.Slug,
		}
	}
	for _, t := range item.Tags {
		out.Tags = append(out.Tags, tagJSON{TagID: t.TagID, Name: t.Name, Slug: t.Slug})
	}
	_ = decodeJSONField(item.GalleryJSON, &out.Gallery)
	_ = decodeJSONField(item.EmbedJSON, &out.Embeds)
	_ = decodeJSONField(item.AttachmentsJSON, &out.Attachments)
	meta := map[string]any{}
	if item.MetaTitle != "" {
		meta["title"] = item.MetaTitle
	}
	if item.MetaDescription != "" {
		meta["description"] = item.MetaDescription
	}
	if item.MetaKeywords != "" {
		meta["keywords"] = item.MetaKeywords
	}
	if item.CanonicalURL != "" {
		meta["canonical_url"] = item.CanonicalURL
	}
	if len(meta) > 0 {
		out.Meta = meta
	}
	fields, err := itemCustomFields(ctx, item.ItemID)
	if err == nil && len(fields) > 0 {
		out.CustomFields = fields
	}
	return out, nil
}

func itemCustomFields(ctx context.Context, itemID uint) (map[string]any, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var values []models.ItemFieldValue
	if err := db.Where("item_id = ?", itemID).Find(&values).Error; err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	out := map[string]any{}
	for _, v := range values {
		out[strconv.FormatUint(uint64(v.FieldID), 10)] = v.Value
	}
	return out, nil
}

func decodeJSONField(raw string, dst any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), dst)
}

func loadVisibleItemByID(ctx context.Context, id uint, viewerGroups []uint) (*models.Item, error) {
	q, err := cms.VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return nil, err
	}
	var item models.Item
	if err := q.Where("items.item_id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	if !groups.CanViewContent(viewerGroups, item.Groups) {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func mediaVisibleOnItems(ctx context.Context, media *models.MediaAsset, viewerGroups []uint) (bool, error) {
	if media == nil {
		return false, nil
	}
	q, err := cms.VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return false, err
	}
	path := strings.TrimSpace(media.Path)
	name := strings.TrimSpace(media.Name)
	var count int64
	err = q.Where(
		"items.image LIKE ? OR items.gallery_json LIKE ? OR items.attachments_json LIKE ? OR items.body LIKE ?",
		"%"+path+"%", "%"+path+"%", "%"+path+"%", "%"+path+"%",
	).Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if name != "" {
		err = q.Where(
			"items.gallery_json LIKE ? OR items.attachments_json LIKE ? OR items.body LIKE ?",
			"%"+name+"%", "%"+name+"%", "%"+name+"%",
		).Count(&count).Error
		return count > 0, err
	}
	return false, nil
}

func parsePageQuery(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20
	if v := strings.TrimSpace(r.URL.Query().Get("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("page_size")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func groupNamesForUser(ctx context.Context, userID uint) ([]string, error) {
	ids, err := groups.UserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Group
	if err := db.Where("group_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, g := range rows {
		names = append(names, g.Name)
	}
	return names, nil
}
