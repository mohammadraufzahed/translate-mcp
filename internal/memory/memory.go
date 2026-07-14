package memory

import (
	"context"
	"time"
)

type Entry struct {
	ID         int64     `json:"id,omitempty"`
	SourceText string    `json:"source_text"`
	TargetText string    `json:"target_text"`
	SourceLang string    `json:"source_language"`
	TargetLang string    `json:"target_language"`
	Domain     string    `json:"domain"`
	Project    string    `json:"project"`
	Client     string    `json:"client"`
	Tags       []string  `json:"tags"`
	Score      float64   `json:"score,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Manager interface {
	Store(ctx context.Context, e Entry) error
	Search(ctx context.Context, query, sourceLang, targetLang string, threshold float64) ([]Entry, error)
}
