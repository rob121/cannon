package htmlpolicy

import (
	"strings"
	"testing"
)

func TestSanitizeContentHTMLStripsScript(t *testing.T) {
	in := `<p>Hi</p><script>alert(1)</script>`
	out := SanitizeContentHTML(PolicyStandard, nil, in)
	if strings.Contains(out, "script") {
		t.Fatalf("out = %q", out)
	}
	if !strings.Contains(out, "<p>Hi</p>") {
		t.Fatalf("out = %q", out)
	}
}

func TestSanitizeContentHTMLAllowsYouTubeIframe(t *testing.T) {
	in := `<iframe src="https://www.youtube.com/embed/abc123" width="560" height="315" allowfullscreen></iframe>`
	out := SanitizeContentHTML(PolicyStandard, nil, in)
	if !strings.Contains(out, `src="https://www.youtube.com/embed/abc123"`) {
		t.Fatalf("out = %q", out)
	}
}

func TestSanitizeContentHTMLBlocksUnknownIframe(t *testing.T) {
	in := `<iframe src="https://evil.example/embed"></iframe>`
	out := SanitizeContentHTML(PolicyStandard, nil, in)
	if strings.Contains(out, "evil.example") {
		t.Fatalf("out = %q", out)
	}
}

func TestSanitizeHeadExtraAllowsMetaBlocksScript(t *testing.T) {
	in := `<meta name="theme-color" content="#336699"><script>alert(1)</script>`
	out := SanitizeHeadExtra(in)
	if !strings.Contains(out, `name="theme-color"`) {
		t.Fatalf("out = %q", out)
	}
	if strings.Contains(out, "script") {
		t.Fatalf("out = %q", out)
	}
}

func TestStrictPolicyDropsImage(t *testing.T) {
	in := `<p>x</p><img src="https://example.com/a.png" alt="a">`
	out := SanitizeContentHTML(PolicyStrict, nil, in)
	if strings.Contains(out, "<img") {
		t.Fatalf("out = %q", out)
	}
}
