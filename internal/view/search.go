package view

import "caldo/internal/model"

// SearchResultView contains rendered task fields for global search results.
type SearchResultView struct {
	ID          string
	Title       string
	Description string
	ProjectName string
	LabelNames  string
	Attachments []model.Attachment
}
