package cache

import (
	"context"
	"testing"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/providers"
)

func TestKeyConsistency(t *testing.T) {
	k1 := Key("hello", "en", "es", "openai", "gpt-4o", "legal", "formal", 1)
	k2 := Key("hello", "en", "es", "openai", "gpt-4o", "legal", "formal", 1)
	if k1 != k2 {
		t.Errorf("cache key not consistent: %s vs %s", k1, k2)
	}
}

func TestKeyDifferentiatesInputs(t *testing.T) {
	k1 := Key("hello", "en", "es", "openai", "gpt-4o", "", "", 0)
	k2 := Key("hello ", "en", "es", "openai", "gpt-4o", "", "", 0)
	if k1 == k2 {
		t.Error("cache key should differ for different text")
	}
}

func TestSerializeItem(t *testing.T) {
	item := &Item{
		Response: providers.TranslationResponse{
			Translation: "hola",
			SourceLang:  "en",
			TargetLang:  "es",
		},
		CreatedAt: time.Unix(1, 0).UTC(),
	}
	data, err := SerializeItem(item)
	if err != nil {
		t.Fatalf("SerializeItem: %v", err)
	}
	got, err := DeserializeItem(data)
	if err != nil {
		t.Fatalf("DeserializeItem: %v", err)
	}
	if got.Response.Translation != "hola" {
		t.Errorf("deserialized translation = %q", got.Response.Translation)
	}
}

func TestParseTTL(t *testing.T) {
	if got := ParseTTL("1h", time.Minute); got != time.Hour {
		t.Errorf("ParseTTL(1h) = %v", got)
	}
	if got := ParseTTL("", time.Minute); got != time.Minute {
		t.Errorf("ParseTTL(empty) fallback failed: %v", got)
	}
	if got := ParseTTL("invalid", time.Minute); got != time.Minute {
		t.Errorf("ParseTTL(invalid) fallback failed: %v", got)
	}
}

func TestMaskedText(t *testing.T) {
	if got := MaskedText("short", true); got != "***" {
		t.Errorf("short mask = %q", got)
	}
	if got := MaskedText("hello world", false); got != "hello world" {
		t.Errorf("unmasked text changed: %q", got)
	}
}

func TestChainSetAndGet(t *testing.T) {
	ctx := context.Background()
	l1 := newTestMemoryCache(t)
	l2 := newTestMemoryCache(t)
	c := NewChain(l1, l2)

	resp := providers.TranslationResponse{Translation: "hola"}
	if err := c.Set(ctx, "key", &Item{Response: resp, CreatedAt: time.Now()}, time.Hour); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, hit, err := c.Get(ctx, "key")
	if err != nil || !hit {
		t.Fatalf("Get failed: err=%v hit=%v", err, hit)
	}
	if got.Response.Translation != "hola" {
		t.Errorf("got translation %q", got.Response.Translation)
	}
}

func newTestMemoryCache(t *testing.T) Cache {
	c, err := newMemoryCache(config.CacheTierConfig{MaxEntries: 100})
	if err != nil {
		t.Fatalf("newMemoryCache: %v", err)
	}
	return c
}
