package memory

import (
	"context"
	"testing"
)

func TestInMemoryStoreAndSearchExact(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	if err := m.Store(ctx, Entry{SourceText: "hello", TargetText: "hola", SourceLang: "en", TargetLang: "es"}); err != nil {
		t.Fatalf("Store: %v", err)
	}

	results, err := m.Search(ctx, "hello", "en", "es", 0.9)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].TargetText != "hola" {
		t.Fatalf("expected one result, got %+v", results)
	}
}

func TestInMemorySearchFiltering(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	_ = m.Store(ctx, Entry{SourceText: "hello", TargetText: "hola", SourceLang: "en", TargetLang: "es"})
	_ = m.Store(ctx, Entry{SourceText: "hello", TargetText: "bonjour", SourceLang: "en", TargetLang: "fr"})

	results, err := m.Search(ctx, "hello", "en", "fr", 0.9)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].TargetText != "bonjour" {
		t.Fatalf("expected one French result, got %+v", results)
	}
}

func TestInMemorySearchSort(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	_ = m.Store(ctx, Entry{SourceText: "hello world", TargetText: "hola mundo", SourceLang: "en", TargetLang: "es"})
	_ = m.Store(ctx, Entry{SourceText: "hello", TargetText: "hola", SourceLang: "en", TargetLang: "es"})

	results, err := m.Search(ctx, "hello", "en", "es", 0.5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].SourceText != "hello" {
		t.Errorf("expected closest match first, got %q", results[0].SourceText)
	}
}
