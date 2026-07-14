package common

import "testing"

func TestNormalizeLanguage(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  EN  ", "en"},
		{"EN-US", "en"},
		{"fr-CA", "fr"},
		{"auto", "auto"},
		{"", "auto"},
		{"xx", "xx"},
	}
	for _, c := range cases {
		got := NormalizeLanguage(c.in)
		if got != c.want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLanguageName(t *testing.T) {
	if got := LanguageName("es"); got != "Spanish" {
		t.Errorf("LanguageName(es) = %q, want Spanish", got)
	}
	if got := LanguageName("unknown"); got != "unknown" {
		t.Errorf("LanguageName(unknown) = %q, want unknown", got)
	}
}

func TestSupportedLanguages(t *testing.T) {
	if _, ok := SupportedLanguages["en"]; !ok {
		t.Error("English should be in supported languages")
	}
}
