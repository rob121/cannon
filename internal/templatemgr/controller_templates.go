package templatemgr

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/themes"
)

// ControllerTemplateOption is a selectable controller page template for route overrides.
type ControllerTemplateOption struct {
	Path       string
	Controller string
	Action     string
	Label      string
}

// ListControllerTemplateOptions returns logical template paths for one controller.
// Paths use the default/ prefix expected by the template engine, e.g.
// default/controllers/content/index.html.
func ListControllerTemplateOptions(templateDir, frontendTheme, controllerID string, actionLabels map[string]string) ([]ControllerTemplateOption, error) {
	controllerID = strings.TrimSpace(controllerID)
	if controllerID == "" {
		return nil, nil
	}
	frontendTheme = strings.TrimSpace(frontendTheme)

	relPrefix := "controllers/" + controllerID + "/"
	logicalPrefix := "default/controllers/" + controllerID + "/"
	seen := map[string]struct{}{}
	out := make([]ControllerTemplateOption, 0)

	add := func(logicalPath, source string) {
		logicalPath = filepath.ToSlash(strings.TrimSpace(logicalPath))
		if !strings.HasPrefix(logicalPath, logicalPrefix) || !strings.HasSuffix(strings.ToLower(logicalPath), ".html") {
			return
		}
		if _, ok := seen[logicalPath]; ok {
			return
		}
		seen[logicalPath] = struct{}{}
		action := strings.TrimSuffix(filepath.Base(logicalPath), ".html")
		label := action
		if title := strings.TrimSpace(actionLabels[action]); title != "" {
			label = title + " (" + action + ")"
		}
		if source != "" {
			label += " · " + source
		}
		out = append(out, ControllerTemplateOption{
			Path:       logicalPath,
			Controller: controllerID,
			Action:     action,
			Label:      label,
		})
	}

	for actionID := range actionLabels {
		add(logicalPrefix+actionID+".html", "default action")
	}

	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil, err
	}
	for _, path := range builtins {
		if strings.HasPrefix(path, logicalPrefix) {
			add(path, "built-in")
		}
	}

	for _, theme := range listTemplateDirThemeNames(templateDir) {
		files, err := ListThemeFiles(templateDir, theme)
		if err != nil {
			continue
		}
		for _, file := range files {
			rel := strings.TrimPrefix(file.Path, theme+"/")
			if !strings.HasPrefix(rel, relPrefix) {
				continue
			}
			source := theme + " theme"
			if theme == themes.BuiltinFrontend {
				source = "theme override"
			}
			if theme == frontendTheme {
				source += " · active"
			}
			add("default/"+rel, source)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out, nil
}

// listTemplateDirThemeNames returns theme folder names under template_dir.
func listTemplateDirThemeNames(templateDir string) []string {
	templateDir = strings.TrimSpace(templateDir)
	if templateDir == "" {
		return nil
	}
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "versions" || name == "extension" {
			continue
		}
		if err := themes.ValidateName(name); err != nil {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
