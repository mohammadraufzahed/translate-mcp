package cache

import (
	"context"
	"sync"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
)

type memoryCache struct {
	mu         sync.RWMutex
	items      map[string]*memoryItem
	maxEntries int
}

type memoryItem struct {
	item   *Item
	expire time.Time
}

func newMemoryCache(cfg config.CacheTierConfig) (Cache, error) {
	max := cfg.MaxEntries
	if max <= 0 {
		max = 10000
	}
	mc := &memoryCache{
		items:      make(map[string]*memoryItem),
		maxEntries: max,
	}
	go mc.cleanupLoop()
	return mc, nil
}

func (m *memoryCache) Get(ctx context.Context, key string) (*Item, bool, error) {
	m.mu.RLock()
	it, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || time.Now().After(it.expire) {
		if ok {
			m.mu.Lock()
			delete(m.items, key)
			m.mu.Unlock()
		}
		return nil, false, nil
	}
	return it.item, true, nil
}

func (m *memoryCache) Set(ctx context.Context, key string, item *Item, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Hour
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[key]; !ok && len(m.items) >= m.maxEntries {
		for k := range m.items {
			delete(m.items, k)
			break
		}
	}
	m.items[key] = &memoryItem{item: item, expire: time.Now().Add(ttl)}
	return nil
}

func (m *memoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]*memoryItem)
	return nil
}

func (m *memoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for k, v := range m.items {
			if now.After(v.expire) {
				delete(m.items, k)
			}
		}
		m.mu.Unlock()
	}
}
