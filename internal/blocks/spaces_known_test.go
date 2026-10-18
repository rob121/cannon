package blocks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/templatemeta"
)

func TestExtractSpacesFromSource(t *testing.T) {
	src := `{{space "header"}} {{ space "sidebar" }} {{space ` + "`footer`" + `}} {{space (printf "x-%s" $slug)}}`
	got := ExtractSpacesFromSource(src)
	want := map[string]bool{"header": true, "sidebar": true, "footer": true}
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
	for _, name := range got {
		if !want[name] {
			t.Fatalf("unexpected %q in %#v", name, got)
		}
	}
}

func TestListKnownSpaces(t *testing.T) {
	root := t.TempDir()
	theme := filepath.Join(root, "mysite")
	if err := os.MkdirAll(theme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := templatemeta.Save(theme, templatemeta.PackMeta{Type: templatemeta.TypeFrontend, Status: templatemeta.StatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(theme, "layout.html"), []byte(`{{space "hero"}} {{space "sidebar"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ListKnownSpaces(root, "mysite", []string{"custom-used"})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, name := range got {
		found[name] = true
	}
	for _, want := range []string{"header", "sidebar", "footer", "hero", "custom-used"} {
		if !found[want] {
			t.Fatalf("missing %q in %#v", want, got)
		}
	}
}
