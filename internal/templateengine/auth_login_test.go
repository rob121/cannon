package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/themes"
)

func testFrontendFuncs() map[string]any {
	funcs := map[string]any{
		"homeURL": func() string { return "/" },
		"controllerURL": func(string, string) string { return "/account/profile" },
		"fieldOptions": func(string) []map[string]string {
			return nil
		},
		"fieldValueContains": func(string, string) bool {
			return false
		},
	}
	for k, v := range lang.TestFuncMap() {
		funcs[k] = v
	}
	return funcs
}

func TestAuthLoginTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/controllers/auth/login.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Login": map[string]any{
			"BlockID":       uint(0),
			"LocalEnabled":  true,
			"LoginAction":   "/auth/login",
			"ShowResetLink": true,
			"ResetURL":      "/account/reset-password",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{"form-control", "btn btn-primary", "Forgot password"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in login page: %s", part, out)
		}
	}
}

func TestAuthProfileTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/controllers/auth/profile.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Account": map[string]any{
			"GivenName":  "Jane",
			"FamilyName": "Doe",
			"Username":   "jane",
		},
		"DisplayName": "Jane Doe",
		"ProfileFields": []map[string]any{
			{"FieldID": uint(1), "Label": "Bio", "Type": "textarea", "Name": "Bio"},
		},
		"ProfileValues": map[uint]string{1: "Writer"},
		"TOTPAllowed":   false,
		"PasskeysAllowed": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{"profile-avatar", "Upload image", "field-1", "Writer"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in profile page: %s", part, out)
		}
	}
}

func TestLoginBlockTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/login.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Login": map[string]any{
			"BlockID":      uint(7),
			"LocalEnabled": true,
			"LoginAction":  "/auth/login",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "site-login-block-7") {
		t.Fatalf("unexpected login block: %s", buf.String())
	}
}
