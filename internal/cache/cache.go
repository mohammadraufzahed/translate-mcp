package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/providers"
)

type Item struct {
	Response  providers.TranslationResponse `json:"response"`
	CreatedAt time.Time                     `json:"created_at"`
}

type Cache interface {
	Get(ctx context.Context, key string) (*Item, bool, error)
	Set(ctx context.Context, key string, item *Item, ttl time.Duration) error
	Close() error
}

type chain struct {
	tiers []Cache
}

func NewChain(tiers ...Cache) Cache {
	var c []Cache
	for _, t := range tiers {
		if t != nil {
			c = append(c, t)
		}
	}
	return &chain{tiers: c}
}

func (c *chain) Get(ctx context.Context, key string) (*Item, bool, error) {
	for i, tier := range c.tiers {
		item, hit, err := tier.Get(ctx, key)
		if err != nil || !hit {
			continue
		}
		for j := 0; j < i; j++ {
			ttl := time.Hour
			_ = c.tiers[j].Set(ctx, key, item, ttl)
		}
		return item, true, nil
	}
	return nil, false, nil
}

func (c *chain) Set(ctx context.Context, key string, item *Item, ttl time.Duration) error {
	for _, tier := range c.tiers {
		if err := tier.Set(ctx, key, item, ttl); err != nil {
			return err
		}
	}
	return nil
}

func (c *chain) Close() error {
	for _, tier := range c.tiers {
		_ = tier.Close()
	}
	return nil
}

func Build(cfg config.CacheConfig, providersEnabled []string) (Cache, error) {
	tiers := make([]Cache, 0, 3)

	l1, err := newMemoryCache(cfg.L1)
	if err != nil {
		return nil, err
	}
	tiers = append(tiers, l1)

	if cfg.L2.Type == "redis" {
		l2, err := newRedisCache(cfg.L2)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, l2)
	}

	if cfg.L3.Type == "sqlite" || cfg.L3.Type == "postgres" {
		l3, err := newSQLCache(cfg.L3)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, l3)
	}

	return NewChain(tiers...), nil
}

func Key(text, sourceLang, targetLang, provider, model, context, tone string, glossaryVersion int64) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%d", text, sourceLang, targetLang, provider, model, context, tone, glossaryVersion)
	return hex.EncodeToString(h.Sum(nil))
}

func SerializeItem(item *Item) ([]byte, error) {
	return json.Marshal(item)
}

func DeserializeItem(data []byte) (*Item, error) {
	var item Item
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func ParseTTL(s string, fallback time.Duration) time.Duration {
	if s == "" || s == "0" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func MaskedText(text string, mask bool) string {
	if !mask {
		return text
	}
	runes := []rune(text)
	if len(runes) <= 8 {
		return "***"
	}
	return string(runes[:3]) + "..." + string(runes[len(runes)-3:])
}
