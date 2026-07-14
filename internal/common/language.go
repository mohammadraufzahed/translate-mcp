package common

import (
	"strings"
)

var SupportedLanguages = map[string]string{
	"en":    "English",
	"es":    "Spanish",
	"fr":    "French",
	"de":    "German",
	"it":    "Italian",
	"pt":    "Portuguese",
	"pt-BR": "Portuguese (Brazil)",
	"nl":    "Dutch",
	"pl":    "Polish",
	"ru":    "Russian",
	"zh":    "Chinese",
	"zh-CN": "Chinese (Simplified)",
	"zh-TW": "Chinese (Traditional)",
	"ja":    "Japanese",
	"ko":    "Korean",
	"ar":    "Arabic",
	"hi":    "Hindi",
	"tr":    "Turkish",
	"vi":    "Vietnamese",
	"th":    "Thai",
	"sv":    "Swedish",
	"da":    "Danish",
	"fi":    "Finnish",
	"no":    "Norwegian",
	"cs":    "Czech",
	"el":    "Greek",
	"he":    "Hebrew",
	"id":    "Indonesian",
	"uk":    "Ukrainian",
	"ro":    "Romanian",
}

func NormalizeLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" || code == "auto" {
		return "auto"
	}
	if _, ok := SupportedLanguages[code]; ok {
		return code
	}
	base, _, _ := strings.Cut(code, "-")
	if _, ok := SupportedLanguages[base]; ok {
		return base
	}
	return code
}

func LanguageName(code string) string {
	name, ok := SupportedLanguages[code]
	if ok {
		return name
	}
	return code
}
