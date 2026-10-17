// Package lrucache provides a small bounded LRU map for process-local caches.
package lrucache

import "container/list"

type entry[K comparable, V any] struct {
	key K
	val V
}

// LRU is a fixed-capacity least-recently-used map. Zero value is not usable;
// construct with New.
type LRU[K comparable, V any] struct {
	max   int
	items map[K]*list.Element
	order *list.List // front = most recently used
}

// New returns an LRU that keeps at most max entries. max < 1 means capacity 1.
func New[K comparable, V any](max int) *LRU[K, V] {
	if max < 1 {
		max = 1
	}
	return &LRU[K, V]{
		max:   max,
		items: make(map[K]*list.Element, max),
		order: list.New(),
	}
}

// Get returns the value and promotes it to most-recently used.
func (l *LRU[K, V]) Get(key K) (V, bool) {
	var zero V
	if l == nil {
		return zero, false
	}
	el, ok := l.items[key]
	if !ok {
		return zero, false
	}
	l.order.MoveToFront(el)
	return el.Value.(*entry[K, V]).val, true
}

// Peek returns the value without updating recency.
func (l *LRU[K, V]) Peek(key K) (V, bool) {
	var zero V
	if l == nil {
		return zero, false
	}
	el, ok := l.items[key]
	if !ok {
		return zero, false
	}
	return el.Value.(*entry[K, V]).val, true
}

// Set stores value, evicting the least-recently used entry when over capacity.
func (l *LRU[K, V]) Set(key K, val V) {
	if l == nil {
		return
	}
	if el, ok := l.items[key]; ok {
		el.Value.(*entry[K, V]).val = val
		l.order.MoveToFront(el)
		return
	}
	el := l.order.PushFront(&entry[K, V]{key: key, val: val})
	l.items[key] = el
	for l.order.Len() > l.max {
		back := l.order.Back()
		if back == nil {
			break
		}
		ent := back.Value.(*entry[K, V])
		delete(l.items, ent.key)
		l.order.Remove(back)
	}
}

// Delete removes one key.
func (l *LRU[K, V]) Delete(key K) {
	if l == nil {
		return
	}
	el, ok := l.items[key]
	if !ok {
		return
	}
	delete(l.items, key)
	l.order.Remove(el)
}

// Clear drops every entry.
func (l *LRU[K, V]) Clear() {
	if l == nil {
		return
	}
	l.items = make(map[K]*list.Element, l.max)
	l.order.Init()
}

// Len returns the number of entries.
func (l *LRU[K, V]) Len() int {
	if l == nil {
		return 0
	}
	return l.order.Len()
}

// Cap returns the maximum number of entries.
func (l *LRU[K, V]) Cap() int {
	if l == nil {
		return 0
	}
	return l.max
}

// Keys returns current keys (most-recent first). Snapshot only; safe to Delete while ranging the result.
func (l *LRU[K, V]) Keys() []K {
	if l == nil || l.order.Len() == 0 {
		return nil
	}
	out := make([]K, 0, l.order.Len())
	for el := l.order.Front(); el != nil; el = el.Next() {
		out = append(out, el.Value.(*entry[K, V]).key)
	}
	return out
}
