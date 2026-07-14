package memory

import (
	"context"
	"strings"
	"sync"
	"time"
)

type inMemory struct {
	mu      sync.RWMutex
	entries []Entry
	nextID  int64
}

func NewInMemory() Manager {
	return &inMemory{entries: make([]Entry, 0)}
}

func (m *inMemory) Store(ctx context.Context, e Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	e.ID = m.nextID
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	m.entries = append(m.entries, e)
	return nil
}

func (m *inMemory) Search(ctx context.Context, query, sourceLang, targetLang string, threshold float64) ([]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]Entry, 0)
	for _, e := range m.entries {
		if sourceLang != "" && e.SourceLang != sourceLang {
			continue
		}
		if targetLang != "" && e.TargetLang != targetLang {
			continue
		}
		score := matchScore(query, strings.ToLower(e.SourceText))
		if score >= threshold {
			e.Score = score
			out = append(out, e)
		}
	}
	sortByScore(out)
	return out, nil
}

func matchScore(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if strings.Contains(b, a) {
		return 0.95
	}
	dist := levenshtein(a, b)
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr := make([]int, len(rb)+1)
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[len(rb)]
}

func sortByScore(list []Entry) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Score > list[i].Score {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}
