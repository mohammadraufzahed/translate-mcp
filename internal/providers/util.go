package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/common"
)

var jsonBlockRE = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

func httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{MaxIdleConnsPerHost: 10},
	}
}

func doJSON(ctx context.Context, client *http.Client, method, url string, headers map[string]string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return data, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, resp.StatusCode, nil
}

func extractJSON(raw []byte) ([]byte, error) {
	s := string(raw)
	if m := jsonBlockRE.FindStringSubmatch(s); m != nil {
		return []byte(m[1]), nil
	}
	start := bytes.Index(raw, []byte("{"))
	if start == -1 {
		return nil, fmt.Errorf("no JSON object found")
	}
	end := bytes.LastIndex(raw, []byte("}"))
	if end == -1 || end < start {
		return nil, fmt.Errorf("no JSON object found")
	}
	return raw[start : end+1], nil
}

type llmResult struct {
	Translation  string   `json:"translation"`
	SourceLang   string   `json:"source_language"`
	Confidence   float64  `json:"confidence"`
	Alternatives []string `json:"alternatives"`
}

func parseLLMResult(raw []byte, fallbackSource string, providerName string, model string) (*TranslationResponse, error) {
	jsonBytes, err := extractJSON(raw)
	if err != nil {
		return &TranslationResponse{
			Translation: string(raw),
			SourceLang:  fallbackSource,
			Provider:    providerName,
			Model:       model,
			Confidence:  0.5,
		}, nil
	}
	var r llmResult
	if err := json.Unmarshal(jsonBytes, &r); err != nil {
		return &TranslationResponse{
			Translation: string(raw),
			SourceLang:  fallbackSource,
			Provider:    providerName,
			Model:       model,
			Confidence:  0.5,
		}, nil
	}
	if r.Translation == "" {
		return nil, fmt.Errorf("empty translation")
	}
	return &TranslationResponse{
		Translation:  r.Translation,
		SourceLang:   common.NormalizeLanguage(commonFirst(r.SourceLang, fallbackSource)),
		Provider:     providerName,
		Model:        model,
		Confidence:   math.Max(0, math.Min(1, r.Confidence)),
		Alternatives: r.Alternatives,
	}, nil
}

func outputBudget(text string) int {
	runes := []rune(text)
	return max(512, len(runes)*4+256)
}

func commonFirst(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func commonLanguageList() []Language {
	codes := make([]string, 0, len(common.SupportedLanguages))
	for c := range common.SupportedLanguages {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	langs := make([]Language, 0, len(codes))
	for _, c := range codes {
		langs = append(langs, Language{Code: c, Name: common.SupportedLanguages[c]})
	}
	return langs
}
