package glossary

import (
	"strings"
	"testing"
)

func TestGlossaryApplyAndRestore(t *testing.T) {
	g := New()
	g.Add(Entry{SourceTerm: "API", TargetLang: "es", Translation: "API", CaseSensitive: true})
	g.Add(Entry{SourceTerm: "server", TargetLang: "es", Translation: "servidor", CaseSensitive: false})

	masked, mapping := g.Apply("The server exposes an API", "es")
	if mapping == nil || len(mapping) == 0 {
		t.Fatal("expected mapping")
	}
	for placeholder, translation := range mapping {
		if !strings.Contains(masked, placeholder) {
			t.Errorf("placeholder %s not in masked text %q", placeholder, masked)
		}
		if translation == "" {
			t.Errorf("empty translation for %s", placeholder)
		}
	}

	got := g.Restore("El __TERM_1__ expone una __TERM_0__", mapping)
	if !strings.Contains(got, "servidor") || !strings.Contains(got, "API") {
		t.Errorf("restore failed: %q", got)
	}
}

func TestGlossaryGet(t *testing.T) {
	g := New()
	g.Add(Entry{SourceTerm: "hello", TargetLang: "es", Translation: "hola"})
	g.Add(Entry{SourceTerm: "world", TargetLang: "fr", Translation: "monde"})
	g.Add(Entry{SourceTerm: "all", Translation: "tout"})

	// Targeted query should include language-specific and global entries.
	if len(g.Get("es")) != 2 {
		t.Errorf("expected 2 Spanish entries (one targeted + one global), got %d", len(g.Get("es")))
	}
	if len(g.Get("")) != 1 {
		t.Errorf("expected 1 entry with no target language, got %d", len(g.Get("")))
	}
	if len(g.Entries()) != 3 {
		t.Errorf("expected 3 total entries, got %d", len(g.Entries()))
	}
}

func TestGlossaryVersion(t *testing.T) {
	g := New()
	if g.Version() != 0 {
		t.Errorf("expected initial version 0, got %d", g.Version())
	}
	g.Add(Entry{})
	if g.Version() != 1 {
		t.Errorf("expected version 1, got %d", g.Version())
	}
}
