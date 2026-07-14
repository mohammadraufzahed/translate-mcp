package providers

import (
	"strings"
	"testing"
)

func TestExtractJSONBlock(t *testing.T) {
	raw := []byte("some text\n```json\n{\"translation\": \"hola\"}\n```\nmore")
	got, err := extractJSON(raw)
	if err != nil {
		t.Fatalf("extractJSON: %v", err)
	}
	if !strings.Contains(string(got), `"translation"`) {
		t.Errorf("extractJSON did not return object: %s", got)
	}
}

func TestExtractJSONNoFences(t *testing.T) {
	raw := []byte("prefix {\"translation\": \"hola\"} suffix")
	got, err := extractJSON(raw)
	if err != nil {
		t.Fatalf("extractJSON: %v", err)
	}
	if string(got) != "{\"translation\": \"hola\"}" {
		t.Errorf("got %q", got)
	}
}

func TestExtractJSONMissing(t *testing.T) {
	_, err := extractJSON([]byte("no json here"))
	if err == nil {
		t.Fatal("expected error for missing JSON")
	}
}

func TestParseLLMResult(t *testing.T) {
	raw := []byte("{\"translation\": \"hola\", \"source_language\": \"en\", \"confidence\": 0.95}")
	resp, err := parseLLMResult(raw, "en", "openai", "gpt-4o")
	if err != nil {
		t.Fatalf("parseLLMResult: %v", err)
	}
	if resp.Translation != "hola" || resp.Provider != "openai" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestParseLLMResultInvalidJSONFallback(t *testing.T) {
	raw := []byte("hola")
	resp, err := parseLLMResult(raw, "en", "openai", "gpt-4o")
	if err != nil {
		t.Fatalf("parseLLMResult: %v", err)
	}
	if resp.Translation != "hola" {
		t.Errorf("fallback translation = %q", resp.Translation)
	}
}

func TestOutputBudget(t *testing.T) {
	if got := outputBudget(""); got != 512 {
		t.Errorf("outputBudget(\"\") = %d, want 512", got)
	}
	// max(512, 5*4+256) -> 512
	if got := outputBudget("hello"); got != 512 {
		t.Errorf("outputBudget(hello) = %d, want 512", got)
	}
	// Enough runes to exceed 512: need (512-256)/4 = 64 runes.
	long := "a"
	for len(long) < 65 {
		long += "a"
	}
	if got := outputBudget(long); got <= 512 {
		t.Errorf("outputBudget(long) = %d, expected > 512", got)
	}
}

func TestCommonLanguageList(t *testing.T) {
	langs := commonLanguageList()
	if len(langs) == 0 {
		t.Fatal("expected language list")
	}
	if langs[0].Code >= langs[len(langs)-1].Code {
		t.Error("language list not sorted")
	}
}
