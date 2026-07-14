package common

import (
	"strings"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  hello   world  ", "hello world"},
		{"\n\nfoo\tbar\n\n", "foo bar"},
		{"", ""},
	}
	for _, c := range cases {
		got := NormalizeText(c.in)
		if got != c.want {
			t.Errorf("NormalizeText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTruncateText(t *testing.T) {
	in := "hello world"
	if got := TruncateText(in, 100); got != in {
		t.Errorf("TruncateText should not change short text: %q", got)
	}
	got := TruncateText(in, 5)
	want := "hello"
	if got != want {
		t.Errorf("TruncateText(%q, 5) = %q, want %q", in, got, want)
	}
}

func TestEstimateTokens(t *testing.T) {
	text := strings.Repeat("a", 40)
	got := EstimateTokens(text)
	if got != 11 { // 40/4 + 1
		t.Errorf("EstimateTokens(%q) = %d, want 11", text, got)
	}
}

func TestSplitParagraphs(t *testing.T) {
	in := "p1\n\np2\n\n\n\np3"
	out := SplitParagraphs(in)
	if len(out) != 3 || out[0] != "p1" || out[1] != "p2" || out[2] != "p3" {
		t.Errorf("SplitParagraphs failed: %v", out)
	}
}

func TestIsMostlyLatin(t *testing.T) {
	if !IsMostlyLatin("hello") {
		t.Error("expected hello to be mostly latin")
	}
	if IsMostlyLatin("你好") {
		t.Error("expected Chinese to not be mostly latin")
	}
}
