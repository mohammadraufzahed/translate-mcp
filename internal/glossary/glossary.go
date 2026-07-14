package glossary

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type Entry struct {
	SourceTerm    string `json:"source_term"`
	TargetLang    string `json:"target_language"`
	Translation   string `json:"translation"`
	Context       string `json:"context"`
	CaseSensitive bool   `json:"case_sensitive"`
}

type Glossary struct {
	mu      sync.RWMutex
	entries []Entry
	version int64
}

func New() *Glossary {
	return &Glossary{entries: make([]Entry, 0)}
}

func (g *Glossary) Add(e Entry) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entries = append(g.entries, e)
	g.version++
}

func (g *Glossary) Get(targetLang string) []Entry {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range g.entries {
		if e.TargetLang == "" || e.TargetLang == targetLang {
			out = append(out, e)
		}
	}
	return out
}

func (g *Glossary) Apply(text, targetLang string) (string, map[string]string) {
	entries := g.Get(targetLang)
	if len(entries) == 0 {
		return text, nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].SourceTerm) > len(entries[j].SourceTerm)
	})
	mapping := make(map[string]string)
	masked := text
	for i, e := range entries {
		placeholder := fmt.Sprintf("__TERM_%d__", i)
		re := g.termRegexp(e)
		masked = re.ReplaceAllString(masked, placeholder)
		mapping[placeholder] = e.Translation
	}
	return masked, mapping
}

func (g *Glossary) Restore(translated string, mapping map[string]string) string {
	if mapping == nil {
		return translated
	}
	for placeholder, translation := range mapping {
		translated = strings.ReplaceAll(translated, placeholder, translation)
	}
	return translated
}

func (g *Glossary) Version() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.version
}

func (g *Glossary) termRegexp(e Entry) *regexp.Regexp {
	term := regexp.QuoteMeta(e.SourceTerm)
	if !e.CaseSensitive {
		return regexp.MustCompile(`(?i)\b` + term + `\b`)
	}
	return regexp.MustCompile(`\b` + term + `\b`)
}

func (g *Glossary) Entries() []Entry {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Entry, len(g.entries))
	copy(out, g.entries)
	return out
}
