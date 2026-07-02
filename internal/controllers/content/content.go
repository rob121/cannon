package content

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/markdown"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const ControllerID = "content"

const (
	ActionIndex    = "index"
	ActionCategory = "category"
	ActionItem     = "item"
	ActionTag      = "tag"
	ActionAuthor   = "author"
	ActionSearch   = "search"
	ActionFeed     = "feed"
	ActionEditNew  = "edit-new"
	ActionEdit     = "edit"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

func Definition() controllers.Definition {
	return controllers.Definition{
		ID:          ControllerID,
		Title:       "Content",
		Description: "Items, categories, tags, author listings, search, feeds, and frontend editing.",
		Actions: []controllers.ActionDefinition{
			{ID: ActionIndex, Title: "Home", Methods: []string{http.MethodGet}, DefaultPath: "/"},
			{ID: ActionCategory, Title: "Category", Methods: []string{http.MethodGet}, DefaultPath: "/content/category/*"},
			{ID: ActionItem, Title: "Item", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/content/item/*"},
			{ID: ActionTag, Title: "Tag", Methods: []string{http.MethodGet}, DefaultPath: "/content/tag/*"},
			{ID: ActionAuthor, Title: "Author", Methods: []string{http.MethodGet}, DefaultPath: "/content/author/*"},
			{ID: ActionSearch, Title: "Search", Methods: []string{http.MethodGet}, DefaultPath: "/content/search"},
			{ID: ActionFeed, Title: "Feed", Methods: []string{http.MethodGet}, DefaultPath: "/content/feed/*"},
			{ID: ActionEditNew, Title: "Create Item", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/content/edit/new", RequireAuth: true},
			{ID: ActionEdit, Title: "Edit Item", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/content/edit/*", RequireAuth: true},
		},
	}
}

func (c *Controller) Handle(ctx *controllers.Context, actionID string) controllers.Result {
	switch actionID {
	case ActionIndex:
		return c.handleIndex(ctx)
	case ActionCategory:
		return c.handleCategory(ctx)
	case ActionItem:
		return c.handleItem(ctx)
	case ActionTag:
		return c.handleTag(ctx)
	case ActionAuthor:
		return c.handleAuthor(ctx)
	case ActionSearch:
		return c.handleSearch(ctx)
	case ActionFeed:
		return c.handleFeed(ctx)
	case ActionEditNew:
		return c.handleEditNew(ctx)
	case ActionEdit:
		return c.handleEdit(ctx)
	default:
		return controllers.Error(http.StatusNotFound, "unknown content action")
	}
}

func (c *Controller) handleIndex(ctx *controllers.Context) controllers.Result {
	items, _, err := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, cms.ListOptions{
		Page:  1,
		Limit: 12,
		Sort:  "sort",
	})
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	featured, _, _ := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, cms.ListOptions{
		Featured: true,
		Page:     1,
		Limit:    6,
	})
	categories, _ := cms.CategoryTree(ctx.GoContext())
	return controllers.HTML("Home", map[string]any{
		"RouteName":  ctx.Route.Name,
		"Items":      items,
		"Featured":   featured,
		"Categories": categories,
	})
}

func (c *Controller) handleCategory(ctx *controllers.Context) controllers.Result {
	slug := strings.Trim(ctx.PathSuffix(), "/")
	if slug == "" {
		return controllers.Error(http.StatusNotFound, "category not found")
	}
	cat, err := cms.CategoryBySlug(ctx.GoContext(), slug, ctx.ViewerGroups)
	if err != nil {
		return controllers.Error(http.StatusNotFound, "category not found")
	}
	page := queryPage(ctx.Request)
	items, total, err := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, cms.ListOptions{
		CategoryID: cat.CategoryID,
		Page:       page,
		Limit:      20,
	})
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	categories, _ := cms.CategoryTree(ctx.GoContext())
	data := map[string]any{
		"Category":   cat,
		"Items":      items,
		"Total":      total,
		"Page":       page,
		"Categories": categories,
	}
	if tpl, err := cms.CategoryTemplate(ctx.GoContext(), cat); err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	} else if tpl != "" {
		return controllers.HTMLPage(cat.Name, tpl, data)
	}
	return controllers.HTML(cat.Name, data)
}

func (c *Controller) handleItem(ctx *controllers.Context) controllers.Result {
	slug := strings.Trim(ctx.PathSuffix(), "/")
	if slug == "" {
		return controllers.Error(http.StatusNotFound, "item not found")
	}
	if ctx.Request.Method == http.MethodPost {
		return c.handleItemComment(ctx, slug)
	}
	item, err := cms.ItemBySlug(ctx.GoContext(), slug, ctx.ViewerGroups)
	if err != nil {
		return controllers.Error(http.StatusNotFound, "item not found")
	}
	fields, _ := cms.ItemFieldMap(ctx.GoContext(), item.ItemID)
	comments, _ := cms.ApprovedComments(ctx.GoContext(), item.ItemID)
	related, _ := cms.RelatedItems(ctx.GoContext(), ctx.ViewerGroups, item, 5)
	commentCount, _ := cms.CommentCount(ctx.GoContext(), item.ItemID)
	settings, _ := cms.LoadSettings(ctx.GoContext())
	bodyHTML, _ := markdown.ToHTML(item.Body)
	introHTML, _ := markdown.ToHTML(item.Intro)
	renderArgs := map[string]any{
		"item":      item,
		"fields":    fields,
		"comments":  comments,
		"BodyHTML":  bodyHTML,
		"IntroHTML": introHTML,
	}
	if _, err := hooks.Fire(ctx.GoContext(), hooks.OnItemBeforeRender, renderArgs); err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	canEdit := false
	if ctx.Authenticated() {
		if user, err := ctx.CurrentUser(); err == nil {
			canEdit, _ = cms.CanEditItem(ctx.GoContext(), user.UserID, item)
		}
	}
	data := map[string]any{
		"Item":           item,
		"Fields":         fields,
		"Comments":       comments,
		"Related":        related,
		"CommentCount":   commentCount,
		"CommentSettings": settings,
		"CanEdit":        canEdit,
		"BodyHTML":       template.HTML(bodyHTML),
		"IntroHTML":      template.HTML(introHTML),
	}
	applyItemSEO(data, item, ctx)
	return controllers.HTML(item.Title, data)
}

func (c *Controller) handleItemComment(ctx *controllers.Context, slug string) controllers.Result {
	item, err := cms.ItemBySlug(ctx.GoContext(), slug, ctx.ViewerGroups)
	if err != nil {
		return controllers.Error(http.StatusNotFound, "item not found")
	}
	if err := ctx.Request.ParseForm(); err != nil {
		return controllers.Error(http.StatusBadRequest, err.Error())
	}
	var userID *uint
	if ctx.Authenticated() {
		if user, err := ctx.CurrentUser(); err == nil {
			userID = &user.UserID
		}
	}
	in := cms.CommentInput{
		ItemID:      item.ItemID,
		UserID:      userID,
		AuthorName:  ctx.Request.FormValue("author_name"),
		AuthorEmail: ctx.Request.FormValue("author_email"),
		Body:        ctx.Request.FormValue("body"),
		IP:          clientIP(ctx.Request),
	}
	_, err = cms.CreateComment(ctx.GoContext(), in, ctx.Authenticated())
	if err != nil {
		return controllers.Redirect(http.StatusSeeOther, cms.ItemURL(slug)+"?comment_error=1#comments")
	}
	return controllers.Redirect(http.StatusSeeOther, cms.ItemURL(slug)+"?comment_posted=1#comments")
}

func (c *Controller) handleTag(ctx *controllers.Context) controllers.Result {
	slug := strings.Trim(ctx.PathSuffix(), "/")
	if slug == "" {
		return controllers.Error(http.StatusNotFound, "tag not found")
	}
	tag, err := cms.TagBySlug(ctx.GoContext(), slug)
	if err != nil {
		return controllers.Error(http.StatusNotFound, "tag not found")
	}
	page := queryPage(ctx.Request)
	items, total, err := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, cms.ListOptions{
		TagID: tag.TagID,
		Page:  page,
		Limit: 20,
	})
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	return controllers.HTML(tag.Name, map[string]any{
		"Tag":   tag,
		"Items": items,
		"Total": total,
		"Page":  page,
	})
}

func (c *Controller) handleAuthor(ctx *controllers.Context) controllers.Result {
	key := strings.Trim(ctx.PathSuffix(), "/")
	if key == "" {
		return controllers.Error(http.StatusNotFound, "author not found")
	}
	author, err := findAuthor(ctx, key)
	if err != nil {
		return controllers.Error(http.StatusNotFound, "author not found")
	}
	page := queryPage(ctx.Request)
	items, total, err := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, cms.ListOptions{
		AuthorID: author.UserID,
		Page:     page,
		Limit:    20,
	})
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	name := authorDisplayName(author)
	return controllers.HTML(name, map[string]any{
		"Author": author,
		"Items":  items,
		"Total":  total,
		"Page":   page,
	})
}

func (c *Controller) handleSearch(ctx *controllers.Context) controllers.Result {
	query := strings.TrimSpace(ctx.Request.URL.Query().Get("q"))
	page := queryPage(ctx.Request)
	categoryID := queryUint(ctx.Request, "category")
	tagID := queryUint(ctx.Request, "tag")
	authorID := queryUint(ctx.Request, "author")
	sort := ctx.Request.URL.Query().Get("sort")
	opts := cms.ListOptions{
		Query:      query,
		CategoryID: categoryID,
		TagID:      tagID,
		AuthorID:   authorID,
		Sort:       sort,
		Page:       page,
		Limit:      20,
	}
	items, total, err := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, opts)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	categories, _ := cms.CategoryTree(ctx.GoContext())
	tags, _ := cms.ListTags(ctx.GoContext())
	title := "Search"
	if query != "" {
		title = "Search: " + query
	}
	return controllers.HTML(title, map[string]any{
		"Query":      query,
		"Items":      items,
		"Total":      total,
		"Page":       page,
		"Categories": categories,
		"Tags":       tags,
		"CategoryID": categoryID,
		"TagID":      tagID,
		"AuthorID":   authorID,
		"Sort":       sort,
	})
}

func (c *Controller) handleFeed(ctx *controllers.Context) controllers.Result {
	suffix := strings.Trim(strings.Trim(ctx.PathSuffix(), "/"), "/")
	siteURL := siteBaseURL(ctx)
	siteName := ctx.Site.Name
	feedTitle := siteName
	feedLink := siteURL + "/content/feed"
	feedDesc := "Latest content from " + siteName
	opts := cms.ListOptions{Page: 1, Limit: 25}

	parts := strings.Split(suffix, "/")
	switch {
	case suffix == "" || suffix == "global":
		// global feed
	case len(parts) >= 2 && parts[0] == "category":
		slug := strings.Join(parts[1:], "/")
		cat, err := cms.CategoryBySlug(ctx.GoContext(), slug, ctx.ViewerGroups)
		if err != nil {
			return controllers.Error(http.StatusNotFound, "category not found")
		}
		opts.CategoryID = cat.CategoryID
		feedTitle = cat.Name + " — " + siteName
		feedLink = siteURL + cms.CategoryURL(slug)
		feedDesc = "Items in " + cat.Name
	case len(parts) >= 2 && parts[0] == "tag":
		slug := strings.Join(parts[1:], "/")
		tag, err := cms.TagBySlug(ctx.GoContext(), slug)
		if err != nil {
			return controllers.Error(http.StatusNotFound, "tag not found")
		}
		opts.TagID = tag.TagID
		feedTitle = tag.Name + " — " + siteName
		feedLink = siteURL + cms.TagURL(slug)
		feedDesc = "Items tagged " + tag.Name
	case len(parts) >= 2 && parts[0] == "author":
		key := strings.Join(parts[1:], "/")
		author, err := findAuthor(ctx, key)
		if err != nil {
			return controllers.Error(http.StatusNotFound, "author not found")
		}
		opts.AuthorID = author.UserID
		name := authorDisplayName(author)
		feedTitle = name + " — " + siteName
		feedLink = siteURL + cms.AuthorURL(key)
		feedDesc = "Items by " + name
	default:
		return controllers.Error(http.StatusNotFound, "feed not found")
	}

	items, _, err := cms.ListItems(ctx.GoContext(), ctx.ViewerGroups, opts)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	body, err := cms.BuildRSS(siteName, siteURL, feedTitle, feedLink, feedDesc, items)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	return controllers.Raw(http.StatusOK, "application/rss+xml; charset=utf-8", body)
}

func applyItemSEO(data map[string]any, item *models.Item, ctx *controllers.Context) {
	metaTitle := strings.TrimSpace(item.MetaTitle)
	if metaTitle == "" {
		metaTitle = item.Title
	}
	metaDesc := strings.TrimSpace(item.MetaDescription)
	if metaDesc == "" {
		metaDesc = strings.TrimSpace(item.Intro)
	}
	canonical := strings.TrimSpace(item.CanonicalURL)
	if canonical == "" {
		canonical = siteBaseURL(ctx) + cms.ItemURL(item.Slug)
	}
	ogImage := strings.TrimSpace(item.Image)
	data["MetaTitle"] = metaTitle
	data["MetaDescription"] = metaDesc
	data["MetaKeywords"] = strings.TrimSpace(item.MetaKeywords)
	data["CanonicalURL"] = canonical
	data["OGImage"] = ogImage
}

func siteBaseURL(ctx *controllers.Context) string {
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	if fwd := ctx.Request.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host := ctx.Request.Host
	if host == "" && ctx.Site.Host != "" {
		host = ctx.Site.Host
	}
	return scheme + "://" + strings.TrimRight(host, "/")
}

func findAuthor(ctx *controllers.Context, key string) (*models.User, error) {
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return nil, err
	}
	var user models.User
	if id, convErr := strconv.ParseUint(key, 10, 64); convErr == nil {
		err = db.Where("user_id = ? AND status = ?", id, models.StatusActive).First(&user).Error
	} else {
		err = db.Where("username = ? AND status = ?", key, models.StatusActive).First(&user).Error
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func authorDisplayName(user *models.User) string {
	if user == nil {
		return ""
	}
	name := strings.TrimSpace(strings.TrimSpace(user.GivenName + " " + user.FamilyName))
	if name == "" {
		return user.Username
	}
	return name
}

func queryPage(r *http.Request) int {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		return 1
	}
	return page
}

func queryUint(r *http.Request, key string) uint {
	v, _ := strconv.ParseUint(r.URL.Query().Get(key), 10, 64)
	return uint(v)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
