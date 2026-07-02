package templateengine

import (
	"bytes"
	"html/template"
	"testing"
)

func TestSprigFuncsAvailable(t *testing.T) {
	funcs := FuncMap(nil, nil)
	if _, ok := funcs["upper"]; !ok {
		t.Fatal("expected sprig upper function")
	}
	if _, ok := funcs["space"]; !ok {
		t.Fatal("expected cannon space function")
	}
	if _, ok := funcs["lenspace"]; !ok {
		t.Fatal("expected cannon lenspace function")
	}
}

func TestSprigUpperInTemplate(t *testing.T) {
	tpl, err := template.New("test").Funcs(FuncMap(nil, nil)).Parse(`{{upper "hello"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "HELLO" {
		t.Fatalf("upper: got %q", buf.String())
	}
}

func TestCannonAddOverridesSprig(t *testing.T) {
	tpl, err := template.New("test").Funcs(FuncMap(nil, nil)).Parse(`{{add 1 2}}`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "3" {
		t.Fatalf("add: got %q", buf.String())
	}
}
