package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routemeta"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/user"
)

// Context carries request-scoped data for controller actions.
type Context struct {
	Request      *http.Request
	Route        models.Route
	Action       ActionDefinition
	Site         *config.SiteConfig
	User         *user.Service
	ViewerGroups []uint
	Template     *templateengine.Engine
}

// NewContext builds a controller context from the current request.
func NewContext(r *http.Request, route models.Route, action ActionDefinition, tpl *templateengine.Engine, viewerGroups []uint) (*Context, error) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return nil, err
	}
	svc, _ := user.FromContext(r.Context())
	return &Context{
		Request:      r,
		Route:        route,
		Action:       action,
		Site:         site,
		User:         svc,
		ViewerGroups: viewerGroups,
		Template:     tpl,
	}, nil
}

func (c *Context) GoContext() context.Context {
	return c.Request.Context()
}

func (c *Context) Authenticated() bool {
	if c.User == nil {
		return false
	}
	_, ok := c.User.CurrentID()
	return ok
}

func (c *Context) CurrentUser() (*models.User, error) {
	if c.User == nil {
		return nil, user.ErrNotAuthenticated
	}
	return c.User.Current(c.GoContext())
}

func (c *Context) CanView(contentGroups []models.Group) bool {
	return groups.CanView(c.ViewerGroups, contentGroups)
}

// PathSuffix returns the wildcard segment after a route pattern ending in /*.
func (c *Context) PathSuffix() string {
	return PathSuffix(c.Route.Path, c.Request.URL.Path)
}

// RouteMeta returns a string value from the route placement metadata.
func (c *Context) RouteMeta(key string) string {
	return routemeta.MetadataString(c.Route.Metadata, key)
}

// PathSuffix extracts the trailing segment for wildcard controller routes.
func PathSuffix(routePath, requestPath string) string {
	if !strings.HasSuffix(routePath, "/*") {
		return ""
	}
	prefix := strings.TrimSuffix(routePath, "/*")
	if requestPath == prefix {
		return ""
	}
	if strings.HasPrefix(requestPath, prefix+"/") {
		return strings.TrimPrefix(requestPath, prefix+"/")
	}
	return ""
}
