package templateengine_test

import (
	"fmt"
	"testing"
	"github.com/rob121/cannon/internal/templateengine"
)

func TestListBuiltins(t *testing.T) {
	names, err := templateengine.BuiltinTemplates()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if _, err := templateengine.ReadBuiltin(n); err != nil {
			t.Errorf("read %s: %v", n, err)
		}
	}
	fmt.Println("count", len(names))
}
