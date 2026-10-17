package lrucache

import "testing"

func TestLRUEvictsOldest(t *testing.T) {
	l := New[int, string](2)
	l.Set(1, "a")
	l.Set(2, "b")
	l.Set(3, "c")
	if _, ok := l.Get(1); ok {
		t.Fatal("expected key 1 evicted")
	}
	if v, ok := l.Get(2); !ok || v != "b" {
		t.Fatalf("key 2 = %q ok=%v", v, ok)
	}
	if v, ok := l.Get(3); !ok || v != "c" {
		t.Fatalf("key 3 = %q ok=%v", v, ok)
	}
}

func TestLRUGetPromotes(t *testing.T) {
	l := New[int, string](2)
	l.Set(1, "a")
	l.Set(2, "b")
	if _, ok := l.Get(1); !ok {
		t.Fatal("missing 1")
	}
	l.Set(3, "c")
	if _, ok := l.Get(2); ok {
		t.Fatal("expected key 2 evicted after promoting 1")
	}
	if _, ok := l.Get(1); !ok {
		t.Fatal("expected key 1 retained")
	}
}

func TestLRUDeleteClear(t *testing.T) {
	l := New[string, int](3)
	l.Set("x", 1)
	l.Delete("x")
	if l.Len() != 0 {
		t.Fatalf("len=%d", l.Len())
	}
	l.Set("a", 1)
	l.Set("b", 2)
	l.Clear()
	if l.Len() != 0 {
		t.Fatalf("len after clear=%d", l.Len())
	}
}
