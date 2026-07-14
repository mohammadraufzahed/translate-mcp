package common

import (
	"regexp"
	"strings"
	"unicode"
)

var whitespaceRE = regexp.MustCompile(`\s+`)

func NormalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = whitespaceRE.ReplaceAllString(s, " ")
	return s
}

func TruncateText(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

func EstimateTokens(text string) int64 {
	runes := []rune(text)
	return int64(len(runes)/4 + 1)
}

func SplitParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func IsMostlyLatin(s string) bool {
	latin := 0
	total := 0
	for _, r := range s {
		if unicode.Is(unicode.Latin, r) {
			latin++
		}
		if unicode.IsLetter(r) {
			total++
		}
	}
	if total == 0 {
		return true
	}
	return float64(latin)/float64(total) > 0.5
}
