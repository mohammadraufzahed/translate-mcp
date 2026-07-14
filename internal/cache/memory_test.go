package cache

import (
	"context"
	"testing"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/providers"
)

func TestMemoryCacheSetAndGet(t *testing.T) {
	ctx := context.Background()
	c, err := newMemoryCache(config.CacheTierConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("newMemoryCache: %v", err)
	}

	resp := providers.TranslationResponse{Translation: "bonjour"}
	if err := c.Set(ctx, "key1", &Item{Response: resp, CreatedAt: time.Now()}, time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, hit, err := c.Get(ctx, "key1")
	if err != nil || !hit || got.Response.Translation != "bonjour" {
		t.Fatalf("Get failed: err=%v hit=%v translation=%q", err, hit, got.Response.Translation)
	}
}

func TestMemoryCacheExpiration(t *testing.T) {
	ctx := context.Background()
	c, err := newMemoryCache(config.CacheTierConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("newMemoryCache: %v", err)
	}

	resp := providers.TranslationResponse{Translation: "hola"}
	_ = c.Set(ctx, "k", &Item{Response: resp, CreatedAt: time.Now()}, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	_, hit, err := c.Get(ctx, "k")
	if err != nil || hit {
		t.Fatalf("expected expired cache miss: err=%v hit=%v", err, hit)
	}
}

func TestMemoryCacheMaxEntries(t *testing.T) {
	ctx := context.Background()
	c, err := newMemoryCache(config.CacheTierConfig{MaxEntries: 2})
	if err != nil {
		t.Fatalf("newMemoryCache: %v", err)
	}

	for i, k := range []string{"a", "b", "c"} {
		resp := providers.TranslationResponse{Translation: k}
		if err := c.Set(ctx, k, &Item{Response: resp, CreatedAt: time.Now()}, time.Hour); err != nil {
			t.Fatalf("Set %d: %v", i, err)
		}
	}
	// Capacity is 2, adding a third may evict one.
	total := 0
	for _, k := range []string{"a", "b", "c"} {
		_, hit, _ := c.Get(ctx, k)
		if hit {
			total++
		}
	}
	if total > 2 {
		t.Errorf("expected at most 2 cached entries, got %d", total)
	}
}
