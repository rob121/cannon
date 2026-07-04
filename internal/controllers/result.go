package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/routemeta"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
)

// Result is a controller response.
type Result interface {
	Write(w http.ResponseWriter, r *http.Request, ctx *Context) error
}

type redirectResult struct {
	URL  string
	Code int
}

func Redirect(code int, url string) Result {
	if code == 0 {
		code = http.StatusSeeOther
	}
	return redirectResult{URL: url, Code: code}
}

func (r redirectResult) Write(w http.ResponseWriter, req *http.Request, _ *Context) error {
	if r.Code == http.StatusSeeOther {
		httpx.RedirectSeeOther(w, req, r.URL)
		return nil
	}
	httpx.Redirect(w, req, r.URL)
	return nil
}

type htmlResult struct {
	Title        string
	Data         map[string]any
	PageTemplate string
}

func HTML(title string, data map[string]any) Result {
	if data == nil {
		data = map[string]any{}
	}
	if title != "" {
		data["Title"] = title
	}
	return htmlResult{Title: title, Data: data}
}

func HTMLPage(title, pageTemplate string, data map[string]any) Result {
	if data == nil {
		data = map[string]any{}
	}
	if title != "" {
		data["Title"] = title
	}
	return htmlResult{Title: title, Data: data, PageTemplate: pageTemplate}
}

func (r htmlResult) Write(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if ctx.Template == nil {
		return fmt.Errorf("template engine missing")
	}
	controllerID := ctx.Route.Controller
	actionID := ctx.Route.ControllerAction
	ctx.Template.SetHookContext(req.Context())
	displayArgs := map[string]any{
		"title":      r.Title,
		"controller": controllerID,
		"action":     actionID,
		"data":       r.Data,
	}
	if out, err := hooks.Fire(req.Context(), hooks.OnContentBeforeDisplay, displayArgs); err != nil {
		return err
	} else if m, ok := out["data"].(map[string]any); ok {
		r.Data = m
	}
	page := templateengine.ControllerTemplatePath(controllerID, actionID)
	if override := routemeta.MetadataString(ctx.Route.Metadata, "template"); override != "" {
		page = override
	}
	if r.PageTemplate != "" {
		page = r.PageTemplate
	}
	err := ctx.Template.Render(w, "default/layout.html", page, r.Data)
	ctx.Template.SetHookContext(nil)
	return err
}

type errorResult struct {
	Code    int
	Message string
}

func Error(code int, message string) Result {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	return errorResult{Code: code, Message: message}
}

func (r errorResult) Write(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if ctx != nil && ctx.Template != nil {
		homeURL := sites.DefaultRoutePath(req.Context())
		message := strings.TrimSpace(r.Message)
		if message == "" {
			message = templateengine.DefaultErrorMessage(r.Code)
		}
		ctx.Template.SetHookContext(req.Context())
		defer ctx.Template.SetHookContext(nil)
		return ctx.Template.RenderError(w, r.Code, map[string]any{
			"Title":        templateengine.ErrorTitle(r.Code),
			"ErrorCode":    r.Code,
			"ErrorMessage": message,
			"HomeURL":      homeURL,
		})
	}
	http.Error(w, r.Message, r.Code)
	return nil
}

type statusResult struct {
	Code int
}

func Status(code int) Result {
	return statusResult{Code: code}
}

func (r statusResult) Write(w http.ResponseWriter, _ *http.Request, _ *Context) error {
	w.WriteHeader(r.Code)
	return nil
}

type rawResult struct {
	Code        int
	ContentType string
	Body        []byte
}

func Raw(code int, contentType string, body []byte) Result {
	if code == 0 {
		code = http.StatusOK
	}
	return rawResult{Code: code, ContentType: contentType, Body: body}
}

func (r rawResult) Write(w http.ResponseWriter, _ *http.Request, _ *Context) error {
	if r.ContentType != "" {
		w.Header().Set("Content-Type", r.ContentType)
	}
	w.WriteHeader(r.Code)
	if len(r.Body) > 0 {
		_, _ = w.Write(r.Body)
	}
	return nil
}

// JSON encodes data as a JSON response.
func JSON(code int, data any) Result {
	if code == 0 {
		code = http.StatusOK
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return Error(http.StatusInternalServerError, err.Error())
	}
	return Raw(code, "application/json", raw)
}
